package base

import (
	"context"
	"fmt"
	"time"

	"github.com/PecozQ/aimx-library/domain/dto"

	"whatsdare.com/fullstack/aimx/backend/common"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"

	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	//User and Roles
	CreateUserEndpoint   endpoint.Endpoint
	verifyOTPEndpoint    endpoint.Endpoint
	SendQREndpoint       endpoint.Endpoint
	SendQRVerifyEndpoint endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		//UserAndRole
		CreateUserEndpoint:   Middleware(makeCreateUserEndpoint(s), common.TimeoutMs),
		verifyOTPEndpoint:    Middleware(makeVerifyotpEndpoint(s), common.TimeoutMs),
		SendQREndpoint:       Middleware(makeSendQREndpoint(s), common.TimeoutMs),
		SendQRVerifyEndpoint: Middleware(makeSendQRVerifyEndpoint(s), common.TimeoutMs),
	}
}
func Middleware(endpoint endpoint.Endpoint, timeout time.Duration) endpoint.Endpoint {
	return service.ErrorHandlingMiddleware(service.TimeoutMiddleware(5 * timeout)(endpoint))
}

func makeCreateUserEndpoint(s service.Service) endpoint.Endpoint {
	fmt.Println("after decode makeCreateUserEndpoint")
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		fmt.Println("after decode makeCreateUserEndpoint", request)
		req := request.(*dto.UserAuthRequest)
		res, err := s.SendEmailOTP(ctx, req)
		if err != nil {
			return nil, err
		}
		return model.Response{Message: res.Message}, nil
	}
}

func makeVerifyotpEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		fmt.Println("after decode makeCreateUserEndpoint", &request)
		req := request.(*dto.UserAuthDetail)
		res, err := s.VerifyOTP(ctx, req)
		if err != nil {
			return nil, err
		}
		return model.Response{Message: res.Message}, nil
	}
}

func makeSendQREndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		fmt.Println("after decode makeCreateUserEndpoint", &request)
		req := request.(*dto.UserAuthDetail)
		res, err := s.RegisterAuth(ctx, req)
		if err != nil {
			return nil, err
		}
		return model.UserAuthResponse{QRImage: res.QRImage, QRURL: res.QRURL}, nil
	}
}

func makeSendQRVerifyEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		fmt.Println("after decode makeCreateUserEndpoint", &request)
		req := request.(*dto.UserAuthDetail)
		res, err := s.VerifyTOTP(ctx, req)
		if err != nil {
			return nil, err
		}
		return model.Response{Message: res.Message}, nil
	}
}
