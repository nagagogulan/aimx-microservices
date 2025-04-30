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
	GetFormByType(ctx context.Context, doc_type, page, limit int) (*model.GetFormResponse, error)
	UpdateForm(ctx context.Context, id string, status string) (*model.Response, error)

	CreateFormType(ctx context.Context, formtype dto.FormType) (*dto.FormType, error)
	GetAllFormTypes(ctx context.Context) ([]dto.FormType, error)

	GetFilteredForms(ctx context.Context, formType int, searchParam dto.SearchParam) ([]*dto.FormDTO, int64, error)
	GetFilterFieldsByType(ctx context.Context, filterType int) (*entity.FilterFieldRequest, error)

	ShortListDocket(ctx context.Context, userId string, dto dto.ShortListDTO) (bool, error)
	RateDocket(ctx context.Context, userId string, dto dto.RatingDTO) (bool, error)

	GetCommentsById(ctx context.Context, interactionId string) ([]*dto.CommentData, error)
}

type service struct {
	templateRepo     repository.TemplateRepositoryService
	formRepo         repository.FormRepositoryService
	formTypeRepo     repository.FormTypeRepositoryService
	organizationRepo repository.OrganizationRepositoryService
	commEventRepo    repository.CommEventRepositoryService
	filterfieldRepo  repository.AddSearchfilterService
	logger           kitlog.Logger
}

func NewService(templateRepo repository.TemplateRepositoryService, formRepo repository.FormRepositoryService, formTypeRepo repository.FormTypeRepositoryService,
	organizationRepo repository.OrganizationRepositoryService, commEventRepo repository.CommEventRepositoryService, filterfieldRepo repository.AddSearchfilterService) Service {
	fmt.Println("db interface connected")
	return &service{templateRepo: templateRepo, formRepo: formRepo, formTypeRepo: formTypeRepo, organizationRepo: organizationRepo, commEventRepo: commEventRepo, filterfieldRepo: filterfieldRepo}
}
