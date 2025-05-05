package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
)

func MakeHTTPHandler(endpoints Endpoints) http.Handler {
	r := gin.New()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://54.251.209.147:3000", "http://localhost:3000", "http://13.229.196.7:3000"}, // Replace with your frontend's origin
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Base router group: /api/v1
	router := r.Group(fmt.Sprintf("%s/%s", common.BasePath, common.Version))

	// Role
	api := router.Group("/profile")
	// r := gin.Default()
	// endpoints := NewEndpoint(s)

	// api := r.Group("/api/v1/profile")
	{
		api.GET("/:id", gin.WrapF(httptransport.NewServer(
			endpoints.GetUserProfileEndpoint,
			decodeUUIDParam, // This will now extract 'id' from the URL path
			encodeResponse,
		).ServeHTTP))

		// PUT: /profile
		api.PUT("/", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateUserProfileEndpoint,
			decodeUpdateUserRequest, // Decode body to user
			encodeResponse,
		).ServeHTTP))
	}

	// General Setting APIs
	general := router.Group("/general-setting")
	{
		general.POST("/", gin.WrapF(httptransport.NewServer(
			endpoints.CreateGeneralSettingEndpoint,
			decodeGeneralSettingRequest,
			encodeResponse,
		).ServeHTTP))

		general.PUT("/", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateGeneralSettingEndpoint,
			decodeGeneralSettingRequest,
			encodeResponse,
		).ServeHTTP))

		general.GET("/", gin.WrapF(httptransport.NewServer(
			endpoints.GetAllGeneralSettingEndpoint,
			decodeEmptyRequest,
			encodeResponse,
		).ServeHTTP))
	}

	organization := router.Group("/organization")
	{
		organization.GET("/non-singhealth", gin.WrapF(httptransport.NewServer(
			endpoints.GetAllNonSingHealthOrganizationsEndpoint,
			decodeEmptyRequest,
			encodeResponse,
		).ServeHTTP))
	}

	settings := router.Group("/org-setting")
	{
		settings.POST("/update", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateOrganizationSettingByOrgIDEndpoint,
			decodeUpdateOrganizationSettingRequest,
			encodeResponse,
		).ServeHTTP))

		settings.POST("/get", gin.WrapF(httptransport.NewServer(
			endpoints.GetOrganizationSettingByOrgIDEndpoint,
			decodeGetOrganizationSettingRequest,
			encodeResponse,
		).ServeHTTP))
	}

	return r
}

func decodeUUIDParam(_ context.Context, r *http.Request) (interface{}, error) {
	// This assumes path ends with /:id
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path")
	}
	idStr := parts[len(parts)-1]
	return idStr, nil // ← string is passed to endpoint
}

func decodeUpdateUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.UpdateUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}

	// Convert to entity.User
	user := &entities.User{
		ID:       req.ID,
		FullName: req.FullName,
		UserName: req.UserName,
		Country:  req.Country,
	}

	return user, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

func decodeGeneralSettingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.GeneralSettingRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeEmptyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func decodeUpdateOrganizationSettingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.OrganizationSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil // Return pointer
}

func decodeGetOrganizationSettingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.OrganizationSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil // Return pointer
}
