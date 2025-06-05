package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	kafkas "github.com/PecozQ/aimx-library/kafka"
	"github.com/gofrs/uuid"
	"github.com/segmentio/kafka-go"
	"whatsdare.com/fullstack/aimx/backend/model"
)

type Service interface {
	//UploadFile(ctx context.Context, filePath string) (string, error)
	UploadFile(ctx context.Context, req model.UploadRequest) (*model.UploadResponse, error)
	GetFile(ctx context.Context, filePath string) ([]byte, string, error)
	UploadDataset(ctx context.Context, fileHeader *multipart.FileHeader) (dto.DatasetUploadResponse, error)
	//GetFileList(ctx context.Context) ([]string, error)
	DeleteFile(ctx context.Context, filepath model.DeleteFileRequest) error
	OpenFile(ctx context.Context, filePath string) (*os.File, error)
	ChunkFileToKafka(ctx context.Context, req dto.ChunkFileRequest) (*dto.ChunkFileResponse, error)
	GetAllSampleDatasets(ctx context.Context) ([]dto.SampleDatasetResponse, error)
	TestKong(ctx context.Context) (map[string]string, error)
}

type fileService struct {
	sampleDatasetRepo repository.SampleDatasetRepositoryService
}

func NewService(sampleDatasetRepo repository.SampleDatasetRepositoryService) Service {
	return &fileService{
		sampleDatasetRepo: sampleDatasetRepo,
	}
}

// UploadDataset handles saving the uploaded dataset file using streaming.
func (s *fileService) UploadDataset(ctx context.Context, fileHeader *multipart.FileHeader) (dto.DatasetUploadResponse, error) {
	// Open the uploaded file
	src, err := fileHeader.Open()
	if err != nil {
		return dto.DatasetUploadResponse{}, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Generate a 16-digit UUID
	id, err := uuid.NewV4()
	if err != nil {
		return dto.DatasetUploadResponse{}, fmt.Errorf("failed to generate UUID: %w", err)
	}

	uuidStr := id.String()

	// Create the directory structure: /dataset/{{uuid}}/sample/
	dirPath := filepath.Join("shared", "datasets", uuidStr, "sample")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return dto.DatasetUploadResponse{}, fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Use the original filename from the upload
	fileName := fileHeader.Filename

	// Full path to save the file
	fullPath := filepath.Join(dirPath, fileName)

	// Create the destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return dto.DatasetUploadResponse{}, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Stream the file content to the destination
	written, err := io.Copy(dst, src)
	if err != nil {
		// Attempt to remove partially written file
		os.Remove(fullPath)
		return dto.DatasetUploadResponse{}, fmt.Errorf("failed to save file: %w", err)
	}

	// Return the response with the stored path
	return dto.DatasetUploadResponse{
		Message:  "Dataset uploaded successfully",
		FileName: fileName,
		FilePath: fullPath,
		FileSize: written,
	}, nil
}

func (s *fileService) UploadFile(ctx context.Context, req model.UploadRequest) (*model.UploadResponse, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID")
	}

	enumLabel := common.ValueMapper(req.FormType, "FileFormat", "ENUM_TO_HASH")
	timestamp := time.Now().Format("20060102_150405")

	validDatasetExtensions := map[string]bool{"csv": true, "xlsx": true, "zip": true}
	validImageExtensions := map[string]bool{"jpg": true, "jpeg": true, "png": true}
	validFileFormats := map[string]bool{"docx": true, "pdf": true}
	validDocketFileFormats := map[string]bool{"pkl": true, "joblib": true, "pth": true, "h5": true, "onnx": true}

	ext := strings.ToLower(req.Extension)
	var filePath string

	// File extension validation based on file type
	// baseDir, err := os.Getwd()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get working directory: %w", err)
	// }

	// Prepare relative folder path based on form type
	switch enumLabel {
	case 0:
		if !validDatasetExtensions[ext] {
			return nil, errcom.ErrInvalidDatasetFileExtension
		}
		filePath = fmt.Sprintf("datasetfile/%s/%s_%s", id.String(), timestamp, id.String())
	case 1:
		if !validImageExtensions[ext] {
			return nil, errcom.ErrInvalidImageFileExtension
		}
		filePath = fmt.Sprintf("images/%s/%s_%s", id.String(), timestamp, id.String())
	case 2:
		if !validFileFormats[ext] {
			return nil, errcom.ErrInvalidDocumentFileFormat
		}
		filePath = fmt.Sprintf("file/%s/%s_%s", id.String(), timestamp, id.String())
	case 3:
		if !validDocketFileFormats[ext] {
			return nil, errcom.ErrInvalidDocketFormat
		}
		filePath = fmt.Sprintf("docketfile/%s/%s_%s", id.String(), timestamp, id.String())
	default:
		return nil, errcom.ErrUnsupportedFileFormat
	}

	// Combine with base directory and "data"
	fullDir := filepath.Join("shared", filePath)

	// Create all required directories
	if err := os.MkdirAll(fullDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to create upload directory: %w", err)
	}

	// Construct full file name and path
	newFileName := fmt.Sprintf("%s_%s.%s", timestamp, id.String(), ext)
	fullPath := filepath.Join(fullDir, newFileName)

	// Create destination file
	outFile, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create file")
	}
	defer outFile.Close()

	// Stream the file content to the new file
	contentReader := bytes.NewReader(req.Content)
	if _, err := io.Copy(outFile, contentReader); err != nil {
		return nil, fmt.Errorf("failed to stream file")
	}
	normalizedPath := strings.ReplaceAll(fullPath, "\\", "/")
	// Return response
	return &model.UploadResponse{
		ID:       id,
		FilePath: normalizedPath,
	}, nil
}

