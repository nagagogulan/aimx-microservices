package service

import (
	"context"
	"fmt"
	"log"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
)

// FormsService handles interactions with form data
type FormsService interface {
	GetFormByID(ctx context.Context, formID string) (*dto.FormDTO, error)
	UpdateMetadataWithFormData(metadata map[string]interface{}) (map[string]interface{}, error)
}

type formsService struct {
	repo repository.FormRepositoryService
}

// NewFormsService creates a new instance of the FormsService
func NewFormsService(repo repository.FormRepositoryService) FormsService {
	return &formsService{
		repo: repo,
	}
}

// GetFormByID retrieves form data by its ID
func (s *formsService) GetFormByID(ctx context.Context, formID string) (*dto.FormDTO, error) {
	// Get form data from repository
	form, err := s.repo.GetFormById(ctx, formID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving form data: %w", err)
	}

	return form, nil
}

// UpdateMetadataWithFormData updates the metadata with form data
func (s *formsService) UpdateMetadataWithFormData(metadata map[string]interface{}) (map[string]interface{}, error) {
	// Extract the modelDatasetUrl UUID from metadata
	modelDatasetUrlValue, exists := metadata["modelDatasetUrl"]
	if !exists {
		return metadata, fmt.Errorf("modelDatasetUrl not found in metadata")
	}

	datasetUUID, ok := modelDatasetUrlValue.(string)
	if !ok {
		return metadata, fmt.Errorf("modelDatasetUrl is not a string")
	}

	log.Printf("Fetching form data for dataset UUID: %s", datasetUUID)

	// Get form data
	form, err := s.GetFormByID(context.Background(), datasetUUID)
	if err != nil {
		return metadata, err
	}

	// Check if MetaData exists in the form
	if form.MetaData == nil {
		return metadata, fmt.Errorf("MetaData not found in form data")
	}

	// Try to extract originalDataset from MetaData
	metaData, ok := form.MetaData.(map[string]interface{})
	if !ok {
		return metadata, fmt.Errorf("MetaData is not a map")
	}

	originalDataset, exists := metaData["originalDataset"]
	if !exists {
		return metadata, fmt.Errorf("originalDataset not found in form MetaData")
	}

	log.Printf("Found originalDataset in form data")

	// Update the modelDatasetUrl in metadata with the originalDataset value
	metadata["modelDatasetUrl"] = originalDataset

	return metadata, nil
}
