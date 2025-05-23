package worker

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "time"

    "whatsdare.com/fullstack/aimx/backend/model"
    "whatsdare.com/fullstack/aimx/backend/service"

    kafkas "github.com/PecozQ/aimx-library/kafka"
)

// StartDocketStatusWorker starts a worker to consume docket status messages
func StartDocketStatusWorker(statusService service.StatusService) {
    topic := "docket-status"
    groupID := "docket-status-consumer-group"
    brokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
    
    if brokerAddress == "" {
        brokerAddress = "localhost:9092" // Default if not set
    }

    reader := kafkas.GetKafkaReader(topic, groupID, brokerAddress)
    defer reader.Close()

    log.Printf("Started docket status consumer. Listening on topic: %s with group ID: %s", topic, groupID)

    for {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        m, err := reader.ReadMessage(ctx)
        cancel()

        if err != nil {
            if err == context.DeadlineExceeded {
                continue
            }
            log.Printf("Error reading message from Kafka: %v", err)
            time.Sleep(1 * time.Second)
            continue
        }

        log.Printf("Received docket status message: Key: %s", string(m.Key))

        var statusMsg model.DocketStatusMessage
        if err := json.Unmarshal(m.Value, &statusMsg); err != nil {
            log.Printf("Error unmarshalling docket status message: %v", err)
            continue
        }

        log.Printf("Processing docket status update for UUID: %s, Status: %s", 
            statusMsg.UUID, statusMsg.Status)

        // Update the docket status in the database
        err = statusService.UpdateDocketStatus(context.Background(), statusMsg.UUID, statusMsg.Status)
        if err != nil {
            log.Printf("Error updating docket status: %v", err)
            continue
        }

        log.Printf("Successfully updated status for docket UUID: %s to %s", 
            statusMsg.UUID, statusMsg.Status)
    }
}