package service

import (
	"context"
	"fmt"

	entity "github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/repository"
	kitlog "github.com/go-kit/log"
)

type Service interface {
	CreateTemplate(ctx context.Context, template entity.Template) (*entity.Template, error)
	GetTemplateByType(ctx context.Context, Type int,id string) (*entity.Template, error)
	UpdateTemplate(ctx context.Context, id string, template entity.Template) (*entity.Template, error)
	DeleteTemplate(ctx context.Context, id string) error
}

type service struct {
	templateRepo repository.TemplateRepositoryService
	logger       kitlog.Logger
}

func NewService(templateRepo repository.TemplateRepositoryService) Service {
	fmt.Println("db interface connected")
	return &service{templateRepo: templateRepo}
}
