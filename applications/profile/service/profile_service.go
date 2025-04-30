package service

import (
	"context"
	"fmt"

	common "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/gofrs/uuid"
)

type Service interface {
	UpdateUserProfile(ctx context.Context, user *entities.User) error
	GetUserProfile(ctx context.Context, id uuid.UUID) (*dto.UserResponseWithDetails, error)
	CreateGeneralSetting(ctx context.Context, setting *dto.GeneralSettingRequest) error
	UpdateGeneralSetting(ctx context.Context, setting *dto.GeneralSettingRequest) (*dto.GeneralSettingResponse, error)
	GetAllGeneralSettings(ctx context.Context) ([]*dto.GeneralSettingResponse, error)
	GetAllNonSingHealthOrganizations(ctx context.Context) ([]*dto.OrganizationListResponse, error)
	UpdateOrganizationSettingByOrgID(ctx context.Context, setting *dto.OrganizationSettingRequest) error
	GetOrganizationSettingByOrgID(ctx context.Context, organizationID uuid.UUID) (*dto.OrganizationSettingResponse, error)

}

type service struct {
	repo               repository.UserCRUDService
	generalSettingRepo repository.GeneralSettingRepository
	orgRepo            repository.OrganizationRepositoryService
	orgSettingRepo  repository.OrganizationSettingRepository

}

func NewService(
	repo repository.UserCRUDService,
	generalSettingRepo repository.GeneralSettingRepository,
	orgRepo repository.OrganizationRepositoryService,
	orgSettingRepo  repository.OrganizationSettingRepository,
) Service {
	return &service{
		repo:               repo,
		generalSettingRepo: generalSettingRepo,
		orgRepo:            orgRepo,
		orgSettingRepo: orgSettingRepo,
	}
}

func (s *service) UpdateUserProfile(ctx context.Context, user *entities.User) error {
	return s.repo.UpdateUser(ctx, user)
}

func (s *service) GetUserProfile(ctx context.Context, id uuid.UUID) (*dto.UserResponseWithDetails, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *service) CreateGeneralSetting(ctx context.Context, setting *dto.GeneralSettingRequest) error {
	// Map string to int
	fmt.Println(common.ENUM_TO_HASH, setting.MaxProjectDocketSizeUnit)
	unitMap := common.ENUM_TO_HASH["MaxProjectDocketSizeUnit"]

	unitInt, ok := unitMap[setting.MaxProjectDocketSizeUnit]
	fmt.Println(unitInt, ok)
	if !ok {
		return fmt.Errorf("invalid MaxProjectDocketSizeUnit: %s", setting.MaxProjectDocketSizeUnit)
	}

	entity := &entities.GeneralSetting{
		DefaultDeletionDays:      setting.DefaultDeletionDays,
		DefaultArchivingDays:     setting.DefaultArchivingDays,
		MaxActiveProjects:        setting.MaxActiveProjects,
		MaxUsersPerOrganization:  setting.MaxUsersPerOrganization,
		MaxProjectDocketSize:     setting.MaxProjectDocketSize,
		MaxProjectDocketSizeUnit: unitInt, // Save as int
		ScheduledEvaluationTime:  setting.ScheduledEvaluationTime,
	}
	return s.generalSettingRepo.CreateGeneralSetting(entity)
}

func (s *service) UpdateGeneralSetting(ctx context.Context, setting *dto.GeneralSettingRequest) (*dto.GeneralSettingResponse, error) {
	unitMap := common.ENUM_TO_HASH["MaxProjectDocketSizeUnit"]
	unitInt, ok := unitMap[setting.MaxProjectDocketSizeUnit]
	if !ok {
		return nil, fmt.Errorf("invalid MaxProjectDocketSizeUnit: %s", setting.MaxProjectDocketSizeUnit)
	}

	// Important: pass ID also
	entity := &entities.GeneralSetting{
		ID:                       setting.ID, // you must pass ID to identify which record to update
		DefaultDeletionDays:      setting.DefaultDeletionDays,
		DefaultArchivingDays:     setting.DefaultArchivingDays,
		MaxActiveProjects:        setting.MaxActiveProjects,
		MaxUsersPerOrganization:  setting.MaxUsersPerOrganization,
		MaxProjectDocketSize:     setting.MaxProjectDocketSize,
		MaxProjectDocketSizeUnit: unitInt,
		ScheduledEvaluationTime:  setting.ScheduledEvaluationTime,
	}

	updatedEntity, err := s.generalSettingRepo.UpdateGeneralSetting(entity)
	if err != nil {
		return nil, err
	}

	unitEnum := common.HASH_TO_ENUM["MaxProjectDocketSizeUnit"][updatedEntity.MaxProjectDocketSizeUnit]
	if unitEnum == "" {
		unitEnum = "UNKNOWN"
	}

	response := &dto.GeneralSettingResponse{
		ID:                       updatedEntity.ID,
		DefaultDeletionDays:      updatedEntity.DefaultDeletionDays,
		DefaultArchivingDays:     updatedEntity.DefaultArchivingDays,
		MaxActiveProjects:        updatedEntity.MaxActiveProjects,
		MaxUsersPerOrganization:  updatedEntity.MaxUsersPerOrganization,
		MaxProjectDocketSize:     updatedEntity.MaxProjectDocketSize,
		MaxProjectDocketSizeUnit: unitEnum,
		ScheduledEvaluationTime:  updatedEntity.ScheduledEvaluationTime,
		CreatedAt:                updatedEntity.CreatedAt,
		UpdatedAt:                updatedEntity.UpdatedAt,
	}

	return response, nil
}

