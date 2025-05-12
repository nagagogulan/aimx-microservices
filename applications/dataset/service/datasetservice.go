package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PecozQ/aimx-library/common"
	"github.com/gofrs/uuid"
	"whatsdare.com/fullstack/aimx/backend/model"
)

type Service interface {
	//UploadFile(ctx context.Context, filePath string) (string, error)
	UploadFile(ctx context.Context, req model.UploadRequest) (*model.UploadResponse, error)
	GetFile(ctx context.Context, filePath string) ([]byte, string, error)
	//GetFileList(ctx context.Context) ([]string, error)
	DeleteFile(ctx context.Context, filepath model.DeleteFileRequest) error
	OpenFile(ctx context.Context, filePath string) (*os.File, error)
}

type fileService struct{}

func NewService() Service {
	return &fileService{}
}

func (s *fileService) UploadFile(ctx context.Context, req model.UploadRequest) (*model.UploadResponse, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %v", err)
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
	switch enumLabel {
	case 0:
		if !validDatasetExtensions[ext] {
			return nil, fmt.Errorf("invalid dataset file extension: only .csv, .xlsx, .zip allowed")
		}
		filePath = fmt.Sprintf("datasetfile/%s/%s_%s", id.String(), timestamp, id.String())
	case 1:
		if !validImageExtensions[ext] {
			return nil, fmt.Errorf("invalid image file extension: only .jpg, .jpeg, .png allowed")
		}
		filePath = fmt.Sprintf("images/%s/%s_%s", id.String(), timestamp, id.String())
	case 2:
		if !validFileFormats[ext] {
			return nil, fmt.Errorf("invalid file format: only .docx, .pdf allowed")
		}
		filePath = fmt.Sprintf("file/%s/%s_%s", id.String(), timestamp, id.String())
	case 3:
		if !validDocketFileFormats[ext] {
			return nil, fmt.Errorf("invalid docket format: only .pkl, .joblib, .pth, .h5, .onnx allowed")
		}
		filePath = fmt.Sprintf("docketfile/%s/%s_%s", id.String(), timestamp, id.String())
	default:
		return nil, fmt.Errorf("unsupported file format")
	}

	// Create necessary directories
	if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to create upload dir: %w", err)
	}

	// Construct full file path
	newFileName := fmt.Sprintf("%s_%s.%s", timestamp, id.String(), ext)
	fullPath := filepath.Join(filePath, newFileName)

	// Check if file exceeds a certain size (e.g., 50MB)
	const maxSize = 50 * 1024 * 1024 // 50 MB
	if len(req.Content) > maxSize {
		return nil, fmt.Errorf("file size exceeds the maximum allowed size of 50MB")
	}

	// Create destination file
	outFile, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create file: %w", err)
	}
	defer outFile.Close()

	// Stream the file content to the new file
	contentReader := bytes.NewReader(req.Content)
	if _, err := io.Copy(outFile, contentReader); err != nil {
		return nil, fmt.Errorf("failed to stream file: %w", err)
	}

	// Return response
	return &model.UploadResponse{
		ID:       id,
		FilePath: fullPath,
	}, nil
}

func (s *fileService) GetFile(ctx context.Context, filePath string) ([]byte, string, error) {
	// Check if the file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil, "", fmt.Errorf("file not found: %w", err)
		}
		return nil, "", fmt.Errorf("unable to access file: %w", err)
	}

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read file: %w", err)
	}

	// Extract filename from the file path
	fileName := filepath.Base(filePath)

	return content, fileName, nil
}
func (s *fileService) DeleteFile(ctx context.Context, filepath model.DeleteFileRequest) error {

	if _, err := os.Stat(filepath.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filepath)
	}

	// Delete the file
	if err := os.Remove(filepath.FilePath); err != nil {
		return fmt.Errorf("unable to delete file %s: %w", filepath, err)
	}
	return nil
}

func (s *fileService) OpenFile(ctx context.Context, fileURL string) (*os.File, error) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Errorf("Error getting current working directory:", err)
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
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Ensure the file is closed when done, if the function is extended
	// defer file.Close()

	return file, nil
}
