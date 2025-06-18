package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"os/signal"
	"time"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	kafkas "github.com/PecozQ/aimx-library/kafka"
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

	formDTO, err := FormRepository.UpdateDeactivateStatus(ctx, formObjectID, "READY_FOR_REVIEW")
	if err != nil {
		return nil, errcom.ErrUnabletoUpdate
	}
	if err == nil && formDTO != nil {
		var email string
		for _, field := range formDTO.Fields {
			if field.Label == "Admin Email Address" {
				fmt.Println("Found Admin Email Address field:")
				fmt.Printf("ID: %d, Placeholder: %s, Value: %v\n", field.ID, field.Placeholder, field.Value)

				// Type assertion
				var ok bool
				email, ok = field.Value.(string)
				if !ok {
					return nil, errcom.ErrInvalidEmail
				}
			}
		}

		// If status is a string constant, define it
		status := "READY_FOR_REVIEW"

		// Send email to extracted address
		sendEmail(email, status)
	}

	// Return whatever `docketStatus` is
	return docketStatus, nil
}
func sendEmail(receiverEmail string, status string) error {
	from := "priyadharshini.twilight@gmail.com"
	password := "rotk reak madc kwkf"
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	auth := smtp.PlainAuth("", from, password, smtpHost)
	to := []string{receiverEmail}
	message := []byte{}

	// Properly format the message
	// Properly format the HTML message

	switch status {
	case "READY_FOR_REVIEW":
		message = []byte("From: SingHealth <" + from + ">\r\n" +
			"To: " + receiverEmail + "\r\n" +
			"Subject: Action Required: Your Organization Is Ready for Review\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			"<html>" +
			"<body style='font-family: Arial, sans-serif;'>" +
			"  <div style='background-color: #f4f4f4; padding: 20px;'>" +
			"    <h2 style='color: #2e6c8b;'>📋 Your Organization Is Ready for Review</h2>" +
			"    <p>Dear <strong>" + receiverEmail + "</strong>,</p>" +
			"    <p>Your organization has completed the initial setup and is now <strong>ready for review</strong> by our team.</p>" +
			"    <p>Next steps:</p>" +
			"    <ul>" +
			"      <li><strong>Review</strong> your submitted information for accuracy.</li>" +
			"      <li><strong>Ensure</strong> that all required documents are uploaded.</li>" +
			"      <li><strong>Wait</strong> for our approval notification via email.</li>" +
			"    </ul>" +
			"    <p>We appreciate your cooperation and will get back to you as soon as the review is complete.</p>" +
			"    <p>Best regards,</p>" +
			"    <p><strong>SingHealth Team</strong></p>" +
			"    <p style='color: #888;'>This is an automated email, please do not reply.</p>" +
			"  </div>" +
			"</body>" +
			"</html>")
	}

	// Send the email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return err
	}
	fmt.Println("READY_FOR_REVIEW mail sent successfully")
	return nil
}
