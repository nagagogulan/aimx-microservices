package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/PecozQ/aimx-library/domain/dto"
	kafkas "github.com/PecozQ/aimx-library/kafka"
	"github.com/segmentio/kafka-go"
)

// DatasetPathMsg represents the structure of a message received from the sample-dataset-paths topic
type DatasetPathMsg struct {
	Name     string      `json:"name"`
	UUID     string      `json:"uuid"`
	FilePath string      `json:"filepath"`
	FileSize int64       `json:"filesize"`
	FormData dto.FormDTO `json:"formData"` // FormData is used by the dataset_subscriber
}

// StartDatasetPathWorker initializes a Kafka consumer for the sample-dataset-paths topic
func StartDatasetPathWorker() {
	log.Println("Starting dataset path worker....... => ")

	// Create a Kafka reader for the sample-dataset-paths topic
	reader := kafkas.GetKafkaReader("sample-dataset-paths", "dataset-path-consumer-group", os.Getenv("KAFKA_BROKER_ADDRESS")) // This is the internal kafka broker address

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		// We don't need to handle signals here as they're already handled in main.go
		// This is just a placeholder for the context cancellation
		<-sigChan
		log.Println("Received shutdown signal, closing dataset path worker...")
		cancel()
	}()

	// Main loop to process messages
	for {
		select {
		case <-ctx.Done():
			log.Println("Dataset path worker shutting down...")
			return
		default:
			// Read message with timeout
			readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
			m, err := reader.ReadMessage(readCtx)
			readCancel()

			if err != nil {
				if err == context.DeadlineExceeded || err == context.Canceled {
					continue
				}
				log.Printf("Error reading message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Process the message
			var msg DatasetPathMsg
			if err := json.Unmarshal(m.Value, &msg); err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				continue
			}

			// Process the file path
			log.Printf("Processing file path: %s (UUID: %s)", msg.FilePath, msg.UUID)
			if err := processFilePath(ctx, msg); err != nil {
				log.Printf("Error processing file path: %v", err)
			}
		}
	}
}

// processFilePath reads a file and sends it in chunks to the sample-dataset-chunk topic
func processFilePath(ctx context.Context, msg DatasetPathMsg) error {
	// Open the file
	file, err := os.Open(msg.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size to determine when we're at the end
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}
	fileSize := fileInfo.Size()

	// Initialize Kafka writer for the sample-dataset-chunk topic
	writer := kafkas.GetKafkaWriter("sample-dataset-chunk", os.Getenv("KAFKA_BROKER_ADDRESS"))

	// Set up buffered reader and chunk processing
	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024*512) // 500kb chunks
	chunkIndex := 0
	bytesRead := int64(0)

	log.Printf("Starting to chunk file: %s (UUID: %s, Size: %d bytes)",
		msg.FilePath, msg.UUID, fileSize)

	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read error: %w", err)
		}

		if n > 0 {
			bytesRead += int64(n)
			isLastChunk := bytesRead >= fileSize || err == io.EOF

			// Create message according to the required format
			// Include the formData in each chunk message
			chunkMsg := map[string]interface{}{
				"name":          msg.Name,
				"uuid":          msg.UUID,
				"is_last_chunk": isLastChunk,
				"filepath":      msg.FilePath,
				"chunkData":     buffer[:n],
				"chunkIndex":    chunkIndex,
				"formData":      msg.FormData, // Include the form data
			}
			fmt.Println("the chunk msg is given as: ", chunkMsg)

			// Marshal the message to JSON
			chunkData, err := json.Marshal(chunkMsg)
			if err != nil {
				return fmt.Errorf("failed to marshal chunk data: %w", err)
			}

			// Send the message to Kafka
			err = writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(msg.UUID),
				Value: chunkData,
			})
			if err != nil {
				return fmt.Errorf("kafka chunk send error: %w", err)
			}

			log.Printf("Sent chunk %d for file %s (UUID: %s, Last Chunk: %v)",
				chunkIndex, filepath.Base(msg.FilePath), msg.UUID, isLastChunk)

			chunkIndex++
		}

		if err == io.EOF {
			break
		}
	}

	log.Printf("Finished sending %d chunks for file: %s (UUID: %s)",
		chunkIndex, msg.FilePath, msg.UUID)

	return nil
}
