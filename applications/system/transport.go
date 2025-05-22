package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/middleware"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	"whatsdare.com/fullstack/aimx/backend/model"
)

// MakeHTTPHandler configures HTTP handlers for the endpoints.
func MakeHTTPHandler(endpoints Endpoints) http.Handler {
	options := []httptransport.ServerOption{httptransport.ServerErrorEncoder(errcom.EncodeError)}
	r := gin.New()
	r.Use(gin.Logger())

	// Base router group: /api/v1
	router := r.Group(fmt.Sprintf("%s/%s/%s", common.BasePath, common.Version, "system"))

	// Notification endpoint
	notificationAPI := router.Group("/notifications")
	{
		notificationAPI.POST("/post-message", gin.WrapF(httptransport.NewServer(
			endpoints.SendNotificationEndpoint,
			decodeSendNotificationRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

		notificationAPI.POST("/update-token", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateFirebaseTokenEndpoint,
			decodeUpdateFirebaseTokenRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

	}
	auditLogAPI := router.Group("/audit-logs")
	{
		auditLogAPI.POST("/create", gin.WrapF(httptransport.NewServer(
			endpoints.AuditLogsEndpoint,
			decodeAuditLogsRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

		auditLogAPI.GET("/get", gin.WrapF(httptransport.NewServer(
			endpoints.GetAuditLogEndpoint,
			decodeGetAuditLogRequest,
			encodeResponse,
			options...,
		).ServeHTTP))
		auditLogAPI.GET("/findlogs", gin.WrapF(httptransport.NewServer(
			endpoints.FindAuditLogByUserEndpoint,
			decodeFindAuditLogByUserRequest,
			encodeResponse,
			options...,
		).ServeHTTP))
	}

	// Test endpoint for Kong
	router.GET("/test", gin.WrapF(httptransport.NewServer(
		endpoints.TestKongEndpoint,
		decodeTestKongRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	return r
}

// decodeSendNotificationRequest decodes the HTTP request for sending a notification.
func decodeSendNotificationRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, errs := middleware.DecodeHeaderGetClaims(r)
	if errs != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req struct {
		UserID  string `json:"user_id"`
		Message string `json:"message"`
	}

	// Decode JSON request body into `req`
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request body: %v", err)
	}

	return map[string]interface{}{
		"user_id": req.UserID,
		"message": req.Message,
	}, nil
}

// encodeResponse encodes the response into a JSON format.
func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

func decodeUpdateFirebaseTokenRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, errs := middleware.DecodeHeaderGetClaims(r)
	if errs != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req struct {
		UserID        string `json:"user_id"`
		FirebaseToken string `json:"firebase_token"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %v", err)
	}
	return map[string]interface{}{
		"user_id":        req.UserID,
		"firebase_token": req.FirebaseToken,
	}, nil
}

func decodeAuditLogsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	// _, errs := middleware.DecodeHeaderGetClaims(r)
	// if errs != nil {
	// 	return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	// }
	var req dto.AuditLogs
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %v", err)
	}
	return map[string]interface{}{
		"audit_log": &req,
	}, nil
}

func decodeGetAuditLogRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, errs := middleware.DecodeHeaderGetClaims(r)
	if errs != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	role := r.URL.Query().Get("role")
	orgID := r.URL.Query().Get("org_id")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if role == "" {
		return nil, fmt.Errorf("role is required")
	}
	return map[string]interface{}{
		"role":   role,
		"org_id": orgID,
		"page":   page,
		"limit":  limit,
	}, nil
}

func decodeFindAuditLogByUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	q := r.URL.Query()
	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	}

	return &model.FindAuditByUserRequest{
		UserName: q.Get("user_name"),
		Page:   page,
		Limit:  limit,
	}, nil
}

func decodeTestKongRequest(_ context.Context, r *http.Request) (interface{}, error) {
	// No request body needed for this endpoint
	return nil, nil
}
