package model

import "github.com/gofrs/uuid"

type UploadRequest struct {
	FileName string `json:"fileName"`
	Content  []byte `json:"content"`
	FilePath string `json:"file_path"`
}
type UploadResponse struct {
	ID       uuid.UUID `json:"id"`
	FilePath string    `json:"file_path"`
	Err      string    `json:"error,omitempty"`
}
type GetFileRequest struct {
	FilePath string `json:"filePath"`
}

type GetFileResponse struct {
	FileName string `json:"fileName"`
	Content  []byte `json:"content"`
	Err      string `json:"error,omitempty"`
}

type DeleteFileRequest struct {
	FileName string `json:"file_name"`
}
type DeleteFileResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
