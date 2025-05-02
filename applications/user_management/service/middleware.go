package service

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
)

// Error middleware handler...
func ErrorHandlingMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		response, err := next(ctx, request)
		if err != nil {
			appErr := FromError(err)
			return nil, appErr
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
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			// If you want to allow specific origins only, replace "*" with origin whitelisting logic
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		// Recommended settings for modern apps
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
