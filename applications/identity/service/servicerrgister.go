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
	LoginWithOTP(ctx context.Context, req *dto.UserAuthRequest) (*model.Response, error)
	VerifyOTP(ctx context.Context, req *dto.UserAuthDetail) (*model.UserAuthResponse, error)
	VerifyTOTP(ctx context.Context, req *dto.UserAuthDetail) (*model.Response, error)
	UpdateAccessToken(ctx context.Context, req *dto.RefreshAuthDetail) (*model.RefreshTokenResponse, error)
	TestKong(ctx context.Context) (*model.Response, error)
}

type service struct {
	TempUserRepo repository.UserRepositoryService
	OrgRepo      repository.OrganizationRepositoryService
	UserRepo     repository.UserCRUDService
	RoleRepo     repository.RoleRepositoryService
	logger       kitlog.Logger
}

func NewService(tempUserRepo repository.UserRepositoryService, orgRepo repository.OrganizationRepositoryService,
	userRepo repository.UserCRUDService, roleRepo repository.RoleRepositoryService) Service {
	fmt.Println("db interface connected")
	return &service{
		TempUserRepo: tempUserRepo,
		OrgRepo:      orgRepo,
		UserRepo:     userRepo,
		RoleRepo:     roleRepo,
	}
}
