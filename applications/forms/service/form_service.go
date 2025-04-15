package service

import (
	"context"
	"fmt"

	"github.com/PecozQ/aimx-library/apperrors"
	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
)

func (s *service) CreateForm(ctx context.Context, form dto.FormDTO) (*dto.FormDTO, error) {
	createdForm, err := s.formRepo.CreateForm(ctx, form)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "CreateTemplate", err.Error(), err, "CreateBy", createdForm)
		return nil, err
	}
	return createdForm, err
}

func (s *service) GetFormByType(ctx context.Context, doc_type int) ([]*dto.FormDTO, error) {
	// if id != "" {
	// 	template, err := s.templateRepo.GetTemplateById(ctx, id)
	// 	if err != nil {
	// 		fmt.Println("****************************")
	// 		//commonlib.LogMessage(s.logger, commonlib.Error, "GetTemplate", err.Error(), err, "type", Type)
	// 		return nil, errors.New("Template Not Found")
	// 	}
	// 	return template, nil
	// }
	// if Type > 0 {
	formList, errs := s.formRepo.GetFormByType(ctx, doc_type)
	if errs != nil {
		fmt.Println("****************************")
		commonlib.LogMessage(s.logger, commonlib.Error, "GetForms", errs.Error(), errs, "type", doc_type)
		return nil, apperrors.ErrNotFound
	}
	if commonlib.IsEmpty(formList) {
		return nil, apperrors.ErrNotFound
	}
	return formList, nil

	// }
	// return nil, nil
}
