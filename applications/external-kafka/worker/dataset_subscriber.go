package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	kafkas "github.com/PecozQ/aimx-library/kafka"
	"github.com/gofrs/uuid"
)

// DatasetChunkMsg represents the structure of a message received from the sample-dataset-chunk topic
type DatasetChunkMsg struct {
	Name        string      `json:"name"`
	UUID        string      `json:"uuid"`
	IsLastChunk bool        `json:"is_last_chunk"`
	FilePath    string      `json:"filepath"`
	ChunkData   []byte      `json:"chunkData"`
	ChunkIndex  int         `json:"chunkIndex"`
	FormData    dto.FormDTO `json:"formData"`
	UserName    string      `json:"userName"`
	UserId      string      `json:"userId"`
}

// FileAssembler keeps track of chunks for a specific file
type FileAssembler struct {
	Name       string
	UUID       string
	OutputPath string
	ChunkCount int
	Complete   bool
	LastUpdate time.Time
	mu         sync.Mutex
}

// Map to track file assemblers by UUID
var fileAssemblers = make(map[string]*FileAssembler)
var assemblersMutex sync.Mutex

// FormRepo holds the form repository service
var FormRepo repository.FormRepositoryService

// SampleDatasetRepo holds the sample dataset repository service
var SampleDatasetRepo repository.SampleDatasetRepositoryService

var userRepo repository.UserCRUDService

var rolerepo repository.RoleRepositoryService

// StartDatasetChunkSubscriber initializes a Kafka consumer for the sample-dataset-chunk topic
func StartDatasetChunkSubscriber(formRepo repository.FormRepositoryService, sampleDatasetRepo repository.SampleDatasetRepositoryService, usersRepo repository.UserCRUDService, rolerepo repository.RoleRepositoryService) {
	// Set the form repository
	FormRepo = formRepo
	// Set the sample dataset repository
	SampleDatasetRepo = sampleDatasetRepo
	userRepo = usersRepo

	log.Println("Starting dataset chunk subscriber...")

	// Create a Kafka reader for the sample-dataset-chunk topic
	reader := kafkas.GetKafkaReader("sample-dataset-chunk", "dataset-chunk-consumer-group", os.Getenv("KAFKA_INT_BROKER_ADDRESS"))

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		// We don't need to handle signals here as they're already handled in main.go
		// This is just a placeholder for the context cancellation
		<-sigChan
		log.Println("Received shutdown signal, closing dataset chunk subscriber...")
		cancel()
	}()

	// Start a goroutine to clean up stale file assemblers
	go cleanupStaleAssemblers(ctx)

	// Create default output directory if it doesn't exist
	// This will only be used if the FilePath in the message is empty
	outputDir := "./datasets"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("Error creating default output directory: %v", err)
		return
	}

	// Main loop to process messages
	for {
		select {
		case <-ctx.Done():
			log.Println("Dataset chunk subscriber shutting down...")
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
			var msg DatasetChunkMsg
			if err := json.Unmarshal(m.Value, &msg); err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				continue
			}

			// Process the chunk
			processChunk(msg, outputDir)
		}
	}
}

