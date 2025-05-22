package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	errcom "github.com/PecozQ/aimx-library/apperrors"

	"github.com/go-kit/kit/endpoint"
)

// Error middleware handler...
func ErrorHandlingMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		response, err := next(ctx, request)
		if err != nil {
			appErr := errcom.FromError(err)
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

		allowedOrigins := map[string]bool{
			"http://localhost:3000":     true,
			"http://54.251.96.179:3000": true,
			"http://13.229.196.7:3000":  true,
		}

		fmt.Printf("CORS check - Origin: %s, Allowed: %v\n", origin, allowedOrigins[origin])

		// if allowedOrigins[origin] {
		// 	w.Header().Set("Access-Control-Allow-Origin", origin)
		// 	w.Header().Set("Access-Control-Allow-Credentials", "true") // ✅ REQUIRED for tokens
		// 	w.Header().Set("Vary", "Origin")
		// }
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true") // ✅ REQUIRED for tokens

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") // ✅ includes Authorization

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
