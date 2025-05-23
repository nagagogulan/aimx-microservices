package model

// FileUploadRequest holds the request for file upload.
// We might not need a specific request model if we handle the multipart form directly in the transport.
// However, it's good practice to define it if there are other metadata fields expected.
type FileUploadRequest struct {
	// FileName string // Example: if you want to allow client to specify a name
	// FileData []byte // This will be handled as a stream for large files
}

// FileUploadResponse holds the response after a file upload.
type FileUploadResponse struct {
	Message  string `json:"message"`
	FileName string `json:"file_name,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
	FilePath string `json:"file_path,omitempty"`
}

// DocketStatusMessage represents the structure of a message for docket status updates
type DocketStatusMessage struct {
	UUID   string `json:"uuid"`
	Status string `json:"status"` // "failed" or "success"
}
