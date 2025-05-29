package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"whatsdare.com/fullstack/aimx/backend/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
)

func MakeHTTPHandler(s service.Service) http.Handler {
	fmt.Println("connect http handuler")
	options := []httptransport.ServerOption{httptransport.ServerErrorEncoder(errcom.EncodeError)}

	r := gin.New()
	endpoints := NewEndpoint(s)

	// r.Use(func(c *gin.Context) {
	// 	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	// 	c.Next()
	// })
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://54.251.96.179:3000", "http://localhost:3000", "http://13.229.196.7:3000"}, // Replace with your frontend's origin
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	router := r.Group(fmt.Sprintf("%s/%s/%s", common.BasePath, common.Version, "identity"))

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
	router.POST("/totpverify", gin.WrapF(httptransport.NewServer(
		endpoints.SendQRVerifyEndpoint,
		decodeVerifyTOTPUserRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	router.POST("/refresh-token", gin.WrapH(httptransport.NewServer(
		endpoints.RefreshTokenEndpoint,
		decodeRefreshTokenRequest,
		encodeResponse,
		options...,
	)))

	// Search Organization endpoint
	router.GET("/searchOrganization", gin.WrapF(httptransport.NewServer(
		endpoints.SearchOrganizationEndpoint,
		decodeSearchOrganizationRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// Test endpoint for Kong
	router.GET("/test", gin.WrapF(httptransport.NewServer(
		endpoints.TestKongEndpoint,
		decodeTestKongRequest,
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
func decodeVerifyTOTPUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
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

func decodeRefreshTokenRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.RefreshAuthDetail

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeSearchOrganizationRequest(_ context.Context, r *http.Request) (interface{}, error) {
	// Get the searchTerm from query parameters
	searchTerm := r.URL.Query().Get("searchTerm")
	if searchTerm != "" {
		return searchTerm, nil
	}
	//searchTerm is coming empty so to  get all Organizations using added "all"
	return "all", nil
}

func decodeTestKongRequest(_ context.Context, r *http.Request) (interface{}, error) {
	// No request body needed for this endpoint
	return nil, nil
}
