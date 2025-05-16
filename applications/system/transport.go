package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PecozQ/aimx-library/common"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
)

// MakeHTTPHandler configures HTTP handlers for the endpoints.
func MakeHTTPHandler(endpoints Endpoints) http.Handler {
	r := gin.New()
	r.Use(gin.Logger())

	// Base router group: /api/v1
	router := r.Group(fmt.Sprintf("%s/%s", common.BasePath, common.Version))

	// Notification endpoint
	notificationAPI := router.Group("/notifications")
	{
		notificationAPI.POST("/post-message", gin.WrapF(httptransport.NewServer(
			endpoints.SendNotificationEndpoint,
			decodeSendNotificationRequest,
			encodeResponse,
		).ServeHTTP))

		notificationAPI.POST("/update-token", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateFirebaseTokenEndpoint,
			decodeUpdateFirebaseTokenRequest,
			encodeResponse,
		).ServeHTTP))

	}

	return r
}

// decodeSendNotificationRequest decodes the HTTP request for sending a notification.
func decodeSendNotificationRequest(_ context.Context, r *http.Request) (interface{}, error) {
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
