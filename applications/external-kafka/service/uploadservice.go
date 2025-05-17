package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"

	"whatsdare.com/fullstack/aimx/backend/kafka"
	"whatsdare.com/fullstack/aimx/backend/model"
)

// UploadService defines the interface for file upload operations.
type UploadService interface {
	UploadFile(ctx context.Context, fileHeader *multipart.FileHeader) (model.FileUploadResponse, error)
}

// NewUploadService creates a new instance of UploadService.
func NewUploadService(logger log.Logger, uploadDir string) UploadService {
	// Ensure upload directory exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create upload directory %s: %v", uploadDir, err))
		}
	}
	return &uploadService{
		logger:    logger,
		uploadDir: uploadDir,
	}
}

type uploadService struct {
	logger    log.Logger
	uploadDir string // Directory to save uploaded files
}

// UploadFile handles saving the uploaded file.
func (s *uploadService) UploadFile(ctx context.Context, fileHeader *multipart.FileHeader) (model.FileUploadResponse, error) {
	s.logger.Log("method", "UploadFile", "filename", fileHeader.Filename, "size", fileHeader.Size)

	src, err := fileHeader.Open()
	if err != nil {
		s.logger.Log("method", "UploadFile", "error", "failed to open file header", "err", err)
		return model.FileUploadResponse{}, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Generate a unique filename to avoid collisions and add a timestamp
	uniqueFileName := fmt.Sprintf("%s-%s%s",
		time.Now().Format("20060102150405"),
		uuid.New().String(),
		filepath.Ext(fileHeader.Filename))

	dstPath := filepath.Join(s.uploadDir, uniqueFileName)

	dst, err := os.Create(dstPath)
	if err != nil {
		s.logger.Log("method", "UploadFile", "error", "failed to create destination file", "path", dstPath, "err", err)
		return model.FileUploadResponse{}, fmt.Errorf("failed to create file on server: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, src)
	if err != nil {
		s.logger.Log("method", "UploadFile", "error", "failed to copy file content", "path", dstPath, "err", err)
		// Attempt to remove partially written file
		os.Remove(dstPath)
		return model.FileUploadResponse{}, fmt.Errorf("failed to save file: %w", err)
	}

	s.logger.Log("method", "UploadFile", "status", "file uploaded successfully", "path", dstPath, "written_bytes", written)

	kafka.ProduceFilePath(dstPath, "docket-chunks", os.Getenv("KAFKA_BROKER_ADDRESS"))

	return model.FileUploadResponse{
		Message:  "File uploaded successfully.",
		FileName: uniqueFileName,
		FileSize: written,
		FilePath: dstPath,
	}, nil
}
