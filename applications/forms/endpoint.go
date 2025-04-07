package base

import (
	"context"
	"time"

	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/go-kit/kit/endpoint"
	"whatsdare.com/fullstack/aimx/backend/service"

	"whatsdare.com/fullstack/aimx/backend/model"
)

type Endpoints struct {
	CreateTemplateEndpoint  endpoint.Endpoint
	GetTemplateByIDEndpoint endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		CreateTemplateEndpoint: Middleware(makeCreateTemplateEndpoint(s), commonlib.TimeoutMs),
		// GetTemplateByIDEndpoint: Middleware(makeGetTemplateByIDEndpoint(s), common.TimeoutMs),
	}
}

// Middlewares applies both error handling and timeout middleware to an endpoint...
func Middleware(endpoint endpoint.Endpoint, timeout time.Duration) endpoint.Endpoint {
	return service.ErrorHandlingMiddleware(service.TimeoutMiddleware(5 * timeout)(endpoint))
}

func makeCreateTemplateEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		reqWithContext := request.(service.RequestWithContext)
		req := reqWithContext.Request.(model.CreateTemplateRequest)
		template, err := s.CreateTemplate(reqWithContext.Ctx, req.Template)
		if err != nil {
			return nil, err
		}
		return template, nil
		// return model.CreateUserResponse{Message: commonRepo.Create_Message, User: model.UserResponse{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, IsLocked: user.IsLocked, ProfileImage: user.ProfileImage, IsFirstLogin: user.IsFirstLogin, Role: model.UserRole{ID: role.ID, Name: role.Name}, RolePermission: user.RolePermissions}}, nil
	}
}
