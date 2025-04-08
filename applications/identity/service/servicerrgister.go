package service

import (
	"context"
	"fmt"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	kitlog "github.com/go-kit/log"
	"whatsdare.com/fullstack/aimx/backend/model"
)

type Service interface {
	// User management
	LoginWithOTP(ctx context.Context, req *dto.UserAuthRequest) (*model.Response, error)
	VerifyOTP(ctx context.Context, req *dto.UserAuthDetail) (*model.UserAuthResponse, error)
}
type service struct {
	UserRepo repository.UserRepositoryService
	logger   kitlog.Logger
}

func NewService(UserRepo repository.UserRepositoryService) Service {
	fmt.Println("db interface connected")
	return &service{UserRepo: UserRepo}
}
