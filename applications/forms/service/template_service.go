package service

import (
	"context"
	"errors"

	commonlib "github.com/PecozQ/aimx-library/common"
	entity "github.com/PecozQ/aimx-library/domain/entities"
)

func (s *service) CreateTemplate(ctx context.Context, template entity.Template) (*entity.Template, error) {
	createdTemplate, err := s.templateRepo.CreateTemplate(ctx, template)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "CreateTemplate", err.Error(), err, "CreateBy", template)
		return nil, err
	}
	return createdTemplate, err
}
func (s *service) GetTemplateByType(ctx context.Context, Type int, id string) (*entity.Template, error) {
	if id != "" {
		template, err := s.templateRepo.GetTemplateById(ctx, id)
		if err != nil {
			commonlib.LogMessage(s.logger, commonlib.Error, "GetTemplate", err.Error(), err, "id", Type)
			return nil, errors.New("Template Not Found")
		}
		return template, nil
	}
	if Type > 0 {
		template, errs := s.templateRepo.GetTemplateByType(ctx, Type)
		if errs != nil {
			commonlib.LogMessage(s.logger, commonlib.Error, "GetTemplate", errs.Error(), errs, "type", Type)
			return nil, errs
		}

		return template, nil

	}
	return nil, nil
}
func (s *service) UpdateTemplate(ctx context.Context, id string, template entity.Template) (*entity.Template, error) {
	updatedTemplate, err := s.templateRepo.UpdateTemplate(ctx, id, template)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			commonrepo.LogMessage(s.logger, commonrepo.Error, "Tempalteget", commonrepo.ErrGroupNotFound.Error(), nil, "Template", claims.UserID)
			return commonrepo.ErrGroupNotFound
		}
		return nil, err
	}
	return updatedTemplate, nil
}
func (s *service) DeleteTemplate(ctx context.Context, id string) error {
	err := s.templateRepo.DeleteTemplate(ctx, id)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "DeleteTemplate", err.Error(), err, "id", id)
		return err
	}
	return nil
}
