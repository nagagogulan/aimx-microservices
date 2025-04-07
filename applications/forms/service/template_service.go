package service

import (
	"context"

	commonlib "github.com/PecozQ/aimx-library/common"
	entity "github.com/PecozQ/aimx-library/domain/entities"
)

func (s *service) createTemplate(ctx context.Context, template entity.Template) (*entity.Template, error) {
	createdTemplate, err := s.templateRepo.CreateTemplate(ctx, template)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "CreateTemplate", err.Error(), err, "CreateBy", template)
		return nil, err
	}
	return createdTemplate, err
}
