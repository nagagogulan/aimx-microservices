package service

import (
	"context"
	"errors"
	"fmt"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	errorlib "github.com/PecozQ/aimx-library/apperrors"
	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	entity "github.com/PecozQ/aimx-library/domain/entities"
	"whatsdare.com/fullstack/aimx/backend/model"
)

func (s *service) CreateTemplate(ctx context.Context, template entity.Template) (*entity.Template, error) {
	createdTemplate, err := s.templateRepo.CreateTemplate(ctx, template)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "CreateTemplate", err.Error(), err, "CreateBy", template)
		return nil, err
	}

	// Check if Fields is nil
	if createdTemplate.Fields == nil {
		return nil, fmt.Errorf("fields in created template are nil")
	}

	// Initialize labels
	var labels []string
	for _, field := range createdTemplate.Fields {
		if field.Filter { // Check "filterapplicable" field
			// Ensure the label is not nil or empty
			if field.Label != "" {
				labels = append(labels, field.Label)
			} else {
				fmt.Println("Warning: Found empty label for field:", field)
			}
		}
	}
	filterFieldsRequest := &dto.FilterFieldRequest{
		Type:         createdTemplate.Type,
		FilterFields: labels,
	}

	// Log the FilterFieldsRequest
	fmt.Println("FilterFieldsRequest:", filterFieldsRequest)

	// Call AddSearchfilterFields to add the filter fields
	errs := s.filterfieldRepo.AddSearchfilterFields(ctx, filterFieldsRequest)
	if errs != nil {
		fmt.Println("Error in AddSearchfilterFields:", errs)
		return nil, errcom.ErrNotFound
	}

	return createdTemplate, nil
}

func (s *service) GetTemplateByType(ctx context.Context, Type int, id string) (*entity.Template, error) {
	if id != "" {
		template, err := s.templateRepo.GetTemplateById(ctx, id)
		if err != nil {
			commonlib.LogMessage(s.logger, commonlib.Error, "GetTemplate", err.Error(), err, "id", Type)
			return nil, NewCustomError(errorlib.ErrNotFound, err)
		}
		return template, nil
	}
	if Type > 0 {
		template, errs := s.templateRepo.GetTemplateByType(ctx, Type)
		if errs != nil {
			commonlib.LogMessage(s.logger, commonlib.Error, "GetTemplate", errs.Error(), errs, "type", Type)
			return nil, NewCustomError(errorlib.ErrNotFound, errs)
		}

		return template, nil

	}
	return nil, nil
}

func (s *service) UpdateTemplate(ctx context.Context, id string, template entity.Template) (*entity.Template, error) {
	updatedTemplate, err := s.templateRepo.UpdateTemplate(ctx, id, template)
	if err != nil {
		if errors.Is(err, errors.New(errorlib.ErrRecordNotFound)) {
			commonlib.LogMessage(s.logger, commonlib.Error, "Tempalteget", err.Error(), nil, "Template", id)
			return nil, NewCustomError(errorlib.ErrNotFound, err)
		}
		return nil, err
	}
	return updatedTemplate, nil
}
func (s *service) DeleteTemplate(ctx context.Context, id string) (*model.Response, error) {
	err := s.templateRepo.DeleteTemplate(ctx, id)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "DeleteTemplate", err.Error(), err, "id", id)
		return nil, NewCustomError(errorlib.ErrNotFound, err)
	}
	return &model.Response{Message: "Successfully Template deleted"}, nil
}
func (s *service) GetFilterFieldsByType(ctx context.Context, filterType int) (*entities.FilterFieldRequest, error) {
	// Call the repository method to get filter fields by type
	filterFields, err := s.filterfieldRepo.GetFilterFieldsByType(ctx, filterType)
	if err != nil {
		// If there's an error, return it
		return nil, err
	}

	// Return the result
	return filterFields, nil
}
