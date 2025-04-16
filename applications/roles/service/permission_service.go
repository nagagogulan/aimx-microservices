package service

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
)

type PermissionService interface {
	CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest) (*dto.PermissionResponse, error)
	GetPermissionByID(ctx context.Context, id uuid.UUID) (*dto.PermissionResponse, error)
	UpdatePermission(ctx context.Context, req *dto.UpdatePermissionRequest) (*dto.PermissionResponse, error)
	DeletePermission(ctx context.Context, id uuid.UUID) error
	ListPermissions(ctx context.Context) ([]dto.PermissionResponse, error)
}

type permissionService struct {
	repo repository.PermissionRepositoryService
}

func NewPermissionService(repo repository.PermissionRepositoryService) PermissionService {
	return &permissionService{repo: repo}
}

func (s *permissionService) CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest) (*dto.PermissionResponse, error) {
	return s.repo.CreatePermission(ctx, req)
}

func (s *permissionService) GetPermissionByID(ctx context.Context, id uuid.UUID) (*dto.PermissionResponse, error) {
	return s.repo.GetPermissionByID(ctx, id)
}

func (s *permissionService) UpdatePermission(ctx context.Context, req *dto.UpdatePermissionRequest) (*dto.PermissionResponse, error) {
	return s.repo.UpdatePermission(ctx, req)
}

func (s *permissionService) DeletePermission(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePermission(ctx, id)
}

func (s *permissionService) ListPermissions(ctx context.Context) ([]dto.PermissionResponse, error) {
	return s.repo.GetAllPermissions(ctx)
}
