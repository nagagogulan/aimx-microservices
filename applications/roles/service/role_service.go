package service

import (
	"context"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/gofrs/uuid"
)

type RoleService interface {
	CreateRole(ctx context.Context, req *dto.CreateRoleRequest) (*dto.RoleResponse, error)
	GetRoleByID(ctx context.Context, id uuid.UUID) (*dto.RoleResponse, error)
	UpdateRole(ctx context.Context, req *dto.UpdateRoleRequest) (*dto.RoleResponse, error)
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context) ([]dto.RoleResponse, error)
	TestKong(ctx context.Context) (map[string]string, error)
}

type roleService struct {
	repo repository.RoleRepositoryService
}

func NewRoleService(repo repository.RoleRepositoryService) RoleService {
	return &roleService{repo: repo}
}

func (s *roleService) CreateRole(ctx context.Context, req *dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	return s.repo.CreateRole(ctx, req)
}

func (s *roleService) GetRoleByID(ctx context.Context, id uuid.UUID) (*dto.RoleResponse, error) {
	return s.repo.GetRoleByID(ctx, id)
}

func (s *roleService) UpdateRole(ctx context.Context, req *dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	return s.repo.UpdateRole(ctx, req)
}

func (s *roleService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRole(ctx, id)
}

func (s *roleService) ListRoles(ctx context.Context) ([]dto.RoleResponse, error) {
	return s.repo.GetAllRoles(ctx)
}

// TestKong is a simple endpoint to check if Kong is running
func (s *roleService) TestKong(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"message": "role kong api up and running",
	}, nil
}
