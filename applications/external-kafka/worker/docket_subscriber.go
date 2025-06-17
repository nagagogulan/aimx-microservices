package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	kafkas "github.com/PecozQ/aimx-library/kafka"
	errcom "github.com/PecozQ/aimx-library/apperrors"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/gofrs/uuid"
)

var DocketMetricRepo repository.DocketMetricsRepository
var DocketStatusRepo repository.DocketStatusRepositoryService
var FormRepository repository.FormRepositoryService

func StartDocketStatusResultSubscriber(
	docketMetricsRepo repository.DocketMetricsRepository,
	docketStatusRepo repository.DocketStatusRepositoryService,
	formRepo repository.FormRepositoryService) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("❌ Recovered from panic in docket-status-result subscriber: %v", r)
		}
	}()

	log.Println("📥 Starting docket-status-result subscriber...")

	DocketMetricRepo = docketMetricsRepo
	DocketStatusRepo = docketStatusRepo
	FormRepository = formRepo

	reader := kafkas.GetKafkaReader(
		"docket-status",
		"docket-status-group",
		os.Getenv("KAFKA_INT_BROKER_ADDRESS"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, closing docket-status-result subscriber...")
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 docket-status-result subscriber shutting down...")
			return
		default:
			readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
			m, err := reader.ReadMessage(readCtx)
			readCancel()

			if err != nil {
				if err == context.DeadlineExceeded || err == context.Canceled {
					continue
				}
				log.Printf("❌ Error reading message from docket-status-result: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			var msg struct {
				UUID    string      `json:"uuid"`
				Status  string      `json:"status"`
				Metrics interface{} `json:"metrics"`
			}

			if err := json.Unmarshal(m.Value, &msg); err != nil {
				log.Printf("❌ Error unmarshalling docket-status-result message: %v", err)
				continue
			}

			log.Printf("✅ Docket Status Update Received:\n  UUID    = %s\n  Status  = %s\n  Metrics = %+v",
				msg.UUID, msg.Status, msg.Metrics)

			// Process the message
			if _, err := processDocketStatus(ctx, msg.UUID, msg.Status, msg.Metrics); err != nil {
				log.Printf("❌ Failed to process docket status: %v", err)
			}
		}
	}
}

func processDocketStatus(ctx context.Context, uuidStr string, status string, metrics interface{}) (*dto.DocketStatusResponse, error) {
	log.Printf("🔔 Processing DocketStatus update: UUID=%s, Status=%s", uuidStr, status)

	var metricHexID string

	// Step 1: Save metrics if status is success
	if status == "success" {
		newMetric := &dto.DocketMetricsDTO{
			DocketStatusID: uuidStr,
			Metadata:       metrics,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		savedMetric, err := DocketMetricRepo.Create(ctx, *newMetric)
		if err != nil {
			log.Printf("❌ Failed to create DocketMetrics: %v", err)
			return nil, err
		}

		log.Printf("✅ DocketMetrics created with ID: %s", savedMetric.ID.Hex())
		metricHexID = savedMetric.ID.Hex()
	}

	// Step 2: Convert UUID string to uuid.UUID
	parsedUUID, err := uuid.FromString(uuidStr)
	if err != nil {
		log.Printf("❌ Invalid UUID format: %v", err)
		return nil, err
	}

	// Step 3: Prepare update request
	updateReq := &dto.UpdateDocketStatusRequest{
		ID:              parsedUUID,
		Status:          status,
		DocketMetricsId: metricHexID, // Empty string if not "success"
	}

	// Step 4: Perform update
	docketStatus, err := DocketStatusRepo.UpdateDocketStatus(ctx, updateReq)
	if err != nil {
		log.Printf("❌ Failed to update DocketStatus: %v", err)
		return nil, err
	}

	formObjectID, err := primitive.ObjectIDFromHex(docketStatus.DocketId)
	if err != nil {
		log.Printf("❌ Invalid ObjectID: %v", err)
		return nil, fmt.Errorf("invalid ObjectID for form: %v", err)
	}

	err = FormRepository.UpdateDeactivateStatus(ctx, formObjectID, "READY_FOR_REVIEW")
	if err != nil {
		return nil, errcom.ErrUnabletoUpdate
	}

	return docketStatus, nil
}
