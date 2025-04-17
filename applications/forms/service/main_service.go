package service

import (
	"context"
	"fmt"

	"github.com/PecozQ/aimx-library/domain/dto"
	entity "github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/repository"
	kitlog "github.com/go-kit/log"
	"whatsdare.com/fullstack/aimx/backend/model"
)

type Service interface {
	CreateTemplate(ctx context.Context, template entity.Template) (*entity.Template, error)
	GetTemplateByType(ctx context.Context, Type int, id string) (*entity.Template, error)
	UpdateTemplate(ctx context.Context, id string, template entity.Template) (*entity.Template, error)
	DeleteTemplate(ctx context.Context, id string) (*model.Response, error)

	CreateForm(ctx context.Context, forms dto.FormDTO) (*dto.FormDTO, error)
	GetFormByType(ctx context.Context, form_type int) ([]*dto.FormDTO, error)
	UpdateForm(ctx context.Context, id string, status string) (bool, error)

	CreateFormType(ctx context.Context, formtype dto.FormType) (*dto.FormType, error)
	GetAllFormTypes(ctx context.Context) ([]dto.FormType, error)
}

type service struct {
	templateRepo repository.TemplateRepositoryService
	formRepo     repository.FormRepositoryService
	formTypeRepo repository.FormTypeRepositoryService
	logger       kitlog.Logger
}

func NewService(templateRepo repository.TemplateRepositoryService, formRepo repository.FormRepositoryService, formTypeRepo repository.FormTypeRepositoryService) Service {
	fmt.Println("db interface connected")
	return &service{templateRepo: templateRepo, formRepo: formRepo, formTypeRepo: formTypeRepo}
}
