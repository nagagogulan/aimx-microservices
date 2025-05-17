package model

import (
	"mime/multipart"

	"github.com/gofrs/uuid"
)

type UploadProfileImageRequest struct {
	UserID     uuid.UUID
	FileHeader *multipart.FileHeader
}

type UploadProfileImageResponse struct {
	Message   string `json:"message"`
	ImagePath string `json:"image_path"`
}
