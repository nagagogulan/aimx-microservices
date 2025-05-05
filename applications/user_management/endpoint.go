package base

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/gofrs/uuid"
	"whatsdare.com/fullstack/aimx/backend/service"
)

type Endpoints struct {
	ListUsersEndpoint      endpoint.Endpoint
	DeleteUserEndpoint     endpoint.Endpoint
	DeactivateUserEndpoint endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		ListUsersEndpoint:      makeListUsersEndpoint(s),
		DeleteUserEndpoint:     makeDeleteUserEndpoint(s.DeleteUser),
		DeactivateUserEndpoint: makeDeactivateUserEndpoint(s.DeactivateUser),
	}
}

func makeListUsersEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(map[string]interface{})
		orgIDRaw := req["organisation_id"]
		var orgID uuid.UUID
		switch v := orgIDRaw.(type) {
		case uuid.UUID:
			orgID = v
		case string:
			parsed, err := uuid.FromString(v)
			if err != nil {
				return nil, fmt.Errorf("invalid organization_id: %v", err)
			}
			orgID = parsed
		default:
			return nil, fmt.Errorf("organization_id missing or invalid")
		}
		userIDRaw := req["user_id"]
		var userID uuid.UUID
		switch v := userIDRaw.(type) {
		case uuid.UUID:
			userID = v
		case string:
			parsed, err := uuid.FromString(v)
			if err != nil {
				return nil, fmt.Errorf("invalid user_id: %v", err)
			}
			userID = parsed
		default:
			return nil, fmt.Errorf("user_id missing or invalid")
		}

		page := req["page"].(int)
		limit := req["limit"].(int)
		search := req["search"].(string)

		// Capture filters from request
		filters := make(map[string]interface{})
		if filterParams, exists := req["filters"]; exists {
			filterMap, ok := filterParams.(map[string]interface{})
			if ok {
				filters = filterMap
			}
		}
		fmt.Printf("filters", filters)

		return s.ListUsers(ctx, orgID, userID, page, limit, search, filters)
	}
}

func makeDeleteUserEndpoint(handler func(context.Context, uuid.UUID) error) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		err := handler(ctx, request.(uuid.UUID))
		return map[string]string{"message": "deleted"}, err
	}
}

func makeDeactivateUserEndpoint(handler func(context.Context, uuid.UUID) error) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		id, ok := request.(string)
		if !ok {
			return nil, fmt.Errorf("expected string ID but got %T", request)
		}

		uid, err := uuid.FromString(id)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID: %w", err)
		}

		if err := handler(ctx, uid); err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("User %s has been successfully deactivated", uid),
		}, nil
	}
}
