package service

import (
	"context"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/gofrs/uuid"

)

type Service interface {
	UpdateUserProfile(ctx context.Context, user *entities.User) error
	GetUserProfile(ctx context.Context, id uuid.UUID) (*dto.UserResponseWithDetails, error)
}

type service struct {
	repo repository.UserCRUDService
}

func NewService(repo repository.UserCRUDService) Service {
	return &service{repo: repo}
}

func (s *service) UpdateUserProfile(ctx context.Context, user *entities.User) error {
	return s.repo.UpdateUser(ctx, user)
}

func (s *service) GetUserProfile(ctx context.Context, id uuid.UUID) (*dto.UserResponseWithDetails, error) {
	return s.repo.GetUserByID(ctx, id)
}
