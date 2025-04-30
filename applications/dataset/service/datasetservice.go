package service

import (
	"context"
	"fmt"
	"log"
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
}

type fileService struct{}

func NewService() Service {
	return &fileService{}
}

func (s *fileService) UploadFile(ctx context.Context, req model.UploadRequest) (*model.UploadResponse, error) {
	fmt.Println("user id cheing log", req.UserID)
	id, err := uuid.NewV4()
	if err != nil {
		log.Fatalf("failed to generate UUID: %v", err)
	}
	enumLabel := common.ValueMapper(req.FormType, "FileFormat", "ENUM_TO_HASH")
	//timestampStr := time.Now().UTC().Format("20060102T150405Z")
	timestamp := time.Now().Format("20060102_150405")
	validDatasetExtensions := map[string]bool{
		"csv":  true,
		"xlsx": true,
		"zip":  true,
	}
	validImageExtensions := map[string]bool{
		"jpg":  true,
		"jpeg": true,
		"png":  true,
		"gif":  true,
	}
	ext := strings.ToLower(req.Extension)
	var filePath string
	switch enumLabel {
	case 0:
		if !validDatasetExtensions[ext] {
			return nil, fmt.Errorf("invalid dataset file extension: only .csv and .xlsx allowed")
		}
		filePath = fmt.Sprintf("dataset/%s/sample/%s_%s", id.String(), timestamp, id.String())
	case 1:
		if !validImageExtensions[ext] {
			return nil, fmt.Errorf("invalid image file extension: only .jpg, .jpeg, .png, .gif allowed")
		}
		filePath = fmt.Sprintf("images/%s/%s_%s", req.UserID, timestamp, req.UserID)
	case 2:
		filePath = fmt.Sprintf("file/%s/%s_%s", id.String(), timestamp, id.String())
	default:
		return nil, fmt.Errorf("unsupported file format")
	}

	if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to create upload dir: %w", err)
	}
	//originalName := filepath.Base(req.FileName)

	newFileName := fmt.Sprintf("%s_%s.%s", timestamp, id, req.Extension)

	path := filepath.Join(filePath, newFileName)

	if err := os.WriteFile(path, req.Content, 0644); err != nil {
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
