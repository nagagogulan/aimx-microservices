package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
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
	router.POST("/otpverify", gin.WrapF(httptransport.NewServer(
		endpoints.verifyOTPEndpoint,
		decodeVerifyUserRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	return r
}

// Decode register api request...
func decodeCreateUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request *dto.UserAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	fmt.Println("after decode", request)
	// Extract Gin context
	return &dto.UserAuthRequest{Email: request.Email}, nil
}
func decodeVerifyUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dto.UserAuthDetail
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	// Extract Gin context
	return &dto.UserAuthDetail{Email: request.Email, OTP: request.OTP}, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
