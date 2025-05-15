package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/middleware"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/joho/godotenv"
)

func init() {
	// Get the current working directory (from where the command is run)
	dir, err := os.Getwd()
	if err != nil {
		fmt.Errorf("Error getting current working directory:", err)
	}
	fmt.Println("Current Working Directory:", dir)

	// Construct the path to the .env file in the root directory
	envPath := filepath.Join(dir, "../.env")

	// Load the .env file from the correct path
	err = godotenv.Load(envPath)
	if err != nil {
		fmt.Errorf("Error loading .env file")
	}
}

func MakeHTTPHandler(endpoints Endpoints) http.Handler {
	r := gin.New()
	r.Use(gin.Logger())

	// r.Use(cors.New(cors.Config{
	// 	AllowOrigins:     []string{"http://54.251.209.147:3000", "http://localhost:3000"}, // Replace with your frontend's origin
	// 	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	// 	AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	// 	AllowCredentials: true,
	// }))

	// Base router group: /api/v1
	router := r.Group(fmt.Sprintf("%s/%s", common.BasePath, common.Version))

	// Role
	api := router.Group("/user-profile")
	{
		api.GET("/", gin.WrapF(httptransport.NewServer(
			endpoints.ListUsersEndpoint,
			decodeListUsersRequest,
			encodeResponse,
		).ServeHTTP))

		api.DELETE("/:id", gin.WrapF(httptransport.NewServer(
			endpoints.DeleteUserEndpoint,
			decodeUUIDParam,
			encodeResponse,
		).ServeHTTP))

		api.PUT("/deactivate/:id", gin.WrapF(httptransport.NewServer(
			endpoints.DeactivateUserEndpoint,
			decodeUUIDParam,
			encodeResponse,
		).ServeHTTP))
	}
	return r
}

func decodeUUIDParam(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, err // Unauthorized or invalid token
	}
	// This assumes path ends with /:id
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path")
	}
	idStr := parts[len(parts)-1]
	return idStr, nil // ← string is passed to endpoint
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

func decodeListUsersRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	claims, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, err // Unauthorized or invalid token
	}

	// Parse pagination and search
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	search := r.URL.Query().Get("search")
	reqType := r.URL.Query().Get("type")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 {
		limit = 10
	}

	// Parse filters from the request
	filters := make(map[string]interface{})
	filterParams := r.URL.Query().Get("filters")
	if filterParams != "" {
		// If filters are passed as query params, parse them
		err := json.Unmarshal([]byte(filterParams), &filters)
		if err != nil {
			return nil, fmt.Errorf("invalid filters: %v", err)
		}
	}
	return map[string]interface{}{
		"organisation_id": claims.OrganizationID,
		"user_id":         claims.UserID,
		"page":            page,
		"limit":           limit,
		"search":          search,
		"filters":         filters, // Include filters in the request
		"type":			   reqType,
	}, nil
}

