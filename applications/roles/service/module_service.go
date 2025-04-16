package service

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
)

type ModuleService interface {
	CreateModule(ctx context.Context, req *dto.CreateModuleRequest) (*dto.ModuleResponse, error)
	GetModuleByID(ctx context.Context, id uuid.UUID) (*dto.ModuleResponse, error)
	UpdateModule(ctx context.Context, req *dto.UpdateModuleRequest) (*dto.ModuleResponse, error)
	DeleteModule(ctx context.Context, id uuid.UUID) error
	ListModules(ctx context.Context) ([]dto.ModuleResponse, error)
}

type moduleService struct {
	repo repository.ModuleRepositoryService
}

func NewModuleService(repo repository.ModuleRepositoryService) ModuleService {
	return &moduleService{repo: repo}
}

func (s *moduleService) CreateModule(ctx context.Context, req *dto.CreateModuleRequest) (*dto.ModuleResponse, error) {
	return s.repo.CreateModule(ctx, req)
}

func (s *moduleService) GetModuleByID(ctx context.Context, id uuid.UUID) (*dto.ModuleResponse, error) {
	return s.repo.GetModuleByID(ctx, id)
}

func (s *moduleService) UpdateModule(ctx context.Context, req *dto.UpdateModuleRequest) (*dto.ModuleResponse, error) {
	return s.repo.UpdateModule(ctx, req)
}

func (s *moduleService) DeleteModule(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteModule(ctx, id)
}

func (s *moduleService) ListModules(ctx context.Context) ([]dto.ModuleResponse, error) {
	return s.repo.GetAllModules(ctx)
}
