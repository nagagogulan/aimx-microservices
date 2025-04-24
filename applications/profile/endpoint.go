package base

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/PecozQ/aimx-library/domain/entities"
	"whatsdare.com/fullstack/aimx/backend/service"
	"github.com/gofrs/uuid"

)

type Endpoints struct {
	GetUserProfileEndpoint    endpoint.Endpoint
	UpdateUserProfileEndpoint endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		GetUserProfileEndpoint:    makeGetUserProfileEndpoint(s),
		UpdateUserProfileEndpoint: makeUpdateUserProfileEndpoint(s),
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
