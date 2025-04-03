package service

import (
	"context"

	entity "github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/repository"
)

type Service interface {
	CreateTemplate(ctx context.Context, template entity.Template) (*entity.Template, error)
	GetTemplateByID(ctx context.Context, id string) (*entity.Template, error)
}

type service struct {
	templateRepo repository.TemplateRepositoryService
}
