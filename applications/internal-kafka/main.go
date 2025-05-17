package main

import (
	"context"
	"encoding/json"

	// "fmt" // Already removed
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PecozQ/aimx-library/kafka"
)

// Message represents the structure of a message received from the topic,
// matching the publisher's format.
type Message struct {
	FileName    string `json:"file_name"` // Changed from Filename, tag to file_name
	Data        []byte `json:"data"`
	ChunkIndex  int    `json:"chunk_index"`   // Changed from ChunkID, tag to chunk_index
	IsLastChunk bool   `json:"is_last_chunk"` // Flag indicating if this is the last chunk of the file
}

// FIXME: Need to check and store in the same file path as the external

func messageHandler(msg Message) bool {
	// Ensure the dockets directory exists
	if err := os.MkdirAll("./dockets", 0755); err != nil {
		log.Printf("Error creating dockets directory: %v", err)
		return false
	}

	filePath := "./dockets/" + msg.FileName

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

func main() {
	log.Println("Starting Kafka video chunk subscriber...")

	topic := "docket-chunks"
	groupID := "docket-chunk-consumer-group"

	r := kafka.GetKafkaReader(topic, groupID, os.Getenv("KAFKA_BROKER_ADDRESS"))
	defer r.Close()

	log.Printf("Subscribed to Kafka topic '%s' with group ID '%s'", topic, groupID)

	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		log.Printf("Received signal: %s. Shutting down...", sig)
		cancel()
	}()

	log.Println("Waiting for messages...")

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
