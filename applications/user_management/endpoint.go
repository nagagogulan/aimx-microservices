package base

import (
	"context"
	"fmt"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/go-kit/kit/endpoint"
	"github.com/gofrs/uuid"
	"whatsdare.com/fullstack/aimx/backend/service"
)

type Endpoints struct {
	ListUsersEndpoint      endpoint.Endpoint
	DeleteUserEndpoint     endpoint.Endpoint
	DeactivateUserEndpoint endpoint.Endpoint
	ActivateUserEndpoint   endpoint.Endpoint
	TestKongEndpoint       endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		ListUsersEndpoint:      makeListUsersEndpoint(s),
		DeleteUserEndpoint:     makeDeleteUserEndpoint(s.DeleteUser),
		DeactivateUserEndpoint: makeDeactivateUserEndpoint(s.DeactivateUser),
		ActivateUserEndpoint:   makeActivateUserEndpoint(s.ActivateUser),
		TestKongEndpoint:       makeTestKongEndpoint(s),
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
				return nil, errcom.ErrInvalidOrganizationID
			}
			orgID = parsed
		default:
			return nil, errcom.ErrInvalidOrganizationID
		}
		userIDRaw := req["user_id"]
		var userID uuid.UUID
		switch v := userIDRaw.(type) {
		case uuid.UUID:
			userID = v
		case string:
			parsed, err := uuid.FromString(v)
			if err != nil {
				return nil, errcom.ErrInvalidUserID
			}
			userID = parsed
		default:
			return nil, errcom.ErrInvalidUserID
		}

		// Type assertion and fallback for pagination
		page := 1
		if p, ok := req["page"].(int); ok && p > 0 {
			page = p
		}

		limit := 10
		if l, ok := req["limit"].(int); ok && l >= 0 {
			limit = l
		}
		search := req["search"].(string)
		rawType, ok := req["type"].(string)
		if !ok {
			return nil, errcom.ErrInvalidReqType
		}
		// Capture filters from request
		filters := make(map[string]interface{})
		if filterParams, exists := req["filters"]; exists {
			filterMap, ok := filterParams.(map[string]interface{})
			if ok {
				filters = filterMap
			}
		}
		fmt.Printf("filters: %v\n", filters)

		return s.ListUsers(ctx, orgID, userID, page, limit, search, filters, rawType)
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
			return nil, errcom.ErrExpectedStringID
		}

		uid, err := uuid.FromString(id)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID")
		}

		if err := handler(ctx, uid); err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"status":  "success",
			"message": "User has been successfully deactivated",
		}, nil
	}
}

func makeActivateUserEndpoint(handler func(context.Context, uuid.UUID) error) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userID, ok := request.(string)
		if !ok {
			return nil, errcom.ErrExpectedStringID
		}

		uid, err := uuid.FromString(userID)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID")
		}

		if err := handler(ctx, uid); err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"status":  "success",
			"message": "User has been successfully activated",
		}, nil
	}
}

func makeTestKongEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// No request processing needed for this endpoint
		return s.TestKong(ctx)
	}
}
