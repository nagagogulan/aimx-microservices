package service

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
)

type RMPService interface {
	CreateRMP(ctx context.Context, req *dto.CreateRMPRequest) (*dto.RMPResponse, error)
	GetRMPByID(ctx context.Context, id uuid.UUID) (*dto.RMPResponse, error)
	UpdateRMP(ctx context.Context, req *dto.UpdateRMPRequest) (*dto.RMPResponse, error)
	DeleteRMP(ctx context.Context, id uuid.UUID) error
	ListRMP(ctx context.Context) ([]dto.RMPResponse, error)
	GetModulesAndPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]dto.RMPResponse, error)

}

type rmpService struct {
	repo repository.RMPRepositoryService
}

func NewRMPService(repo repository.RMPRepositoryService) RMPService {
	return &rmpService{repo: repo}
}

func (s *rmpService) CreateRMP(ctx context.Context, req *dto.CreateRMPRequest) (*dto.RMPResponse, error) {
	return s.repo.CreateRMP(ctx, req)
}

func (s *rmpService) GetRMPByID(ctx context.Context, id uuid.UUID) (*dto.RMPResponse, error) {
	return s.repo.GetRMPByID(ctx, id)
}

func (s *rmpService) UpdateRMP(ctx context.Context, req *dto.UpdateRMPRequest) (*dto.RMPResponse, error) {
	return s.repo.UpdateRMP(ctx, req)
}

func (s *rmpService) DeleteRMP(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRMP(ctx, id)
}

func (s *rmpService) ListRMP(ctx context.Context) ([]dto.RMPResponse, error) {
	return s.repo.GetAllRMP(ctx)
}

func (s *rmpService) GetModulesAndPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]dto.RMPResponse, error) {
	return s.repo.GetModulesAndPermissionsByRoleID(ctx, roleID)
}

