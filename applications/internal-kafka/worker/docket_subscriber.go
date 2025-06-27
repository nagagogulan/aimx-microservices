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

	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/repository"
	kafkas "github.com/PecozQ/aimx-library/kafka"
	"gorm.io/datatypes"
)

var DocketPayloadRepo repository.DocketPayloadRepositoryService

func StartDocketPayloadSubscriber(payloadRepo repository.DocketPayloadRepositoryService) {
	fmt.Println("docket-metrics subscribe start...")
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

			var payloadMsg entities.IncomingDocketPayload

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

func processDocketPayload(ctx context.Context, msg entities.IncomingDocketPayload) error {
	log.Printf("🔍 Saving payload to DB for UUID: %s", msg.UUID)
	fmt.Println("called processDocketPayload")
	// Step 1: Convert msg.Payload (interface{}) to map JSON
	metricsBytes, err := json.Marshal(msg.Metrics)
	if err != nil {
		log.Printf("❌ Failed to marshal metrics: %v", err)
		return err
	}

	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("❌ Failed to marshal payload: %v", err)
		return err
	}

	fmt.Println("✅ ModelConfig to be saved:", string(metricsBytes))
	fmt.Println("✅ ModelConfig to be saved:", string(payloadBytes))

	// Step 2: Extract DatasetName and DocketName from payload JSON
	var payloadStruct struct {
		ModelDatasetUrl []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"modelDatasetUrl"`
		ModelWeightUrl struct {
			Path string `json:"path"`
		} `json:"modelWeightUrl"`
	}

	if err := json.Unmarshal(payloadBytes, &payloadStruct); err != nil {
		log.Printf("❌ Failed to unmarshal payload fields: %v", err)
	}
	var datasetname string
	var docketname string
	if msg.DatasetName != "" {
		datasetname = msg.DatasetName
	}

	if msg.DocketName != "" {
		docketname = msg.DatasetName
	}
	// Step 3: Convert UUID
	googleID := googleuuid.MustParse(msg.UUID)
	gofrsID, err := gofrsuuid.FromBytes(googleID[:])
	if err != nil {
		log.Printf("❌ Failed to convert UUID: %v", err)
		return err
	}
	metricsJSON := datatypes.JSON(metricsBytes)
	payloadJSON := datatypes.JSON(payloadBytes)
	// Step 4: Construct final ModelConfig entity
	entityModel := &entities.ModelConfig{
		ID:          gofrsID,
		Status:      msg.Status,
		DatasetName: datasetname,
		DocketName:  docketname,
		MetricsJSON: metricsJSON,
		PayloadJSON: payloadJSON,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	fmt.Println("✅ ModelConfig to be saved:", entityModel)

	// Step 5: Save to database
	_, err = DocketPayloadRepo.AddDocketDetails(ctx, entityModel)
	if err != nil {
		log.Printf("❌ Failed to save ModelConfig to DB: %v", err)
		return err
	}

	log.Printf("✅ Payload saved successfully for UUID: %s", msg.UUID)
	return nil
}
