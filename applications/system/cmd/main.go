package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/database/pgsql"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/PecozQ/aimx-library/firebase"
	Kafka "github.com/PecozQ/aimx-library/kafka"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	base "whatsdare.com/fullstack/aimx/backend"
	"whatsdare.com/fullstack/aimx/backend/service"
)

// func init() {
// 	// Get the current working directory (from where the command is run)
// 	dir, err := os.Getwd()
// 	if err != nil {
// 		log.Fatalf("Error getting current working directory: %v", err)
// 		return
// 	}
// 	fmt.Println("Current Working Directory:", dir)

// 	// Construct the path to the .env file in the root directory
// 	envPath := filepath.Join(dir, "../.env")

// 	// Load the .env file from the correct path
// 	err = godotenv.Load(envPath)
// 	if err != nil {
// 		log.Fatalf("Error loading .env file: %v", err)
// 		return
// 	}

// 	// Optionally, you can print a confirmation that the .env file was loaded
// 	fmt.Println("Loaded .env file from:", envPath)
// }

func main() {
	DB, err := pgsql.InitDB(&pgsql.Config{
		// my local host
		DBHost:     "13.229.196.7",
		DBPort:     5432,
		DBUser:     "myappuser",
		DBPassword: "SmartWork@123",
		DBName:     "aimxdb",

		// rds
		// DBHost:     "localhost",
		// DBPort:     5432,
		// DBUser:     "postgres",
		// DBPassword: "password@123",
		// DBName:     "localDb",

		// build dev
		// DBHost:     "localhost",
		// DBPort:     5432,
		// DBUser:     "postgres",
		// DBPassword: "Admin",
		// DBName:     "mylocaldb",
	})
	if err != nil {
		log.Fatalf("Error initializing DB: %v", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Error getting raw DB instance: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping the database: %v", err)
	} else {
		fmt.Println("Database connection successful!")
	}

	if err := pgsql.Migrate(DB); err != nil {
		log.Fatalf("Could not migrate database: %v", err)
	}
	defer sqlDB.Close()
	uri := os.Getenv("MONGO_URI") // replace with your MongoDB URI

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Could not ping to MongoDB: %v", err)
	}

	fmt.Println("Successfully connected to MongoDB!")

	// Get a handle to a collection
	db := client.Database(os.Getenv("MONGO_DBNAME"))
	//collection := db.Collection("templates")

	auditRepo := repository.NewAuditLogsRepositoryService(db)

	firebaseCredentials := map[string]string{
		"FIREBASE_TYPE":                        os.Getenv("FIREBASE_TYPE"),
		"FIREBASE_PROJECT_ID":                  os.Getenv("FIREBASE_PROJECT_ID"),
		"FIREBASE_PRIVATE_KEY_ID":              os.Getenv("FIREBASE_PRIVATE_KEY_ID"),
		"FIREBASE_PRIVATE_KEY":                 os.Getenv("FIREBASE_PRIVATE_KEY"),
		"FIREBASE_CLIENT_EMAIL":                os.Getenv("FIREBASE_CLIENT_EMAIL"),
		"FIREBASE_CLIENT_ID":                   os.Getenv("FIREBASE_CLIENT_ID"),
		"FIREBASE_AUTH_URI":                    os.Getenv("FIREBASE_AUTH_URI"),
		"FIREBASE_TOKEN_URI":                   os.Getenv("FIREBASE_TOKEN_URI"),
		"FIREBASE_AUTH_PROVIDER_X509_CERT_URL": os.Getenv("FIREBASE_AUTH_PROVIDER_X509_CERT_URL"),
		"FIREBASE_CLIENT_X509_CERT_URL":        os.Getenv("FIREBASE_CLIENT_X509_CERT_URL"),
		"FIREBASE_UNIVERSE_DOMAIN":             os.Getenv("FIREBASE_UNIVERSE_DOMAIN"),
	}

	// Initialize Firebase client
	err = firebase.InitializeFirebase(firebaseCredentials)
	if err != nil {
		log.Fatalf("Error initializing Firebase: %v", err)
	}

	notificationRepo := repository.NewNotificationRepo(DB)
	userRepo := repository.NewUserCRUDRepository(DB)

	s := service.NewService(notificationRepo, userRepo, auditRepo)
	endpoints := base.NewEndpoint(s)                // 💡 create Endpoints
	httpHandlers := base.MakeHTTPHandler(endpoints) // ✅ pass Endpoints to HTTP handler

	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(8089),
		Handler: service.CORS(http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`)),
	}

	fmt.Println("Info", "HTTP server started", "port", 8089)
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}

	// Add Kafka consumer for audit logs
	go func() {
		log.Println("Starting audit logs subscriber...")

		// Create a Kafka reader for the audit-logs topic
		reader := Kafka.GetKafkaReader("audit-logs", "audit-logs-consumer-group", os.Getenv("KAFKA_BROKER_ADDRESS"))
		defer reader.Close()

		// Create a context that can be cancelled
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log.Println("Subscribed to Kafka topic 'audit-logs'")

		for {
			// Read message with timeout
			readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
			m, err := reader.ReadMessage(readCtx)
			readCancel()

			if err != nil {
				if err == context.DeadlineExceeded || err == context.Canceled {
					continue
				}
				log.Printf("Error reading audit log message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Process the audit log message
			var auditLog dto.AuditLogs
			if err := json.Unmarshal(m.Value, &auditLog); err != nil {
				log.Printf("Error unmarshalling audit log: %v", err)
				continue
			}

			// Store the audit log in the database
			err = auditRepo.InsertAuditLog(context.Background(), &auditLog)
			if err != nil {
				log.Printf("Error storing audit log: %v", err)
				continue
			}
		}
	}()
}
