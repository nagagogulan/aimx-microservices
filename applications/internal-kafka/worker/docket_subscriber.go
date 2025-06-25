package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	gofrsuuid "github.com/gofrs/uuid"
	googleuuid "github.com/google/uuid"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/repository"
	kafkas "github.com/PecozQ/aimx-library/kafka"
)

var DocketPayloadRepo repository.DocketPayloadRepositoryService

func StartDocketPayloadSubscriber(payloadRepo repository.DocketPayloadRepositoryService) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("❌ Recovered from panic in docket-payload subscriber: %v\n", r)
		}
	}()

	log.Println("📥 Starting docket-payload subscriber...")

	DocketPayloadRepo = payloadRepo

	reader := kafkas.GetKafkaReader(
		"docket-metrics",
		"docket-metrics-group",
		os.Getenv("KAFKA_INT_BROKER_ADDRESS"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		log.Println("🛑 Received shutdown signal, closing docket-payload subscriber...")
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 docket-payload subscriber shutting down...")
			return
		default:
			readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
			m, err := reader.ReadMessage(readCtx)
			readCancel()

			if err != nil {
				if err == context.DeadlineExceeded || err == context.Canceled {
					continue
				}
				log.Printf("❌ Error reading message from docket-payload: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			var payloadMsg dto.IncomingDocketPayload

			if err := json.Unmarshal(m.Value, &payloadMsg); err != nil {
				log.Printf("❌ Error unmarshalling docket-payload message: %v", err)
				continue
			}

			if err := processDocketPayload(ctx, payloadMsg); err != nil {
				log.Printf("❌ Failed to process docket payload: %v", err)
			}
		}
	}
}

func processDocketPayload(ctx context.Context, msg dto.IncomingDocketPayload) error {
	log.Printf("🔍 Saving payload to DB for UUID: %s", msg.UUID)

	// Step 1: Convert msg.Payload (interface{}) to map JSON
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("❌ Failed to marshal payload: %v", err)
		return err
	}

	var dtoModel dto.ModelConfig
	if err := json.Unmarshal(payloadBytes, &dtoModel); err != nil {
		log.Printf("❌ Failed to unmarshal payload to dto.ModelConfig: %v", err)
		return err
	}

	googleID := googleuuid.MustParse(msg.UUID)
	gofrsID, err := gofrsuuid.FromBytes(googleID[:])
	if err != nil {
		log.Printf("❌ Failed to convert UUID: %v", err)
		return err
	}

	// Convert dto to entity
	entityModel := &entities.ModelConfig{
		ID:          gofrsID,
		Status:      dtoModel.Status,
		DatasetName: dtoModel.DatasetName,
		DocketName:  dtoModel.DocketName,
		MetricsJSON: dtoModel.MetricsJSON,
		PayloadJSON: dtoModel.PayloadJSON,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save using repository (expects entity not DTO)
	_, err = DocketPayloadRepo.AddDocketDetails(ctx, entityModel)
	if err != nil {
		log.Printf("❌ Failed to save payload to DB: %v", err)
		return err
	}

	log.Printf("✅ Payload saved successfully for UUID: %s", msg.UUID)
	return nil
}
