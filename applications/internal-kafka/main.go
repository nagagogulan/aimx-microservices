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

	"github.com/segmentio/kafka-go"
)

// Message represents the structure of a message received from the topic,
// matching the publisher's format.
type Message struct {
	FileName   string `json:"file_name"` // Changed from Filename, tag to file_name
	Data       []byte `json:"data"`
	ChunkIndex int    `json:"chunk_index"` // Changed from ChunkID, tag to chunk_index
	// IsLastChunk bool `json:"is_last_chunk"` // Removed as per publisher code
}

// FIXME: Need to check and store in the same file path as the external

func messageHandler(msg Message) {
	filePath := "./" + msg.FileName // Use FileName

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening file %s: %v", filePath, err)
		return
	}
	defer f.Close()

	if _, err := f.Write(msg.Data); err != nil {
		log.Printf("Error writing chunk %d to file %s: %v", msg.ChunkIndex, filePath, err) // Use ChunkIndex
		return
	}

	log.Printf("Chunk %d for file %s saved.", msg.ChunkIndex, msg.FileName) // Use ChunkIndex and FileName

	// Since IsLastChunk is removed, we can't definitively log "completely saved" here.
	// The subscriber will continue to append chunks as they arrive.
	// A timeout mechanism or a separate "end-of-file" message from the publisher
	// would be needed to determine when a file is fully received.
}

func main() {
	log.Println("Starting Kafka video chunk subscriber...")

	kafkaBrokerAddress := []string{os.Getenv("KAFKA_BROKER_ADDRESS")}
	topic := "docket-chunks"
	groupID := "docket-chunk-consumer-group"

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        kafkaBrokerAddress,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
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
