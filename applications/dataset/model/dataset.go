package model

import (
	"github.com/gofrs/uuid"
)

type UploadRequest struct {
	FileName  string `json:"fileName"`
	Content   []byte `json:"content"`
	FormType  string `json:"formType"`
	FileType  string `json:"fileType"`
	Extension string `json:"extension"`
}
type UploadResponse struct {
	ID        uuid.UUID  `json:"id"`
	FilePath  string     `json:"file_path"`
	FileName  string     `json:"fileName"`
	Structure []FileNode `json:"structure,omitempty"`
	Err       string     `json:"error,omitempty"`
}
type GetFileRequest struct {
	FilePath string `json:"filePath"`
}

type GetFileResponse struct {
	FileName    string `json:"fileName"`
	Content     []byte `json:"content"`
	Err         string `json:"error,omitempty"`
	ContentType string `json:"contentType"`
}

type DeleteFileRequest struct {
	FormType int    `json:"formType"`
	FilePath string `json:"filePath"`
}
type DeleteFileResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type OpenFileRequest struct {
	FileURL string `json:"fileURL"`
}

// type OpenFileResponse struct {
// 	File *os.File `json:"file,omitempty"`
// 	Err  string   `json:"err,omitempty"`
// }

// type OpenFileResponse struct {
// 	FileName    string      `json:"fileName"`
// 	FileSize    int64       `json:"fileSize"`
// 	FilePath    string      `json:"filePath"`
// 	FileType    string      `json:"fileType"`
// 	FilePreview interface{} `json:"filePreview"` // Add this field
// 	Err         string      `json:"err,omitempty"`
// }

// ExtendedChunkFileRequest represents the extended request for the chunk file API with form data

// FileNode represents a file or folder in the ZIP structure
// type FileNode struct {
// 	Name     string     `json:"name"`
// 	Type     string     `json:"type"` // "file" or "folder"
// 	Preview  string     `json:"preview,omitempty"`
// 	Children []FileNode `json:"children,omitempty"`
// }

// GetZipPreviewRequest represents a request to get a ZIP file preview
type GetZipPreviewRequest struct {
	ZipPath string `json:"zipPath"`
}

// GetZipPreviewResponse represents the response with ZIP structure
type GetZipPreviewResponse struct {
	ID        uuid.UUID  `json:"id,omitempty"`
	FilePath  string     `json:"file_path,omitempty"`
	Structure []FileNode `json:"structure"`
	Err       string     `json:"error,omitempty"`
}
type OpenFileResponse struct {
	ID          string      `json:"id"`
	Structure   []*FileNode `json:"structure"`
	FileName    string      `json:"fileName,omitempty"`
	FileSize    int64       `json:"fileSize,omitempty"`
	FilePath    string      `json:"filePath,omitempty"`
	FilePreview string      `json:"filePreview,omitempty"`
	Err         string      `json:"err,omitempty"`
	FileType    string      `json:"fileType"`
}

type FileNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"` // "file" or "folder"
	Preview  []string    `json:"preview,omitempty"`
	Children []*FileNode `json:"children,omitempty"`
}
