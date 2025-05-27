package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	kafkas "github.com/PecozQ/aimx-library/kafka"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gofrs/uuid"
	"github.com/segmentio/kafka-go"
)

// StatusService defines the interface for docket status operations
type StatusService interface {
	UpdateDocketStatus(ctx context.Context, docketUUID string, status string) error
}

type statusService struct {
	docketStatusRepo repository.DocketStatusRepositoryService
	logger           log.Logger
}

// NewStatusService creates a new instance of StatusService
func NewStatusService(docketStatusRepo repository.DocketStatusRepositoryService, logger log.Logger) StatusService {
	return &statusService{
		docketStatusRepo: docketStatusRepo,
		logger:           logger,
	}
}

// UpdateDocketStatus updates the status of a docket in the database
func (s *statusService) UpdateDocketStatus(ctx context.Context, docketUUID string, status string) error {
	// Validate status
	if status != "success" && status != "failed" {
		return fmt.Errorf("invalid status: %s. Must be 'success' or 'failed'", status)
	}

	// Map the status to the appropriate value for the database
	dbStatus := "COMPLETED"
	if status == "failed" {
		dbStatus = "FAILED"
	}

	// Parse the UUID
	uuid, err := uuid.FromString(docketUUID)
	if err != nil {
		level.Error(s.logger).Log("method", "UpdateDocketStatus", "err", err, "docketUUID", docketUUID)
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	// Create update request
	updateReq := &dto.UpdateDocketStatusRequest{
		ID:     uuid,
		Status: dbStatus,
	}

	// Update the docket status in the database
	_, err = s.docketStatusRepo.UpdateDocketStatus(ctx, updateReq)
	if err != nil {
		level.Error(s.logger).Log("method", "UpdateDocketStatus", "err", err, "docketUUID", docketUUID)
		return fmt.Errorf("failed to update docket status: %w", err)
	}

	level.Info(s.logger).Log("method", "UpdateDocketStatus", "msg", "Status updated successfully",
		"docketUUID", docketUUID, "status", dbStatus)

	// After successful update, publish a message to get-evaluated-metric topic
	if err := s.publishToGetEvaluatedMetric(docketUUID); err != nil {
		level.Error(s.logger).Log("method", "publishToGetEvaluatedMetric", "err", err, "docketUUID", docketUUID)
		// We don't want to fail the whole operation if publishing fails
		// Just log the error and continue
	}

	return nil
}

// publishToGetEvaluatedMetric publishes a message to the get-evaluated-metric topic
func (s *statusService) publishToGetEvaluatedMetric(docketUUID string) error {
	// Get Kafka broker address from environment variable
	brokerAddress := os.Getenv("KAFKA_INT_BROKER_ADDRESS")
	if brokerAddress == "" {
		brokerAddress = "localhost:9092" // Default if not set
	}

	// Create a Kafka writer for the topic
	topic := "get-evaluated-metric"
	writer := kafkas.GetKafkaWriter(topic, brokerAddress)
	defer writer.Close()

	// Create the message payload
	message := map[string]string{
		"uuid": docketUUID,
	}

	// Marshal the message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send the message to Kafka
	err = writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(docketUUID),
		Value: messageBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	level.Info(s.logger).Log("method", "publishToGetEvaluatedMetric",
		"msg", "Published message to get-evaluated-metric topic",
		"docketUUID", docketUUID)

	return nil
}