func (s *service) GetAllGeneralSettings(ctx context.Context) ([]*dto.GeneralSettingResponse, error) {
	settings, err := s.generalSettingRepo.GetAllGeneralSetting()
	if err != nil {
		return nil, err
	}

	var response []*dto.GeneralSettingResponse
	for _, setting := range settings {
		unitEnum := common.HASH_TO_ENUM["MaxProjectDocketSizeUnit"][setting.MaxProjectDocketSizeUnit]
		if unitEnum == "" {
			unitEnum = "UNKNOWN"
		}

		response = append(response, &dto.GeneralSettingResponse{
			ID:                       setting.ID,
			DefaultDeletionDays:      setting.DefaultDeletionDays,
			DefaultArchivingDays:     setting.DefaultArchivingDays,
			MaxActiveProjects:        setting.MaxActiveProjects,
			MaxUsersPerOrganization:  setting.MaxUsersPerOrganization,
			MaxProjectDocketSize:     setting.MaxProjectDocketSize,
			MaxProjectDocketSizeUnit: unitEnum, // Return string back
			ScheduledEvaluationTime:  setting.ScheduledEvaluationTime,
			CreatedAt:                setting.CreatedAt,
			UpdatedAt:                setting.UpdatedAt,
		})
	}

	return response, nil
}

func (s *service) GetAllNonSingHealthOrganizations(ctx context.Context) ([]*dto.OrganizationListResponse, error) {
	orgs, err := s.orgRepo.GetAllNonSingHealthOrganizations()
	if err != nil {
		return nil, err
	}

	var response []*dto.OrganizationListResponse
	for _, org := range orgs {
		response = append(response, &dto.OrganizationListResponse{
			OrganizationID:   org.OrganizationID,
			OrganizationName: org.OrganizationName,
		})
	}

	return response, nil
}

func (s *service) UpdateOrganizationSettingByOrgID(ctx context.Context, setting *dto.OrganizationSettingRequest) error {
	entity := &entities.OrganizationSetting{
		ID:                      setting.ID,
		OrgID:          setting.OrgID,
		DefaultDeletionDays:     setting.DefaultDeletionDays,
		DefaultArchivingDays:    setting.DefaultArchivingDays,
		MaxActiveProjects:       setting.MaxActiveProjects,
		MaxUsersPerOrganization: setting.MaxUsersPerOrganization,
		MaxProjectDocketSize:    setting.MaxProjectDocketSize,
		MaxProjectDocketSizeUnit: setting.MaxProjectDocketSizeUnit,
		ScheduledEvaluationTime: setting.ScheduledEvaluationTime,
	}
	return s.orgSettingRepo.UpdateOrganizationSettingByOrgID(ctx, entity)
}


func (s *service) GetOrganizationSettingByOrgID(ctx context.Context, orgID uuid.UUID) (*dto.OrganizationSettingResponse, error) {
	setting, err := s.orgSettingRepo.GetOrganizationSettingByOrgID(ctx, orgID.String())
	if err != nil {
		return nil, err
	}

	return &dto.OrganizationSettingResponse{
		ID:                       setting.ID,
		OrgID:           setting.OrgID,
		DefaultDeletionDays:      setting.DefaultDeletionDays,
		DefaultArchivingDays:     setting.DefaultArchivingDays,
		MaxActiveProjects:        setting.MaxActiveProjects,
		MaxUsersPerOrganization:  setting.MaxUsersPerOrganization,
		MaxProjectDocketSize:     setting.MaxProjectDocketSize,
		MaxProjectDocketSizeUnit: setting.MaxProjectDocketSizeUnit,
		ScheduledEvaluationTime:  setting.ScheduledEvaluationTime,
		CreatedAt:                setting.CreatedAt,
		UpdatedAt:                setting.UpdatedAt,
	}, nil
}
