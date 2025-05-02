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

	errorlib "github.com/PecozQ/aimx-library/apperrors"
	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/middleware"
	"whatsdare.com/fullstack/aimx/backend/service"
	"github.com/gofrs/uuid"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/joho/godotenv"
	"whatsdare.com/fullstack/aimx/backend/model"
)

func init() {
	// Get the current working directory (from where the command is run)
	dir, err := os.Getwd()
	if err != nil {
		fmt.Errorf("Error getting current working directory:", err)
	}
	fmt.Println("Current Working Directory:", dir)

	// Construct the path to the .env file in the root directory
	envPath := filepath.Join(dir, "./.env")

	// Load the .env file from the correct path
	err = godotenv.Load(envPath)
	if err != nil {
		fmt.Errorf("Error loading .env file")
	}
}

func MakeHttpHandler(s service.Service) http.Handler {
	options := []httptransport.ServerOption{
		httptransport.ServerBefore(func(ctx context.Context, r *http.Request) context.Context {
			return context.WithValue(ctx, "HTTPRequest", r)
		}),
		httptransport.ServerErrorEncoder(errorlib.EncodeError)}

	r := gin.New()
	endpoints := NewEndpoint(s)

	// r.Use(func(c *gin.Context) {
	// 	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	// 	c.Next()
	// })

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://54.251.209.147:3000", "http://localhost:3000"}, // Replace with your frontend's origin
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	router := r.Group(fmt.Sprintf("%s/%s", commonlib.BasePath, commonlib.Version))

	//Register and Login Endpoints...
	router.POST("/template/create", gin.WrapF(httptransport.NewServer(
		endpoints.CreateTemplateEndpoint,
		decodeCreateTemplateRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	// Get Template by ID
	router.GET("/template", gin.WrapF(httptransport.NewServer(
		endpoints.GetTemplateByIDEndpoint,
		decodeGetTemplateByTypeRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// Update Template
	router.PUT("/template/update", gin.WrapF(httptransport.NewServer(
		endpoints.UpdateTemplateEndpoint,
		decodeUpdateTemplateRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// Delete Template
	router.DELETE("/template", gin.WrapF(httptransport.NewServer(
		endpoints.DeleteTemplateEndpoint,
		decodeDeleteTemplateRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	router.POST("/form/create", gin.WrapF(httptransport.NewServer(
		endpoints.CreateFormEndpoint,
		decodeCreateFormRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// Get Template by ID
	router.GET("/form", gin.WrapF(httptransport.NewServer(
		endpoints.GetFormByTypeEndpoint, // ✅ changed from GetTemplateByTypeEndpoint
		decodeGetFormByTypeRequest,      // ✅ updated to match GetTemplateByID
		encodeResponse,
		options...,
	).ServeHTTP))
	router.POST("/formtype/create", gin.WrapF(httptransport.NewServer(
		endpoints.CreateFormTypeEndpoint,
		decodeCreateFormTypeRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// Get Template by ID
	router.GET("/formtype/list", gin.WrapF(httptransport.NewServer(
		endpoints.GetFormTypeEndpoint, // ✅ changed from GetTemplateByTypeEndpoint
		decodeGetFormTypeRequest,      // ✅ updated to match GetTemplateByID
		encodeResponse,
		options...,
	).ServeHTTP))

	// Update Template
	router.PUT("/form/update", gin.WrapF(httptransport.NewServer(
		endpoints.UpdateFormEndpoint,
		decodeUpdateFormRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	router.GET("/form/search", gin.WrapF(httptransport.NewServer(
		endpoints.FilterFormsEndpoint,
		decodeSearchFormsRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	router.GET("/form/searchbyorg", gin.WrapF(httptransport.NewServer(
		endpoints.SearchFormsEndpoint,
		decodeSearchFormsByOrgNameRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	router.POST("/docket/shortlist", gin.WrapF(httptransport.NewServer(
		endpoints.ShortlistDocketEndpoint,
		decodeShortlistDocketRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	router.POST("/docket/rating", gin.WrapF(httptransport.NewServer(
		endpoints.RatingDocketEndpoint,
		decodeRatingEndpointRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	router.GET("/docket/comments", gin.WrapF(httptransport.NewServer(
		endpoints.GetCommentsByIdEndpoint,
		decodeShortlistDocketRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	router.GET("/filterfield/get", gin.WrapF(httptransport.NewServer(
		endpoints.GetFormFilterBYTypeEndpoint,
		decodeGetFilterFieldsByTypeRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	router.PUT("organization/deactivate/:organization_id", gin.WrapF(httptransport.NewServer(
		endpoints.DeactivateOrganizationEndpoint,
		decodeDeactivateOrganizationRequest,  // This uses gin.Context, not http.Request
		encodeResponse,
		options...,
	).ServeHTTP))

	return r
}

func decodeShortlistDocketRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dto.ShortListDTO
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("the error is givne as:", err)
		return nil, err
	}
	return request, nil
}

func decodeRatingEndpointRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dto.RatingDTO
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeHeaderGetClaims(r *http.Request) (*middleware.Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return nil, fmt.Errorf("invalid Authorization header format")
	}

	accessSecret, err := generateJWTSecrets()
	// Validate JWT and extract orgID
	claims, err := middleware.ValidateJWT(token, accessSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid tokennbnbnbnbn: %v", err)
	}
	fmt.Println("the claim is given as:", claims)
	return claims, nil
}

// Fetch JWT secrets from environment variables
func generateJWTSecrets() (string, error) {
	accessSecret := os.Getenv("ACCESS_SECRET")
	if accessSecret == "" {
		return "", fmt.Errorf("JWT secret keys are not set in environment variables")
	}
	return accessSecret, nil
}

// Decode register api request...
func decodeCreateTemplateRequest(ctx context.Context, r *http.Request) (interface{}, error) {

	var request entities.Template
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

// Decode register api request...
func decodeCreateFormRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dto.TemplateDTO
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	// Extract Gin context
	// newCtx, err := commonlib.ExtractGinContext(ctx)
	// if err != nil {
	// 	return nil, err
	// }
	return &dto.FormDTO{Type: request.Type, Sections: request.Sections, Fields: request.Fields}, nil
}
func decodeCreateFormTypeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dto.FormType
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	// Extract Gin context
	// newCtx, err := commonlib.ExtractGinContext(ctx)
	// if err != nil {
	// 	return nil, err
	// }
	return &request, nil
}

func decodeGetTemplateByTypeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id := strings.TrimSpace(r.URL.Query().Get("id")) // remove quotes if passed in URL
	typeStr := r.URL.Query().Get("type")

	req := &model.ParamRequest{}

	if id != "" {
		req.ID = id
	}
	if typeStr != "" {
		typeInt, err := strconv.Atoi(typeStr)
		if err != nil {
			fmt.Println("Error:", err)
			return nil, err
		}
		if typeInt > 0 {
			req.Type = typeInt
		}
	}

	if req.ID == "" && req.Type < 0 {
		return nil, fmt.Errorf("either 'id' or 'type' must be provided")
	}
	return req, nil
}
func decodeSearchFormsByOrgNameRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	formName := strings.TrimSpace(r.URL.Query().Get("formname"))
	typeStr := r.URL.Query().Get("type")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")

	// Create the request object to store the decoded data
	req := model.SearchFormsByOrganizationRequest{
		FormName: formName,
	}
	// Validate that the "organization_name" is provided
	if req.FormName == "" {
		return nil, fmt.Errorf("organization_name must be provided")
	}
	typeInt, err := strconv.Atoi(typeStr)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	if typeInt > 0 {
		req.Type = typeInt
	}
	pages, err := strconv.Atoi(pageStr)
	if err == nil && pages > 0 {
		req.Page = pages
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err == nil && pageSize > 0 {
		req.PageSize = pageSize
	}

	return req, nil
}

func decodeGetFormByTypeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	typeStr := r.URL.Query().Get("type")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")
	status := r.URL.Query().Get("status")

	req := &model.ParamRequest{}
	typeInt, err := strconv.Atoi(typeStr)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}

	if typeInt > 0 {
		req.Type = typeInt
	}
	pages, err := strconv.Atoi(pageStr)
	if err == nil && pages > 0 {
		req.Page = pages
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err == nil && pageSize > 0 {
		req.PageSize = pageSize
	}

	intStatus, err := strconv.Atoi(status)
	if err == nil {
		req.Status = intStatus
	}
	// page, pageSize := commonlib.ParsePaginationParams(r)
	// req.Page = page
	// req.PageSize = pageSize
	return req, nil
}

func decodeGetFormTypeRequest(ctx context.Context, r *http.Request) (interface{}, error) {

	return nil, nil
}

func decodeUpdateTemplateRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request entities.Template

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeUpdateFormRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dto.UpdateFormRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return &request, nil
}

func decodeDeleteTemplateRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id := strings.TrimSpace(r.URL.Query().Get("id")) // remove quotes if passed in URL
	req := &model.ParamRequest{}

	if id != "" {
		req.ID = id
	}
	return req, nil
}
func decodeSearchFormsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req model.SearchFormsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeGetFilterFieldsByTypeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	// remove quotes if passed in URL
	typeStr := r.URL.Query().Get("type")
	req := &model.ParamRequest{}

	if typeStr != "" {
		typeInt, err := strconv.Atoi(typeStr)
		if err != nil {
			fmt.Println("Error:", err)
			return nil, err
		}
		if typeInt > 0 {
			req.Type = typeInt
		}
	}

	if req.Type < 0 {
		return nil, fmt.Errorf("either 'id' or 'type' must be provided")
	}
	return req, nil
}

func decodeDeactivateOrganizationRequest(ctx context.Context, r *http.Request) (interface{}, error) {
    // Extract the organization_id from the URL path using http.Request
    orgIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/organization/deactivate/") // Extract organization_id from path
    
    if orgIDStr == "" {
        return nil, fmt.Errorf("organization_id is required")
    }

    // Convert the string to UUID
    orgID, err := uuid.FromString(orgIDStr)
    if err != nil {
        return nil, fmt.Errorf("invalid organization ID: %v", err)
    }

    return dto.DeactivateOrganizationRequest{
        OrganizationID: orgID,
    }, nil
}




func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
