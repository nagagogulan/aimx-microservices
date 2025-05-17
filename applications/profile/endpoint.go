package base

import (
	"context"
	"fmt"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/go-kit/kit/endpoint"
	"github.com/gofrs/uuid"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"
)

type Endpoints struct {
	GetUserProfileEndpoint     endpoint.Endpoint
	UpdateUserProfileEndpoint  endpoint.Endpoint
	UpdateProfileImageEndpoint endpoint.Endpoint

	CreateGeneralSettingEndpoint             endpoint.Endpoint
	UpdateGeneralSettingEndpoint             endpoint.Endpoint
	GetAllGeneralSettingEndpoint             endpoint.Endpoint
	GetAllNonSingHealthOrganizationsEndpoint endpoint.Endpoint
	UpdateOrganizationSettingByOrgIDEndpoint endpoint.Endpoint
	GetOrganizationSettingByOrgIDEndpoint    endpoint.Endpoint
	CreateOrganizationSettingEndpoint        endpoint.Endpoint
	OverviewEndpoint                         endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		GetUserProfileEndpoint:                   makeGetUserProfileEndpoint(s),
		UpdateUserProfileEndpoint:                makeUpdateUserProfileEndpoint(s),
		UpdateProfileImageEndpoint:               makeUploadProfileImageEndpoint(s),
		CreateGeneralSettingEndpoint:             makeCreateGeneralSettingEndpoint(s),
		UpdateGeneralSettingEndpoint:             makeUpdateGeneralSettingEndpoint(s),
		GetAllGeneralSettingEndpoint:             makeGetAllGeneralSettingEndpoint(s),
		GetAllNonSingHealthOrganizationsEndpoint: makeGetAllNonSingHealthOrganizationsEndpoint(s),
		UpdateOrganizationSettingByOrgIDEndpoint: MakeUpdateOrganizationSettingByOrgIDEndpoint(s),
		GetOrganizationSettingByOrgIDEndpoint:    MakeGetOrganizationSettingByOrgIDEndpoint(s),
		CreateOrganizationSettingEndpoint:        makeCreateOrganizationSettingEndpoint(s),
		OverviewEndpoint:                         makeOverviewEndpoint(s),
	}
}

func makeGetUserProfileEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		idStr, ok := request.(string)
		if !ok {
			return nil, fmt.Errorf("invalid ID format")
		}

		id, err := uuid.FromString(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID")
		}

		return s.GetUserProfile(ctx, id)

	}
}

func makeUpdateUserProfileEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := request.(*entities.User)
		err := s.UpdateUserProfile(ctx, user)
		if err != nil {
			return nil, err
		}
		return map[string]string{"message": "Profile updated successfully"}, nil
	}
}

func makeCreateGeneralSettingEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.GeneralSettingRequest)
		err := s.CreateGeneralSetting(ctx, req)
		if err != nil {
			// return nil, err
			return nil, err

		}
		return map[string]string{"message": "Successfully added"}, nil
	}
}

func makeUpdateGeneralSettingEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.GeneralSettingRequest)

		response, err := s.UpdateGeneralSetting(ctx, req) // <-- response + error
		if err != nil {
			return nil, err
		}

		return response, nil // return full updated object
	}
}

func makeGetAllGeneralSettingEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return s.GetAllGeneralSettings(ctx)
	}
}

func makeGetAllNonSingHealthOrganizationsEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return s.GetAllNonSingHealthOrganizations(ctx)
	}
}

func MakeUpdateOrganizationSettingByOrgIDEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.OrganizationSettingRequest)
		err := s.UpdateOrganizationSettingByOrgID(ctx, req) // <-- Match service signature
		return map[string]string{"message": "Updated successfully"}, err
	}
}

func MakeGetOrganizationSettingByOrgIDEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.OrganizationSettingRequest)
		setting, err := s.GetOrganizationSettingByOrgID(ctx, req.OrgID)
		if err != nil {
			return nil, err

		}
		return setting, err
	}
}

func makeCreateOrganizationSettingEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.OrganizationSettingRequest)
		setting := &entities.OrganizationSetting{
			OrgID:                    req.OrgID,
			DefaultDeletionDays:      req.DefaultDeletionDays,
			DefaultArchivingDays:     req.DefaultArchivingDays,
			MaxActiveProjects:        req.MaxActiveProjects,
			MaxUsersPerOrganization:  req.MaxUsersPerOrganization,
			MaxProjectDocketSize:     req.MaxProjectDocketSize,
			MaxProjectDocketSizeUnit: req.MaxProjectDocketSizeUnit,
			ScheduledEvaluationTime:  req.ScheduledEvaluationTime,
		}
		err := s.CreateOrganizationSetting(ctx, setting)
		if err != nil {
			return nil, err
		}
		return map[string]string{"message": "Organization setting created successfully"}, nil
	}
}

func makeOverviewEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(*dto.OverviewRequest)
		if !ok {
			return nil, fmt.Errorf("invalid request")
		}

		userID, err := uuid.FromString(req.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid user_id UUID: %w", err)
		}

		var orgID *uuid.UUID
		if req.OrgID != "" {
			parsedOrgID, err := uuid.FromString(req.OrgID)
			if err != nil {
				return nil, fmt.Errorf("invalid organization_id UUID: %w", err)
			}
			orgID = &parsedOrgID
		}

		return s.GenerateOverview(ctx, userID, orgID)
	}
}
func makeUploadProfileImageEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*model.UploadProfileImageRequest)

		res, err := s.UploadProfileImage(ctx, req.UserID, req.FileHeader)
		if err != nil {
			return nil, err
		}

		// Wrap and return the final JSON response
		return &model.UploadProfileImageResponse{
			Message:   res.Message,
			ImagePath: res.ImagePath,
		}, nil
	}
}
