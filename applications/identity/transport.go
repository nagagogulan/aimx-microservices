package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	middleware "github.com/PecozQ/aimx-library/middleware"
	"whatsdare.com/fullstack/aimx/backend/common"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"

	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
)

func MakeHTTPHandler(s service.Service) http.Handler {
	fmt.Println("connect http handuler")
	options := []httptransport.ServerOption{}

	r := gin.New()
	endpoints := NewEndpoint(s)

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	router := r.Group(fmt.Sprintf("%s/%s", common.BasePath, common.Version))

	//Register and Login Endpoints...
	router.POST("/login", gin.WrapF(httptransport.NewServer(
		endpoints.CreateUserEndpoint,
		decodeCreateUserRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	router.POST("/verify", gin.WrapF(httptransport.NewServer(
		endpoints.verifyOTPEndpoint,
		decodeVerifyUserRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	return r
}

// Decode register api request...
func decodeCreateUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request model.UserAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	// Extract Gin context
	return middleware.RequestWithContext{Request: request}, nil
}
func decodeVerifyUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request model.UserAuthdetail
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	// Extract Gin context
	return middleware.RequestWithContext{Request: request}, nil
}
func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
