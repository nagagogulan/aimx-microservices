package service

import (
	"context"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	kitlog "github.com/go-kit/log"
	"whatsdare.com/fullstack/aimx/backend/model"
)

type Service interface {
	// User management
	SendEmailOTP(ctx context.Context, req dto.UserAuthRequest) (*model.Response, error)
	VerifyOTP(ctx context.Context, req *dto.UserAuthdetail) (string, error)
}
type service struct {
	UserRepo repository.UserRepositoryService
	logger   kitlog.Logger
}

func NewService(UserRepo repository.UserRepositoryService) Service {
	return &service{UserRepo: UserRepo}
}
