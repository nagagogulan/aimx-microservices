package subscriber

import (
	"context"
	"mime/multipart"

	"github.com/go-kit/kit/endpoint"

	"whatsdare.com/fullstack/aimx/backend/service"
)

// Endpoints holds all Go kit endpoints for the subscriber service.
type Endpoints struct {
	UploadFileEndpoint endpoint.Endpoint
	TestKongEndpoint   endpoint.Endpoint
	// Add other endpoints here if any
}

// MakeEndpoints initializes all Go kit endpoints for the subscriber service.
// It currently only includes the UploadFile endpoint.
// Add other services as parameters if needed for other endpoints.
func MakeEndpoints(svc service.UploadService) Endpoints {
	return Endpoints{
		UploadFileEndpoint: MakeUploadFileEndpoint(svc),
		TestKongEndpoint:   MakeTestKongEndpoint(svc),
	}
}

// MakeUploadFileEndpoint creates an endpoint for the UploadFile method of the UploadService.
func MakeUploadFileEndpoint(svc service.UploadService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		// The request is expected to be *multipart.FileHeader directly from the transport layer
		// for large file uploads, to avoid reading the whole file into memory here.
		fileHeader, ok := request.(*multipart.FileHeader)
		if !ok {
			// This case should ideally be handled by a DecodeRequestFunc in transport,
			// but as a fallback or if transport passes it as generic interface{}.
			return nil, &InvalidRequestTypeError{Expected: "*multipart.FileHeader", Actual: request}
		}
		return svc.UploadFile(ctx, fileHeader)
	}
}

// InvalidRequestTypeError is returned when the request type is not as expected.
type InvalidRequestTypeError struct {
	Expected string
	Actual   interface{}
}

func (e *InvalidRequestTypeError) Error() string {
	return "invalid request type" // Simplified error message
	// return fmt.Sprintf("invalid request type: expected %s, got %T", e.Expected, e.Actual)
}

// MakeTestKongEndpoint creates an endpoint for the TestKong method of the UploadService.
func MakeTestKongEndpoint(svc service.UploadService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		// No request processing needed for this endpoint
		return svc.TestKong(ctx)
	}
}

// Add other endpoint creators here if you have more services/methods
// For example:
// type SomeRequest struct{ ... }
// type SomeResponse struct{ ... }
// func MakeSomeOtherEndpoint(svc service.SomeOtherService) endpoint.Endpoint {
// 	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
// 		req := request.(SomeRequest)
// 		// res, err := svc.SomeMethod(ctx, req.Field)
// 		// return SomeResponse{Result: res}, err
// 		return nil, nil // Placeholder
// 	}
// }
