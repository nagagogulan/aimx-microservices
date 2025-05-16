package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	middleware "github.com/PecozQ/aimx-library/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gofrs/uuid"
)

func MakeHTTPHandler(endpoints Endpoints) http.Handler {
	options := []httptransport.ServerOption{httptransport.ServerErrorEncoder(errcom.EncodeError)}
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://54.251.209.147:3000", "http://localhost:3000", "http://13.229.196.7:3000"}, // Replace with your frontend's origin
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	api := r.Group(fmt.Sprintf("%s/%s/request", common.BasePath, common.Version))
	{
		api.POST("/", gin.WrapF(httptransport.NewServer(
			endpoints.CreateRequestEndpoint,
			decodeCreateRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

		api.PUT("/status", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateRequestStatusEndpoint,
			decodeUpdateRequestStatus,
			encodeResponse,
			options...,
		).ServeHTTP))

		api.GET("/org", gin.WrapF(httptransport.NewServer(
			endpoints.GetRequestsByOrgEndpoint,
			decodeGetRequestsByOrgRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

		api.GET("/", gin.WrapF(httptransport.NewServer(
			endpoints.GetAllRequestsEndpoint,
			decodeGetAllRequestsRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

		api.GET("/:id", gin.WrapF(httptransport.NewServer(
			endpoints.GetRequestByIDEndpoint,
			decodeIDFromPath,
			encodeResponse,
			options...,
		).ServeHTTP))

		api.GET("/request-types", gin.WrapF(httptransport.NewServer(
			endpoints.ListRequestTypes,
			decodeEmptyRequest,
			encodeResponse,
			options...,
		).ServeHTTP))
	}

	return r
}

func decodeCreateRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.CreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeUpdateRequestStatus(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	fmt.Println("called transport")
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return nil, fmt.Errorf("missing id parameter")
	}

	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format for id: %w", err)
	}

	var dto dto.UpdateRequestStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "dto": &dto}, nil
}

// func decodeOrgIDFromQuery(_ context.Context, r *http.Request) (interface{}, error) {
// 	idStr := r.URL.Query().Get("org_id")
// 	return uuid.FromString(idStr)
// }

func decodeEmptyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	return nil, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

func decodeIDFromPath(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	return uuid.FromString(idStr)
}

func decodeGetRequestsByOrgRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	// Parse the query parameters from the request URL
	orgIDStr := r.URL.Query().Get("org_id")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	search := r.URL.Query().Get("search")
	reqType := r.URL.Query().Get("type")

	// Convert orgID to uuid
	orgID, err := uuid.FromString(orgIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid org_id format: %w", err)
	}

	// Convert page and limit to integers
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1 // Default to 1 if the page is invalid
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10 // Default to 10 items per page if limit is invalid
	}

	filters := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		// Exclude non-filter keys
		if key != "org_id" && key != "page" && key != "limit" && key != "search" {
			// Handling filters, e.g., for "status" and "request_type"
			// Attempt to convert certain filters to expected types, like integers
			if key == "status" || key == "request_type" {
				if len(values) > 0 {
					// Assuming filters are passed as integers in the query string
					if filterVal, err := strconv.Atoi(values[0]); err == nil {
						filters[key] = filterVal
					} else {
						return nil, fmt.Errorf("invalid value for %s filter: must be an integer", key)
					}
				}
			} else {
				// For other filters, treat them as strings
				filters[key] = values[0]
			}
		}
	}

	// Create the request structure to pass to the service layer
	return map[string]interface{}{
		"org_id":  orgID,
		"page":    page,
		"limit":   limit,
		"search":  search,
		"filters": filters,
		"reqType": reqType,
	}, nil
}

func decodeGetAllRequestsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	// Parse the query parameters from the request URL
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	search := r.URL.Query().Get("search")

	// Convert page and limit to integers
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1 // Default to 1 if the page is invalid
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10 // Default to 10 items per page if limit is invalid
	}

	filters := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		// Exclude non-filter keys
		if key != "org_id" && key != "page" && key != "limit" && key != "search" {
			// Handling filters, e.g., for "status" and "request_type"
			// Attempt to convert certain filters to expected types, like integers
			if key == "status" || key == "request_type" {
				if len(values) > 0 {
					// Assuming filters are passed as integers in the query string
					if filterVal, err := strconv.Atoi(values[0]); err == nil {
						filters[key] = filterVal
					} else {
						return nil, fmt.Errorf("invalid value for %s filter: must be an integer", key)
					}
				}
			} else {
				// For other filters, treat them as strings
				filters[key] = values[0]
			}
		}
	}

	// Create the request structure to pass to the service layer
	return map[string]interface{}{
		"page":    page,
		"limit":   limit,
		"search":  search,
		"filters": filters,
	}, nil
}
