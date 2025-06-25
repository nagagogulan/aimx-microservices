package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"video-subscriber/service" // Use your local service package instead
	"video-subscriber/worker"

	"github.com/PecozQ/aimx-library/database/pgsql"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/PecozQ/aimx-library/kafka"
	kafkas "github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	temporalclient "go.temporal.io/sdk/client" // Use an alias for the temporal client package
)

// Message represents the structure of a message received from the topic,
// matching the publisher's format.
type Message struct {
	FileName    string `json:"file_name"` // Changed from Filename, tag to file_name
	Data        []byte `json:"data"`
	ChunkIndex  int    `json:"chunk_index"`   // Changed from ChunkID, tag to chunk_index
	IsLastChunk bool   `json:"is_last_chunk"` // Flag indicating if this is the last chunk of the file
}

// DocketEvaluationMessage represents the structure of a message for docket evaluation
type DocketEvaluationMessage struct {
	// DocketUUID string                 `json:"docket_uuid"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp string                 `json:"timestamp"`
}

// FIXME: Need to check and store in the same file path as the external

func messageHandler(msg Message) bool {
	// Ensure the dockets directory exists
	if err := os.MkdirAll("./shared/dockets", 0755); err != nil {
		log.Printf("Error creating dockets directory: %v", err)
		return false
	}

	filePath := "./shared/dockets/" + msg.FileName

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening file %s: %v", filePath, err)
		return false
	}
	defer f.Close()

	if _, err := f.Write(msg.Data); err != nil {
		log.Printf("Error writing chunk %d to file %s: %v", msg.ChunkIndex, filePath, err)
		return false
	}

	// Check if this is the last chunk and log accordingly
	if msg.IsLastChunk {

		// Get file size for logging
		fileInfo, err := os.Stat(filePath)
		var fileSize int64
		if err != nil {
			log.Printf("Error getting file stats: %v", err)
			fileSize = -1
		} else {
			fileSize = fileInfo.Size()
		}

		log.Printf("FILE COMPLETE: %s has been completely saved at %s",
			msg.FileName)
		log.Printf("File details: Size: %d bytes",
			fileSize)

		// Here you could add additional processing for completed files
		// For example, move the file to a different location, trigger a notification, etc.

		// Return true to acknowledge this message
		return true
	}

	// For non-last chunks, we can still acknowledge the message
	return true
}

// Function to handle docket evaluation messages
func handleDocketEvaluation(m kafkas.Message, formsService service.FormsService) {
	log.Printf("Received docket evaluation message: Key: %s", string(m.Key))

	var evalMsg DocketEvaluationMessage
	if err := json.Unmarshal(m.Value, &evalMsg); err != nil {
		log.Printf("Error unmarshalling docket evaluation message: %v", err)
		return
	}

	// Log original metadata for debugging
	metadataBytes, _ := json.MarshalIndent(evalMsg.Metadata, "", "  ")
	log.Printf("Original docket metadata: %s", string(metadataBytes))

	// Update metadata with form data
	updatedMetadata, err := formsService.UpdateMetadataWithFormData(evalMsg.Metadata)
	if err != nil {
		log.Printf("Error updating metadata with form data: %v", err)
		// Continue processing with original metadata
	} else {
		// Update the metadata in the evaluation message
		evalMsg.Metadata = updatedMetadata

		// Log the updated metadata
		updatedMetadataBytes, _ := json.MarshalIndent(evalMsg.Metadata, "", "  ")
		log.Printf("Updated docket metadata: %s", string(updatedMetadataBytes))
	}

	// Post the updated metadata to Temporal queue
	err = postToTemporalQueue(evalMsg.Metadata)
	if err != nil {
		log.Printf("Error posting to Temporal queue: %v", err)
	} else {
		log.Printf("Successfully posted metadata to Temporal queue for UUID: %s", evalMsg.Metadata["uuid"])
	}

}

