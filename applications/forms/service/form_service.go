package service

import (
	"context"
	"errors"

	errcom "github.com/PecozQ/aimx-library/apperrors"
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

	formList, err := s.formRepo.GetFormByType(ctx, doc_type)
	if err != nil {
		//commonlib.LogMessage(s.logger, commonlib.Error, "GetForms", err.Error(), err, "type", doc_type)
		return nil, NewCustomError(errcom.ErrNotFound, err)
	}
	if commonlib.IsEmpty(formList) {
		return nil, NewCustomError(errcom.ErrNotFound, err)
	}
	return formList, nil
}

func (s *service) UpdateForm(ctx context.Context, id string, status string) (bool, error) {
	updatedForm, err := s.formRepo.UpdateForm(ctx, id, status)
	if err != nil {
		if errors.Is(err, errors.New(errcom.ErrRecordNotFound)) {
			commonlib.LogMessage(s.logger, commonlib.Error, "FormUpdate", err.Error(), nil, "form", id)
			return nil, NewCustomError(errcom.ErrNotFound, err)
		}
		return nil, err
	}
	return updatedForm, nil
}
