package worker

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	kafka "github.com/PecozQ/aimx-library/kafka"
)

var Auditrepo repository.AuditLogsRepositoryService

func StartAuditLogSubscriber(auditrepo repository.AuditLogsRepositoryService) {
	Auditrepo = auditrepo

	// Create Kafka reader
	reader := kafka.GetKafkaReader("audit-logs", "audit-logs-consumer-group", os.Getenv("KAFKA_BROKER_ADDRESS"))
	defer reader.Close()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown on Ctrl+C or SIGINT
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	go func() {
		<-sigchan
		log.Println("Shutdown signal received. Exiting Kafka subscriber...")
		cancel()
	}()

	log.Println("Subscribed to Kafka topic 'audit-logs'")

	for {
		select {
		case <-ctx.Done():
			log.Println("Kafka subscriber shutting down...")
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
				log.Printf("Error reading audit log message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Debug: print raw message
			log.Printf("Raw Kafka message: %s", string(m.Value))

			// Parse the message
			var auditLog dto.AuditLogs
			if err := json.Unmarshal(m.Value, &auditLog); err != nil {
				log.Printf("Error unmarshalling audit log: %v", err)
				continue
			}

			// Store in database
			err = Auditrepo.InsertAuditLog(context.Background(), &auditLog)
			if err != nil {
				log.Printf("Error storing audit log: %v", err)
				continue
			}

			log.Printf("Successfully stored audit log for event: %s", auditLog.Activity)
		}
	}

}
