package service

import (
	"context"
	"fmt"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/gofrs/uuid"
)

type Service interface {
	ListUsers(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, page, limit int, search string, filters map[string]interface{}, reqType string) (dto.UpdatedPaginationResponse, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	TestKong(ctx context.Context) (map[string]string, error)
}

type service struct {
	repo repository.UserCRUDService
}

func NewService(repo repository.UserCRUDService) Service {
	return &service{repo: repo}
}

func (s *service) ListUsers(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, page, limit int, search string, filters map[string]interface{}, reqType string) (dto.UpdatedPaginationResponse, error) {
	fmt.Printf("testttt", reqType)
	if reqType == "AllOrganization" {
		return s.repo.ListUsersByCondition(ctx, nil, userID, page, limit, search, filters)
	} else {
		return s.repo.ListUsersByCondition(ctx, &orgID, userID, page, limit, search, filters)
	}

}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *service) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeactivateUser(ctx, id)
}

// TestKong is a simple endpoint to check if Kong is running
func (s *service) TestKong(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"message": "user kong api up and running",
	}, nil
}
