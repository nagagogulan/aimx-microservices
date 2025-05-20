package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	middleware "github.com/PecozQ/aimx-library/middleware"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gofrs/uuid"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"
)

// func init() {
// 	// Get the current working directory (from where the command is run)
// 	dir, err := os.Getwd()
// 	if err != nil {
// 		fmt.Errorf("Error getting current working directory:", err)
// 	}
// 	fmt.Println("Current Working Directory:", dir)

// 	// Construct the path to the .env file in the root directory
// 	envPath := filepath.Join(dir, "../.env")

// 	// Load the .env file from the correct path
// 	err = godotenv.Load(envPath)
// 	if err != nil {
// 		fmt.Errorf("Error loading .env file")
// 	}
// }

func MakeHTTPHandler(s service.Service) http.Handler {
	fmt.Println("connect http handuler")
	options := []httptransport.ServerOption{httptransport.ServerErrorEncoder(errcom.EncodeError)}

	r := gin.New()
	endpoints := NewEndpoint(s)

	// r.Use(cors.New(cors.Config{
	// 	AllowOrigins:     []string{"http://54.251.96.179:3000", "http://localhost:3000", "http://13.229.196.7:3000"}, // Replace with your frontend's origin
	// 	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	// 	AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	// 	AllowCredentials: true,
	// }))

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
			options...,
		).ServeHTTP))

		// PUT: /profile
		api.PUT("/", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateUserProfileEndpoint,
			decodeUpdateUserRequest, // Decode body to user
			encodeResponse,
			options...,
		).ServeHTTP))

		api.POST("/image", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateProfileImageEndpoint,
			decodeUploadProfileImageRequest, // This will now extract 'id' from the URL path
			encodeResponse,
			options...,
		).ServeHTTP))
	}

	// General Setting APIs
	general := router.Group("/general-setting")
	{
		general.POST("/", gin.WrapF(httptransport.NewServer(
			endpoints.CreateGeneralSettingEndpoint,
			decodeGeneralSettingRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

		general.PUT("/", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateGeneralSettingEndpoint,
			decodeGeneralSettingRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

		general.GET("/", gin.WrapF(httptransport.NewServer(
			endpoints.GetAllGeneralSettingEndpoint,
			decodeEmptyRequest,
			encodeResponse,
			options...,
		).ServeHTTP))
	}

	organization := router.Group("/organization")
	{
		organization.GET("/non-singhealth", gin.WrapF(httptransport.NewServer(
			endpoints.GetAllNonSingHealthOrganizationsEndpoint,
			decodeEmptyRequest,
			encodeResponse,
			options...,
		).ServeHTTP))
	}

	settings := router.Group("/org-setting")
	{
		settings.POST("/update", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateOrganizationSettingByOrgIDEndpoint,
			decodeUpdateOrganizationSettingRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

		settings.POST("/get", gin.WrapF(httptransport.NewServer(
			endpoints.GetOrganizationSettingByOrgIDEndpoint,
			decodeGetOrganizationSettingRequest,
			encodeResponse,
			options...,
		).ServeHTTP))

		settings.POST("/create", gin.WrapF(httptransport.NewServer(
			endpoints.CreateOrganizationSettingEndpoint,
			decodeCreateOrganizationSettingRequest,
			encodeResponse,
			options...,
		).ServeHTTP))
	}

	overview := router.Group("/overview")
	{
		overview.GET("/", gin.WrapF(httptransport.NewServer(
			endpoints.OverviewEndpoint,
			decodeOverviewRequest,
			encodeResponse,
			options...,
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
	_, errs := middleware.DecodeHeaderGetClaims(r)
	if errs != nil {
		return nil, errs // Unauthorized or invalid token
	}
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
	_, errs := middleware.DecodeHeaderGetClaims(r)
	if errs != nil {
		return nil, service.NewAppError(errs, http.StatusUnauthorized, errs.Error(), nil)
	}
	var req dto.GeneralSettingRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeEmptyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, errs := middleware.DecodeHeaderGetClaims(r)
	if errs != nil {
		return nil, service.NewAppError(errs, http.StatusUnauthorized, errs.Error(), nil)
	}
	return nil, nil
}

func decodeUpdateOrganizationSettingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.OrganizationSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil // Return pointer
}

func decodeCreateOrganizationSettingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, errs := middleware.DecodeHeaderGetClaims(r)
	if errs != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.OrganizationSettingRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeGetOrganizationSettingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.OrganizationSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil // Return pointer
}

func decodeOverviewRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	claims, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}

	userIDStr := claims.UserID // assuming the struct has a field UserID
	orgIDStr := claims.OrganizationID
	if userIDStr == "" || orgIDStr == "" {
		return nil, fmt.Errorf("Claim details not found in token")
	}

	return &dto.OverviewRequest{
		UserID: userIDStr,
		OrgID:  orgIDStr,
	}, nil
}
func decodeUploadProfileImageRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	// Extract claims from JWT in request headers (authorization)
	claims, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // unauthorized or invalid token
	}

	// Parse userID from claims (assuming claims.UserID is string)
	userID, err := uuid.FromString(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in claims: %w", err)
	}

	// Read file from multipart form
	_, fileHeader, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	fileName := fileHeader.Filename
	fmt.Println("Uploaded file name:", fileName)

	// Return typed request struct
	return &model.UploadProfileImageRequest{
		UserID:     userID,
		FileHeader: fileHeader,
	}, nil
}