// postToTemporalQueue sends the updated metadata to a Temporal workflow
func postToTemporalQueue(metadata map[string]interface{}) error {
	// Get Temporal client configuration from environment variables
	temporalAddress := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddress == "" {
		// Try to use the hardcoded IP address instead of localhost
		temporalAddress = "54.251.96.179:7233"
		log.Printf("TEMPORAL_ADDRESS not set, using: %s", temporalAddress)
	}

	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "default" // Default namespace
		log.Printf("TEMPORAL_NAMESPACE not set, using default: %s", namespace)
	}

	taskQueue := os.Getenv("TEMPORAL_TASK_QUEUE")
	if taskQueue == "" {
		taskQueue = "evaluation" // Default task queue
		log.Printf("TEMPORAL_TASK_QUEUE not set, using default: %s", taskQueue)
	}

	// Generate a workflow ID using the docket UUID
	workflowID := fmt.Sprintf("workflow-%s", metadata["uuid"])

	log.Printf("Connecting to Temporal server at %s", temporalAddress)

	// Create Temporal client
	c, err := temporalclient.NewClient(temporalclient.Options{
		HostPort:  temporalAddress,
		Namespace: namespace,
	})
	if err != nil {
		return fmt.Errorf("failed to create Temporal client: %w", err)
	}
	defer c.Close()

	// Ensure the UUID is in the metadata
	// metadata["uuid"] = docketUUID

	// Log the payload being sent to Temporal
	payloadBytes, _ := json.MarshalIndent(metadata, "", "  ")
	log.Printf("Sending payload to Temporal workflow:\n%s", string(payloadBytes))

	// Prepare workflow options
	options := temporalclient.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
	}

	// Start the workflow
	we, err := c.ExecuteWorkflow(context.Background(), options, "runEval", metadata)
	if err != nil {
		return fmt.Errorf("failed to execute workflow: %w", err)
	}

	log.Printf("Started workflow execution. WorkflowID: %s, RunID: %s", we.GetID(), we.GetRunID())
	return nil
}

