package service

import (
	"context"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/gofrs/uuid"

)

type Service interface {
	ListUsers(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, page, limit int, search string) (dto.PaginatedUsersResponse, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	DeactivateUser(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo repository.UserCRUDService
}

func NewService(repo repository.UserCRUDService) Service {
	return &service{repo: repo}
}

func (s *service) ListUsers(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, page, limit int, search string) (dto.PaginatedUsersResponse, error) {
	return s.repo.ListUsersByOrg(ctx, orgID, userID, page, limit, search)
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *service) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeactivateUser(ctx, id)
}