func (s *fileService) GetFile(ctx context.Context, filePath string) ([]byte, string, error) {
	// Check if the file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil, "", errcom.ErrFileNotFound
		}
		return nil, "", fmt.Errorf("unable to access file")
	}

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read file")
	}

	// Extract filename from the file path
	fileName := filepath.Base(filePath)

	return content, fileName, nil
}
func (s *fileService) DeleteFile(ctx context.Context, filepath model.DeleteFileRequest) error {

	if _, err := os.Stat(filepath.FilePath); os.IsNotExist(err) {
		return errcom.ErrFileDoesNotExist
	}

	// Delete the file
	if err := os.Remove(filepath.FilePath); err != nil {
		return fmt.Errorf("unable to delete file")
	}
	return nil
}

func (s *fileService) OpenFile(ctx context.Context, fileURL string) (*os.File, error) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Errorf("Error getting current working directory")
	}
	fmt.Println("Current Working Directory:", dir)

	localRoot := dir

	// Join with the local path
	localFilePath := filepath.Join(localRoot, fileURL)

	// Check file existence and handle any errors
	if _, err := os.Stat(localFilePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", localFilePath)
		}
		// Handle other errors like permission denied or others
		return nil, fmt.Errorf("error checking file existence: %w", err)
	}

	// Open the file
	file, err := os.Open(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file")
	}

	// Ensure the file is closed when done, if the function is extended
	// defer file.Close()

	return file, nil
}

// ChunkFileToKafka sends a file path to the sample-dataset-paths Kafka topic
// func (s *fileService) ChunkFileToKafka(ctx context.Context, req dto.ChunkFileRequest) (*dto.ChunkFileResponse, error) {
// 	// Validate request parameters
// 	if req.Name == "" {
// 		return nil, fmt.Errorf("name cannot be empty")
// 	}
// 	if req.UUID == "" {
// 		return nil, fmt.Errorf("uuid cannot be empty")
// 	}
// 	if req.FilePath == "" {
// 		return nil, fmt.Errorf("file path cannot be empty")
// 	}

// 	// Verify that the file exists
// 	fileInfo, err := os.Stat(req.FilePath)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			return nil, fmt.Errorf("file does not exist: %s", req.FilePath)
// 		}
// 		return nil, fmt.Errorf("failed to get file stats: %w", err)
// 	}

// 	// Check if it's a regular file
// 	if !fileInfo.Mode().IsRegular() {
// 		return nil, fmt.Errorf("not a regular file: %s", req.FilePath)
// 	}

// 	// Create a message with the file path and metadata
// 	pathMsg := map[string]interface{}{
// 		"name":     req.Name,
// 		"uuid":     req.UUID,
// 		"filepath": req.FilePath,
// 		"filesize": fileInfo.Size(),
// 	}

// 	// Marshal the message to JSON
// 	msgData, err := json.Marshal(pathMsg)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to marshal message data: %w", err)
// 	}

// 	// Initialize Kafka writer for the sample-dataset-paths topic
// 	writer := kafkas.GetKafkaWriter("sample-dataset-paths", os.Getenv("KAFKA_BROKER_ADDRESS"))

// 	// Send the message to Kafka
// 	err = writer.WriteMessages(ctx, kafka.Message{
// 		Key:   []byte(req.UUID),
// 		Value: msgData,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("kafka send error: %w", err)
// 	}

// 	// Return success response with the available fields
// 	return &dto.ChunkFileResponse{
// 		Message:  fmt.Sprintf("File path successfully sent to Kafka topic 'sample-dataset-paths' for chunking. File size: %d bytes", fileInfo.Size()),
// 		Name:     req.Name,
// 		UUID:     req.UUID,
// 		FilePath: req.FilePath,
// 	}, nil
// }

// GetAllSampleDatasets retrieves all sample datasets from the repository
func (s *fileService) GetAllSampleDatasets(ctx context.Context) ([]dto.SampleDatasetResponse, error) {
	// Call the repository method to get all sample datasets
	datasets, err := s.sampleDatasetRepo.GetAllSampleDatasets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sample datasets: %w", err)
	}

	return datasets, nil
}

// TestKong is a simple endpoint to check if Kong is running
func (s *fileService) TestKong(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"message": "dataset kong api up and running",
	}, nil
}

// ExtendedChunkFileToKafka sends a file path and form data to the sample-dataset-paths Kafka topic
func (s *fileService) ChunkFileToKafka(ctx context.Context, req dto.ChunkFileRequest) (*dto.ChunkFileResponse, error) {
	// Validate request parameters
	if req.Name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	if req.UUID == "" {
		return nil, fmt.Errorf("uuid cannot be empty")
	}
	if req.FilePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}
	if req.FormData.Type == 0 {
		return nil, fmt.Errorf("formData cannot be nil")
	}

	// Verify that the file exists
	fileInfo, err := os.Stat(req.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", req.FilePath)
		}
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	// Check if it's a regular file
	if !fileInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", req.FilePath)
	}

	// Use the entire request as the message
	// Marshal the message to JSON
	msgData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message data: %w", err)
	}

	// Initialize Kafka writer for the sample-dataset-paths topic
	writer := kafkas.GetKafkaWriter("sample-dataset-paths", os.Getenv("EXT_KAFKA_BROKER_ADDRESS"))

	// Send the message to Kafka
	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(req.UUID),
		Value: msgData,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka send error: %w", err)
	}

	// Return success response
	return &dto.ChunkFileResponse{
		Message:  fmt.Sprintf("File path and form data successfully sent to Kafka topic 'sample-dataset-paths' for processing. File size: %d bytes", fileInfo.Size()),
		Name:     req.Name,
		UUID:     req.UUID,
		FilePath: req.FilePath,
	}, nil
}
