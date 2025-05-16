package service

import (
	"context"
	"fmt"

	errcom "github.com/PecozQ/aimx-library/apperrors"
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
	GetAllGeneralSettings(ctx context.Context) (*dto.GeneralSettingResponse, error)
	GetAllNonSingHealthOrganizations(ctx context.Context) ([]*dto.OrganizationListResponse, error)
	UpdateOrganizationSettingByOrgID(ctx context.Context, setting *dto.OrganizationSettingRequest) error
	GetOrganizationSettingByOrgID(ctx context.Context, organizationID uuid.UUID) (*dto.OrganizationSettingResponse, error)
	CreateOrganizationSetting(ctx context.Context, setting *entities.OrganizationSetting) error
	GenerateOverview(ctx context.Context, userID uuid.UUID, orgID *uuid.UUID) (interface{}, error)
}

type service struct {
	repo               repository.UserCRUDService
	generalSettingRepo repository.GeneralSettingRepository
	orgRepo            repository.OrganizationRepositoryService
	orgSettingRepo     repository.OrganizationSettingRepository
	formRepo           repository.FormRepositoryService
}

func NewService(
	repo repository.UserCRUDService,
	generalSettingRepo repository.GeneralSettingRepository,
	orgRepo repository.OrganizationRepositoryService,
	orgSettingRepo repository.OrganizationSettingRepository,
	formRepo repository.FormRepositoryService,
) Service {
	return &service{
		repo:               repo,
		generalSettingRepo: generalSettingRepo,
		orgRepo:            orgRepo,
		orgSettingRepo:     orgSettingRepo,
		formRepo:           formRepo,
	}
}

func (s *service) UpdateUserProfile(ctx context.Context, user *entities.User) error {
	return s.repo.UpdateUser(ctx, user)
}

