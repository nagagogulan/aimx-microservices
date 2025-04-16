package base

import (
	"context"
	"fmt"
	"time"

	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/go-kit/kit/endpoint"
	"whatsdare.com/fullstack/aimx/backend/service"

	"errors"

	errcom "github.com/PecozQ/aimx-library/apperrors"

	"whatsdare.com/fullstack/aimx/backend/model"
)

type Endpoints struct {
	CreateTemplateEndpoint  endpoint.Endpoint
	GetTemplateByIDEndpoint endpoint.Endpoint
	UpdateTemplateEndpoint  endpoint.Endpoint
	DeleteTemplateEndpoint  endpoint.Endpoint

	CreateFormEndpoint    endpoint.Endpoint
	GetFormByTypeEndpoint endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		CreateTemplateEndpoint:  Middleware(makeCreateTemplateEndpoint(s), commonlib.TimeoutMs),
		GetTemplateByIDEndpoint: Middleware(makeGetTemplateByTypeEndpoint(s), commonlib.TimeoutMs),
		UpdateTemplateEndpoint:  Middleware(makeUpdateTemplateEndpoint(s), commonlib.TimeoutMs),
		DeleteTemplateEndpoint:  Middleware(makeDeleteTemplateEndpoint(s), commonlib.TimeoutMs),
		// GetTemplateByIDEndpoint: Middleware(makeGetTemplateByIDEndpoint(s), common.TimeoutMs),

		CreateFormEndpoint:    Middleware(makeCreateFormEndpoint(s), commonlib.TimeoutMs),
		GetFormByTypeEndpoint: Middleware(makeGetFormByTypeEndpoint(s), commonlib.TimeoutMs),
	}
}

// Middlewares applies both error handling and timeout middleware to an endpoint...
func Middleware(endpoint endpoint.Endpoint, timeout time.Duration) endpoint.Endpoint {
	return service.ErrorHandlingMiddleware(service.TimeoutMiddleware(5 * timeout)(endpoint))
}

func makeCreateTemplateEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(entities.Template)
		template, err := s.CreateTemplate(ctx, req)
		if err != nil {
			return nil, err
		}
		return template, nil
		// return model.CreateUserResponse{Message: commonRepo.Create_Message, User: model.UserResponse{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, IsLocked: user.IsLocked, ProfileImage: user.ProfileImage, IsFirstLogin: user.IsFirstLogin, Role: model.UserRole{ID: role.ID, Name: role.Name}, RolePermission: user.RolePermissions}}, nil
	}
}
func makeGetTemplateByTypeEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req, ok := request.(*model.ParamRequest)
		if !ok {
			return nil, errors.New("params error")
		}

		if req.ID != "" {
			// If ID is present, prioritize lookup by ID
			template, err := s.GetTemplateByType(ctx, 0, req.ID)
			if err != nil {
				return nil, err // or wrap as needed
			}
			return template, nil
		}

		if req.Type != 0 {
			// If ID is not present, use Type
			template, err := s.GetTemplateByType(ctx, req.Type, "")
			if err != nil {
				return nil, errors.New("Template Not found") // or wrap as needed
			}
			return template, nil
		}

		return nil, errors.New("either ID or Type must be provided")
	}
}

func makeUpdateTemplateEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*model.TemplateRequest)
		fmt.Println("Template ID:", req.Template.ID.Hex())

		template, err := s.UpdateTemplate(ctx, req.Template.ID.Hex(), req.Template)
		if err != nil {
			return nil, err
		}
		return template, nil
	}
}
func makeDeleteTemplateEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(*model.ParamRequest)
		if !ok {
			return nil, errors.New("params error")
		}

		// If ID is present, prioritize lookup by ID
		res, err := s.DeleteTemplate(ctx, req.ID)
		if err != nil {
			return nil, err // or wrap as needed
		}

		return &model.Response{Message: res.Message}, nil
	}
}

func makeCreateFormEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.FormDTO)
		template, err := s.CreateForm(ctx, *req)
		if err != nil {
			return nil, err
		}
		return template, nil
		// return model.CreateUserResponse{Message: commonRepo.Create_Message, User: model.UserResponse{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, IsLocked: user.IsLocked, ProfileImage: user.ProfileImage, IsFirstLogin: user.IsFirstLogin, Role: model.UserRole{ID: role.ID, Name: role.Name}, RolePermission: user.RolePermissions}}, nil
	}
}

func makeGetFormByTypeEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req, ok := request.(*model.ParamRequest)
		if !ok {
			return nil, errors.New("params error")
		}

		if commonlib.IsEmpty(req) {
			return nil, errors.New("Type must be provided")
		}
		// if req.ID != "" {
		// 	// If ID is present, prioritize lookup by ID
		// 	template, err := s.GetTemplateByType(ctx, 0, req.ID)
		// 	if err != nil {
		// 		return nil, err // or wrap as needed
		// 	}
		// 	return template, nil
		// }
		formList, err := s.GetFormByType(ctx, req.Type)
		if err != nil {
			return nil, service.NewCustomError(errcom.ErrNotFound, err) // or wrap as needed
		}
		return formList, nil
	}
}
