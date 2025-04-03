package service

import (
	"context"
	"log"
	"time"

	"github.com/go-kit/kit/endpoint"
)

// Error middleware handler...
func ErrorHandlingMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		response, err := next(ctx, request)
		if err != nil {
			// appErr := FromError(err)
			return nil, err
		}
		return response, nil
	}
}

func TimeoutMiddleware(d time.Duration) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			// Create a new context with timeout
			ctx, cancel := context.WithTimeout(ctx, d)
			defer cancel()

			// Start processing the request...
			start := time.Now()
			response, err := next(ctx, request)
			duration := time.Since(start)

			if duration > d {
				// Log only if it exceeds timeout duration...
				log.Printf("Request timed out after %v", duration)
				// return nil, NewCustomError(common.ErrRequestTimeOut, err)
			}

			// Log duration for long-running requests...
			if duration > time.Second {
				log.Printf("Request processed in %v", duration)
			}
			return response, err
		}
	}
}

type RequestWithContext struct {
	Ctx     context.Context
	Request interface{}
}
