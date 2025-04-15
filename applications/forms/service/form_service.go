package service

import (
	"context"
	"fmt"

	"github.com/PecozQ/aimx-library/apperrors"
	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
)

func (s *service) CreateForm(ctx context.Context, form dto.FormDTO) (*dto.FormDTO, error) {
	fmt.Println("******************************************", form)
	createdForm, err := s.formRepo.CreateForm(ctx, form)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "CreateTemplate", err.Error(), err, "CreateBy", createdForm)
		return nil, err
	}
	return createdForm, err
}

func (s *service) GetFormByType(ctx context.Context, doc_type int) ([]*dto.FormDTO, error) {

	formList, err := s.formRepo.GetFormByType(ctx, doc_type)
	if err != nil {
		fmt.Println("))))))))))))))))))))))))))))))))", err)
		//commonlib.LogMessage(s.logger, commonlib.Error, "GetForms", err.Error(), err, "type", doc_type)
		return nil, err
	}
	if commonlib.IsEmpty(formList) {
		return nil, apperrors.ErrNotFound
	}
	return formList, nil
}
