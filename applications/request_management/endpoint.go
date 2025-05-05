package base

import (
	"context"
	"fmt"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/go-kit/kit/endpoint"
	"github.com/gofrs/uuid"
	"whatsdare.com/fullstack/aimx/backend/service"
)

type Endpoints struct {
	CreateRequestEndpoint       endpoint.Endpoint
	UpdateRequestStatusEndpoint endpoint.Endpoint
	GetRequestsByOrgEndpoint    endpoint.Endpoint
	GetAllRequestsEndpoint      endpoint.Endpoint
	GetRequestByIDEndpoint endpoint.Endpoint
	ListRequestTypes endpoint.Endpoint

}

func NewEndpoint(s service.RequestService) Endpoints {
	return Endpoints{
		CreateRequestEndpoint:       MakeCreateRequestEndpoint(s),
		UpdateRequestStatusEndpoint: MakeUpdateRequestStatusEndpoint(s),
		GetRequestsByOrgEndpoint:    MakeGetRequestsByOrgEndpoint(s),
		GetAllRequestsEndpoint:      MakeGetAllRequestsEndpoint(s),
		GetRequestByIDEndpoint: MakeGetRequestByIDEndpoint(s),
		ListRequestTypes: makeListRequestTypesEndpoint(s),

	}
}

func MakeCreateRequestEndpoint(s service.RequestService) endpoint.Endpoint {
	return Middleware(func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.CreateRequestDTO)
		return map[string]string{"message": "created"}, s.CreateRequest(ctx, req)
	}, common.TimeoutMs)
}

func MakeUpdateRequestStatusEndpoint(s service.RequestService) endpoint.Endpoint {
	return Middleware(func(ctx context.Context, request interface{}) (interface{}, error) {
		fmt.Println("called in endpoint")
		input := request.(map[string]interface{})
		id := input["id"].(uuid.UUID)
		statusDTO := input["dto"].(*dto.UpdateRequestStatusDTO)
		return map[string]string{"message": "status updated"}, s.UpdateRequestStatus(ctx, id, statusDTO)
	}, common.TimeoutMs)
}

func MakeGetRequestsByOrgEndpoint(s service.RequestService) endpoint.Endpoint {
	return Middleware(func(ctx context.Context, request interface{}) (interface{}, error) {
		orgID := request.(uuid.UUID)
		return s.GetRequestsByOrg(ctx, orgID)
	}, common.TimeoutMs)
}

func MakeGetAllRequestsEndpoint(s service.RequestService) endpoint.Endpoint {
	return Middleware(func(ctx context.Context, request interface{}) (interface{}, error) {
		return s.GetAllRequests(ctx)
	}, common.TimeoutMs)
}

func MakeGetRequestByIDEndpoint(s service.RequestService) endpoint.Endpoint {
	return Middleware(func(ctx context.Context, request interface{}) (interface{}, error) {
		id := request.(uuid.UUID)
		return s.GetRequestByID(ctx, id)
	}, common.TimeoutMs)
}


func Middleware(ep endpoint.Endpoint, timeoutMs int) endpoint.Endpoint {
	return ep // Add timeout middleware if needed
}

func makeListRequestTypesEndpoint(s service.RequestService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// Call the service to get the list of request types
		requestTypes, err := s.ListRequestTypes(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch request types: %v", err)
		}
		return dto.ListRequestTypesResponse{
			RequestTypes: requestTypes,
		}, nil
	}
}