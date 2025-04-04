package base

import (
	"context"
	"time"

	"github.com/PecozQ/aimx-library/domain/dto"
	middleware "github.com/PecozQ/aimx-library/middleware"

	"whatsdare.com/fullstack/aimx/backend/common"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"

	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	//User and Roles
	CreateUserEndpoint endpoint.Endpoint
	verifyOTPEndpoint  endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		//UserAndRole
		CreateUserEndpoint: Middleware(makeCreateUserEndpoint(s), common.TimeoutMs),
		verifyOTPEndpoint:  Middleware(makeVerifyotpEndpoint(s), common.TimeoutMs),
	}
}
func Middleware(endpoint endpoint.Endpoint, timeout time.Duration) endpoint.Endpoint {
	return service.ErrorHandlingMiddleware(service.TimeoutMiddleware(5 * timeout)(endpoint))
}

func makeCreateUserEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		reqWithContext := request.(middleware.RequestWithContext)
		req := reqWithContext.Request.(dto.UserAuthRequest)
		res, err := s.SendEmailOTP(reqWithContext.Ctx, req)
		if err != nil {
			return nil, err
		}
		return model.Response{Message: res.Message}, nil
	}
}

func makeVerifyotpEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		reqWithContext := request.(middleware.RequestWithContext)
		req := reqWithContext.Request.(dto.UserAuthdetail)
		res, err := s.VerifyOTP(reqWithContext.Ctx, &req)
		if err != nil {
			return nil, err
		}
		return model.Response{Message: res}, nil
	}
}