// processChunk handles a single chunk message
func processChunk(msg DatasetChunkMsg, outputDir string) {
	assemblersMutex.Lock()
	assembler, exists := fileAssemblers[msg.UUID]
	if !exists {
		// Use the FilePath from the message exactly as is, including file name and extension
		outputPath := msg.FilePath
		if outputPath == "" {
			// If no FilePath is provided, use the default naming convention in the default directory
			outputPath = filepath.Join(outputDir, fmt.Sprintf("%s_%s", msg.Name, msg.UUID))
			log.Printf("No FilePath provided in message, using default path: %s", outputPath)
		} else {
			log.Printf("Using exact FilePath from message: %s", outputPath)
		}

		assembler = &FileAssembler{
			Name:       msg.Name,
			UUID:       msg.UUID,
			OutputPath: outputPath,
			LastUpdate: time.Now(),
		}
		fileAssemblers[msg.UUID] = assembler
		log.Printf("Started receiving chunks for new file: %s (UUID: %s)", msg.Name, msg.UUID)
	}
	assemblersMutex.Unlock()

	// Lock this specific assembler for thread safety
	assembler.mu.Lock()
	defer assembler.mu.Unlock()

	// Update the last update time
	assembler.LastUpdate = time.Now()

	// Ensure the directory structure exists
	// Extract the directory part from the file path
	dirPath := filepath.Dir(assembler.OutputPath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		log.Printf("Error creating directory structure for %s: %v", dirPath, err)
		return
	}
	log.Printf("Directory structure created/verified for path: %s", dirPath)

	// Open the output file in append mode
	file, err := os.OpenFile(assembler.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Error opening output file %s: %v", assembler.OutputPath, err)
		return
	}
	defer file.Close()

	// Write the chunk data to the file
	if _, err := file.Write(msg.ChunkData); err != nil {
		log.Printf("Error writing chunk %d to file %s: %v", msg.ChunkIndex, assembler.OutputPath, err)
		return
	}

	// Increment the chunk count
	assembler.ChunkCount++

	log.Printf("Processed chunk %d for file %s (UUID: %s)", msg.ChunkIndex, msg.Name, msg.UUID)

	// If this is the last chunk, mark the file as complete and create the form directly
	if msg.IsLastChunk {
		assembler.Complete = true
		log.Printf("✅ File assembly complete: %s (UUID: %s), Total chunks: %d",
			msg.Name, msg.UUID, assembler.ChunkCount)
		log.Printf("File saved at: %s", assembler.OutputPath)

		// Get file size for logging
		fileInfo, err := os.Stat(assembler.OutputPath)
		if err == nil {
			log.Printf("File size: %d bytes", fileInfo.Size())
		}
		// If formData is present and FormRepo is set, create the form directly
		if len(msg.FormData.Fields) > 0 && FormRepo != nil {
			// Update form data with file path
			// First, check if we need to create new fields or update existing ones
			// hasFilePathField := false

			// // Check existing fields
			// for i, field := range msg.FormData.Fields {
			// 	if field.Label == "Dataset File Path" {
			// 		msg.FormData.Fields[i].Value = assembler.OutputPath
			// 		hasFilePathField = true
			// 		break
			// 	}
			// }

			// // Add file path field if it doesn't exist
			// if !hasFilePathField {
			// 	msg.FormData.Fields = append(msg.FormData.Fields, dto.FieldDTO{
			// 		ID:    getNextFieldID(msg.FormData.Fields),
			// 		Label: "Dataset File Path",
			// 		Value: assembler.OutputPath,
			// 	})
			// }

			// // Set form type if not already set
			// if msg.FormData.Type == 0 {
			// 	msg.FormData.Type = 2 // Assuming 2 is the type for dataset forms
			// }

			// Create the form
			fmt.Println("check msg in value", msg)
			log.Printf("Creating form for dataset: %s (UUID: %s)", msg.Name, msg.UUID)
			createdForm, err := FormRepo.CreateForm(context.Background(), msg.FormData)
			if err != nil {
				log.Printf("Error creating form: %v", err)
			}
			fmt.Println("check msg in value", createdForm)
			fmt.Println("user id get in msg", msg.UserId)
			if createdForm != nil && msg.UserId != "" {
				fmt.Println("start send audit logs")
				var audit dto.AuditLogs
				//var email string
				id, err := uuid.FromString(msg.UserId)
				if err != nil {
					log.Printf("Invalid UserID format: %v", err)
				}

				user, err := userRepo.GetUserByID(context.Background(), id)
				if err != nil {
					log.Printf("Get user: %v", err)
				}
				res, err := rolerepo.GetRoleByID(context.Background(), user.Role.ID)
				if err != nil {
					log.Printf("Get user: %v", err)
				}

				if createdForm.Type == 2 {
					audit = dto.AuditLogs{
						Timestamp: time.Now().UTC(),
						UserName:  msg.UserName,
						UserID:    msg.UserId,
						Activity:  "Created Dataset",
						Dataset:   msg.Name,
						UserRole:  res.Name,
						Details: map[string]string{
							"form_id":   createdForm.ID.String(),
							"form_type": fmt.Sprintf("%d", createdForm.Type),
						},
					}
					go kafkas.PublishAuditLog(&audit, os.Getenv("KAFKA_BROKER_ADDRESS"), "audit-logs")
				}
			}

			log.Printf("Successfully created form for dataset: %s (UUID: %s, Form ID: %s)",
				msg.Name, msg.UUID, createdForm.ID.Hex())

			// Store values in the SampleDataset table
			if SampleDatasetRepo != nil {
				log.Printf("Creating sample dataset entry for: %s (UUID: %s)", msg.Name, msg.UUID)

				// Create a map with the required fields
				sampleDataset := &dto.CreateSampleDatasetRequest{
					Name:    msg.Name,
					IntUUID: msg.UUID,
					ExtUUID: createdForm.ID.Hex(),
				}
				_, err := SampleDatasetRepo.CreateSampleDataset(context.Background(), sampleDataset)
				if err != nil {
					log.Printf("Error creating sample dataset entry: %v", err)
				} else {
					log.Printf("Successfully created sample dataset entry for: %s (UUID: %s)", msg.Name, msg.UUID)
				}
			} else {
				log.Printf("Sample dataset repository not initialized for %s (UUID: %s), skipping sample dataset creation",
					msg.Name, msg.UUID)
			}

		} else {
			if msg.FormData.Type == 0 && len(msg.FormData.Fields) == 0 {
				log.Printf("No form data found for %s (UUID: %s), skipping form creation",
					msg.Name, msg.UUID)
			} else if FormRepo == nil {
				log.Printf("Form repository not initialized for %s (UUID: %s), skipping form creation",
					msg.Name, msg.UUID)
			}
		}

		// Optionally, remove the assembler from the map
		// assemblersMutex.Lock()
		// delete(fileAssemblers, msg.UUID)
		// assemblersMutex.Unlock()
	}
}

// cleanupStaleAssemblers periodically removes file assemblers that haven't been updated recently
func cleanupStaleAssemblers(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			assemblersMutex.Lock()

			for uuid, assembler := range fileAssemblers {
				// If the assembler hasn't been updated in 30 minutes and is not complete, consider it stale
				if now.Sub(assembler.LastUpdate) > 30*time.Minute && !assembler.Complete {
					log.Printf("Removing stale file assembler for UUID: %s", uuid)
					delete(fileAssemblers, uuid)
				}
			}

			assemblersMutex.Unlock()
		}
	}
}
