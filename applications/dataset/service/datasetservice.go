package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/uuid"
	"whatsdare.com/fullstack/aimx/backend/model"
)

type Service interface {
	//UploadFile(ctx context.Context, filePath string) (string, error)
	UploadFile(ctx context.Context, filename string, content []byte) (*model.UploadResponse, error)
	GetFile(ctx context.Context, filePath string) ([]byte, string, error)
	//GetFileList(ctx context.Context) ([]string, error)
	DeleteFile(ctx context.Context, filename string) error
}

type fileService struct{}

func NewService() Service {
	return &fileService{}
}

func (s *fileService) UploadFile(ctx context.Context, filename string, content []byte) (*model.UploadResponse, error) {
	// Custom destination path
	dir := "./Documents"

	// Ensure the directory exists
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to create upload dir: %w", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	originalName := filepath.Base(filename)

	id, err := uuid.NewV4()
	if err != nil {
		log.Fatalf("failed to generate UUID: %v", err)
	}

	// Generate timestamped filename to avoid overwrite
	newFileName := fmt.Sprintf("%s_%s_%s", timestamp, id, originalName)
	path := filepath.Join(dir, newFileName)

	// Write file to the path
	if err := os.WriteFile(path, content, 0644); err != nil {
		return nil, fmt.Errorf("unable to write file: %w", err)
	}

	return &model.UploadResponse{ID: id, FilePath: path}, nil
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
func (s *fileService) DeleteFile(ctx context.Context, filename string) error {
	dir := "./Documents"
	fullPath := filepath.Join(dir, filename)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("unable to delete file %s: %w", filename, err)
	}
	return nil
}

// func (s *fileService) GetFileList(ctx context.Context) ([]string, error) {
// 	dir := "C:\\Users\\nithiyav\\Documents"

// 	// Read directory entries
// 	files, err := os.ReadDir(dir)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to read directory: %w", err)
// 	}

// 	var fileList []string
// 	for _, file := range files {
// 		if !file.IsDir() {
// 			fileList = append(fileList, file.Name()) // or fileList = append(fileList, filepath.Join(dir, file.Name()))
// 		}
// 	}

// 	return fileList, nil
// }