func main() {
	log.Println("Starting Kafka video chunk subscriber...")

	// Initialize MongoDB connection for form repository
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		// Default MongoDB URI if environment variable is not set
		mongoURI = "mongodb://13.229.196.7:27017"
		log.Printf("MONGO_URI not set, using default: %s", mongoURI)
	}

	mongoDBName := os.Getenv("MONGO_DBNAME")
	if mongoDBName == "" {
		mongoDBName = "mydb"
		log.Printf("MONGO_DBNAME not set, using default: %s", mongoDBName)
	}

	log.Printf("Connecting to MongoDB at %s, database: %s", mongoURI, mongoDBName)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}

	// Ping the database
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("Could not ping to MongoDB: %v", err)
	}

	fmt.Println("Successfully connected to MongoDB!")

	dbPortStr := os.Getenv("DBPORT")
	if dbPortStr == "" {
		log.Println("⚠️ DBPORT not set, defaulting to 5432")
		dbPortStr = "5432"
	}

	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		log.Fatalf("Invalid DBPORT value: %v", err)
	}
	DB, err := pgsql.InitDB(&pgsql.Config{
		DBHost:     os.Getenv("DBHOST"),
		DBPort:     dbPort,
		DBUser:     os.Getenv("DBUSER"),
		DBPassword: os.Getenv("DBPASSWORD"),
		DBName:     os.Getenv("DBNAME"),
	})
	if err != nil {
		log.Fatalf("Error initializing PostgreSQL DB: %v", err)
	}

	// Ping to verify connection
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Error getting raw SQL DB instance: %v", err)
	}

	if err = sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping PostgreSQL database: %v", err)
	} else {
		log.Println("✅ PostgreSQL database connection successful!")
	}

	// Get a handle to the database
	db := mongoClient.Database(mongoDBName)
	formRepo := repository.NewFormRepository(db)
	docketPayloadRepo := repository.NewDocketPayloadRepositoryService(DB)
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()
	// Initialize the forms service
	formsService := service.NewFormsService(formRepo)

	// Get Kafka broker address from environment variable
	kafkaBrokerAddress := os.Getenv("KAFKA_EXT_BROKER_ADDRESS")
	if kafkaBrokerAddress == "" {
		kafkaBrokerAddress = "13.229.196.7:9092" // Use the same IP as MongoDB
		log.Printf("KAFKA_BROKER_ADDRESS not set, using default: %s", kafkaBrokerAddress)
	}

	topic := "docket-chunks"
	groupID := "docket-chunk-consumer-group"

	r := kafka.GetKafkaReader(topic, groupID, kafkaBrokerAddress)
	defer r.Close()

	// Create a reader for the docket evaluation topic
	evalTopic := "send-docket-for-evaluation"
	evalGroupID := "docket-evaluation-consumer-group"

	evalReader := kafka.GetKafkaReader(evalTopic, evalGroupID, kafkaBrokerAddress)
	defer evalReader.Close()

	log.Printf("Subscribed to Kafka topics: '%s' with group ID '%s' and '%s' with group ID '%s' at broker %s",
		topic, groupID, evalTopic, evalGroupID, kafkaBrokerAddress)

	ctx, cancel = context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		log.Printf("Received signal: %s. Shutting down...", sig)
		cancel()
	}()

	// Start a goroutine to handle docket evaluation messages
	go func() {
		log.Println("Starting docket evaluation message consumer...")
		for {
			select {
			case <-ctx.Done():
				log.Println("Context cancelled, exiting docket evaluation consumer.")
				return
			default:
				readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
				m, err := evalReader.ReadMessage(readCtx)
				readCancel()

				if err != nil {
					if err == context.DeadlineExceeded {
						continue
					}
					if err == context.Canceled {
						log.Println("ReadMessage context cancelled for evaluation consumer.")
						return
					}
					log.Printf("Error reading evaluation message from Kafka: %v", err)
					time.Sleep(1 * time.Second)
					continue
				}

				// Handle the docket evaluation message with the forms service
				handleDocketEvaluation(m, formsService)
			}
		}
	}()

	log.Println("Waiting for messages...")

	go worker.StartDocketPayloadSubscriber(docketPayloadRepo)

	// Initialize Temporal client for workflow monitoring
	temporalAddress := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddress == "" {
		temporalAddress = "54.251.96.179:7233" // Use the IP address instead of localhost
		log.Printf("TEMPORAL_ADDRESS not set, using: %s", temporalAddress)
	}

	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "default" // Default namespace
		log.Printf("TEMPORAL_NAMESPACE not set, using default: %s", namespace)
	}

	log.Printf("Initializing Temporal client at %s", temporalAddress)

	// Create Temporal client
	temporalClient, err := temporalclient.NewClient(temporalclient.Options{
		HostPort:  temporalAddress,
		Namespace: namespace,
	})
	if err != nil {
		log.Printf("Warning: Failed to create Temporal client: %v", err)
		log.Println("Continuing without Temporal client...")
	} else {
		defer temporalClient.Close()
		log.Println("Successfully connected to Temporal server")
	}

	// Start the metric worker in a goroutine
	// go func() {
	// 	log.Println("Starting metric worker...")
	// 	worker.StartMetricWorker()
	// }()

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, exiting message loop.")
			return
		default:
			readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
			m, err := r.ReadMessage(readCtx)
			readCancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					continue
				}
				if err == context.Canceled {
					log.Println("ReadMessage context cancelled, likely shutting down.")
					return
				}
				log.Printf("Error reading message from Kafka: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			log.Printf("Message received from Kafka: Topic %s, Partition %d, Offset %d, Key: %s",
				m.Topic, m.Partition, m.Offset, string(m.Key))

			var msgData Message
			if err := json.Unmarshal(m.Value, &msgData); err != nil {
				log.Printf("Error unmarshalling message value: %v. Message: %s", err, string(m.Value))
				continue
			}

			messageHandler(msgData)
		}
	}
}
