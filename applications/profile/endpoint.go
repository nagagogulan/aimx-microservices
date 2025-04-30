package base

import (
	"context"
	"fmt"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/go-kit/kit/endpoint"
	"github.com/gofrs/uuid"
	"whatsdare.com/fullstack/aimx/backend/service"
)

type Endpoints struct {
	GetUserProfileEndpoint    endpoint.Endpoint
	UpdateUserProfileEndpoint endpoint.Endpoint

	CreateGeneralSettingEndpoint             endpoint.Endpoint
	UpdateGeneralSettingEndpoint             endpoint.Endpoint
	GetAllGeneralSettingEndpoint             endpoint.Endpoint
	GetAllNonSingHealthOrganizationsEndpoint endpoint.Endpoint
	UpdateOrganizationSettingByOrgIDEndpoint endpoint.Endpoint
    GetOrganizationSettingByOrgIDEndpoint    endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		GetUserProfileEndpoint:                   makeGetUserProfileEndpoint(s),
		UpdateUserProfileEndpoint:                makeUpdateUserProfileEndpoint(s),
		CreateGeneralSettingEndpoint:             makeCreateGeneralSettingEndpoint(s),
		UpdateGeneralSettingEndpoint:             makeUpdateGeneralSettingEndpoint(s),
		GetAllGeneralSettingEndpoint:             makeGetAllGeneralSettingEndpoint(s),
		GetAllNonSingHealthOrganizationsEndpoint: makeGetAllNonSingHealthOrganizationsEndpoint(s),
		UpdateOrganizationSettingByOrgIDEndpoint: MakeUpdateOrganizationSettingByOrgIDEndpoint(s),
        GetOrganizationSettingByOrgIDEndpoint: MakeGetOrganizationSettingByOrgIDEndpoint(s),

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
			return nil, fmt.Errorf("invalid UUID: %v", err)
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
        err := s.UpdateOrganizationSettingByOrgID(ctx, req)  // <-- Match service signature
        return map[string]string{"message": "Updated successfully"}, err
    }
}

func MakeGetOrganizationSettingByOrgIDEndpoint(s service.Service) endpoint.Endpoint {
    return func(ctx context.Context, request interface{}) (interface{}, error) {
        req := request.(*dto.OrganizationSettingRequest)
        setting, err := s.GetOrganizationSettingByOrgID(ctx, req.OrgID)
        return setting, err
    }
}