func (s *service) GetUserProfile(ctx context.Context, id uuid.UUID) (*dto.UserResponseWithDetails, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *service) CreateGeneralSetting(ctx context.Context, setting *dto.GeneralSettingRequest) error {
	existingSetting, err := s.generalSettingRepo.GetAllGeneralSetting() // Method to get the existing record
	if err != nil {
		return errcom.ErrRecordNotFounds
	}

	if existingSetting != nil {
		// If a record already exists, update it instead of creating a new one
		return errcom.ErrGeneralSettingAlreadyExists
	}
	// Map string to int
	fmt.Println(common.ENUM_TO_HASH, setting.MaxProjectDocketSizeUnit)
	unitMap := common.ENUM_TO_HASH["MaxProjectDocketSizeUnit"]

	unitInt, ok := unitMap[setting.MaxProjectDocketSizeUnit]
	fmt.Println(unitInt, ok)
	if !ok {
		return errcom.ErrInvalidMaxProjectDocketSizeUnit
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
		return nil, errcom.ErrInvalidMaxProjectDocketSizeUnit
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
		return nil, errcom.ErrUnabletoUpdate
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

func (s *service) GetAllGeneralSettings(ctx context.Context) (*dto.GeneralSettingResponse, error) {
	// Fetch the single general setting record
	setting, err := s.generalSettingRepo.GetAllGeneralSetting() // This will return a single record now
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, errcom.ErrRecordNotFounds // Handle case when no setting is found
	}

	// Map the integer value to a string unit using the enum
	unitEnum := common.HASH_TO_ENUM["MaxProjectDocketSizeUnit"][setting.MaxProjectDocketSizeUnit] // No need to use len() here
	if unitEnum == "" {
		unitEnum = "UNKNOWN"
	}

	// Return the mapped GeneralSettingResponse
	response := &dto.GeneralSettingResponse{
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

func (s *service) CreateOrganizationSetting(ctx context.Context, setting *entities.OrganizationSetting) error {
	// Check if the organization setting already exists
	existingSetting, err := s.orgSettingRepo.GetOrganizationSettingByOrgID(ctx, setting.OrgID.String()) // Convert OrgID to string
	if err == nil && existingSetting != nil {
		return errcom.ErrOrganizationSettingExists // Using fmt.Errorf instead of errors.New
	}
	return s.orgSettingRepo.CreateOrganizationSetting(ctx, setting)
}

func (s *service) UpdateOrganizationSettingByOrgID(ctx context.Context, setting *dto.OrganizationSettingRequest) error {
	entity := &entities.OrganizationSetting{
		ID:                       setting.ID,
		OrgID:                    setting.OrgID,
		DefaultDeletionDays:      setting.DefaultDeletionDays,
		DefaultArchivingDays:     setting.DefaultArchivingDays,
		MaxActiveProjects:        setting.MaxActiveProjects,
		MaxUsersPerOrganization:  setting.MaxUsersPerOrganization,
		MaxProjectDocketSize:     setting.MaxProjectDocketSize,
		MaxProjectDocketSizeUnit: setting.MaxProjectDocketSizeUnit,
		ScheduledEvaluationTime:  setting.ScheduledEvaluationTime,
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
		OrgID:                    setting.OrgID,
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

func (s *service) GenerateOverview(ctx context.Context, userID uuid.UUID, orgID *uuid.UUID) (interface{}, error) {
	userDetails, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errcom.ErrRecordNotFounds
	}
	fmt.Printf("userDetails", userDetails)

	role := userDetails.Role.Name // assuming it's part of dto.UserResponseWithDetails
	// return role

	// Now based on role, return KPIs
	switch role {
	case "SuperAdmin":
		orgs, total, lastMonth, thisYear, err := s.orgRepo.GetAllNonSingHealthOrganizationsWithCounts()
		if err != nil {
			return nil, errcom.ErrRecordNotFounds
		}
		totalCount, lastMonthCount, thisYearCount, _, err := s.repo.GetActiveUserCounts(ctx, nil)
		if err != nil {
			return nil, errcom.ErrRecordNotFounds
		}
		total, lastMonth, thisYear, _, _, _, _, latestForms, err := s.formRepo.GetProjectStatsByType(ctx, 3, uuidToStringPtr(orgID))
		if err != nil {
			return nil, errcom.ErrRecordNotFounds
		}

		return map[string]interface{}{
			"organization": map[string]interface{}{
				"items":       orgs,
				"total_count": total,
				"last_month":  lastMonth,
				"this_year":   thisYear,
			},
			"users": map[string]interface{}{
				"total_count": totalCount,
				"last_month":  lastMonthCount,
				"this_year":   thisYearCount,
			},
			"Project": map[string]interface{}{
				"items":       latestForms,
				"total_count": total,
				"last_month":  lastMonth,
				"this_year":   thisYear,
			},
		}, nil

	case "Collaborator":
		return map[string]interface{}{
			"dashboard": "Collaborator KPIs here",
			"role":      role,
		}, nil
	case "Admin":
		totalCount, lastMonthCount, thisYearCount, userList, err := s.repo.GetActiveUserCounts(ctx, uuidToStringPtr(orgID))
		if err != nil {
			return nil, errcom.ErrRecordNotFounds
		}
		total, lastMonth, thisYear, _, _, _, _, latestForms, err := s.formRepo.GetProjectStatsByType(ctx, 3, uuidToStringPtr(orgID))
		if err != nil {
			return nil, errcom.ErrRecordNotFounds
		}

		return map[string]interface{}{
			"users": map[string]interface{}{
				"items":       userList,
				"total_count": totalCount,
				"last_month":  lastMonthCount,
				"this_year":   thisYearCount,
			},
			"Project": map[string]interface{}{
				"items":       latestForms,
				"total_count": total,
				"last_month":  lastMonth,
				"this_year":   thisYear,
			},
		}, nil
	case "User":
		_, _, _, active, archived, pending, rejected, latestForms, err := s.formRepo.GetProjectStatsByType(ctx, 3, uuidToStringPtr(orgID))
		if err != nil {
			return nil, errcom.ErrRecordNotFounds
		}
		return map[string]interface{}{
			"Project": map[string]interface{}{
				"items":          latestForms,
				"active_count":   active,
				"archive_count":  archived,
				"pending_count":  pending,
				"rejected_count": rejected,
			},
		}, nil

	default:
		return map[string]interface{}{
			"dashboard": "Basic KPIs here",
			"role":      role,
		}, nil
	}
}

func uuidToStringPtr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}
