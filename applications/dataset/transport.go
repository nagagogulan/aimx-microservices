package base

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	errorlib "github.com/PecozQ/aimx-library/apperrors"
	commonlib "github.com/PecozQ/aimx-library/common"
	middleware "github.com/PecozQ/aimx-library/middleware"
	"github.com/joho/godotenv"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	//"github.com/gorilla/mux"
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
func MakeHttpHandler(s service.Service) http.Handler {
	options := []httptransport.ServerOption{httptransport.ServerBefore(func(ctx context.Context, r *http.Request) context.Context {
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
		AllowOrigins:     []string{"http://54.251.209.147:3000", "http://localhost:3000", "http://13.229.196.7:3000"}, // Replace with your frontend's origin
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	router := r.Group(fmt.Sprintf("%s/%s", commonlib.BasePath, commonlib.Version))

	//Register and Login Endpoints...
	router.POST("/fileupload", gin.WrapF(httptransport.NewServer(
		endpoints.UploadDataSet,
		decodeUploadRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	router.GET("/file/get", gin.WrapF(httptransport.NewServer(
		endpoints.GetDataSetfile,
		decodeGetFileRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	router.DELETE("/file/delete", gin.WrapF(httptransport.NewServer(
		endpoints.DeleteDataSetfile,
		decodeDeleteFileRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	return r
}
func decodeUploadRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	// Create a new context with organization ID
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	fileType := http.DetectContentType(bytes)
	if fileType == "" {
		return nil, fmt.Errorf("unable to detect file type")
	}
	extension := filepath.Ext(header.Filename) // returns ".jpg", ".csv", etc.
	if extension != "" && extension[0] == '.' {
		extension = extension[1:] // strip the leading dot
	}

	req := model.UploadRequest{
		FileType:  fileType,
		FileName:  header.Filename,
		Content:   bytes,
		Extension: extension, // <-- add this field to your UploadRequest struct
	}
	FormType := r.FormValue("FormType")
	if FormType != "" {
		req.FormType = FormType
	}

	// Return the request object with file_path
	return req, nil
}
func decodeGetFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req model.GetFileRequest

	// Decode the request body into the GetFileRequest struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}

	// Optionally, check for any validation or preprocessing
	if req.FilePath == "" {
		return nil, fmt.Errorf("filePath is required")
	}

	return req, nil
}

func decodeDeleteFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req model.DeleteFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
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
	if err != nil {
		return nil, err
	}
	// Validate JWT and extract orgID
	claims, err := middleware.ValidateJWT(token, accessSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid tokennbnbnbnbn: %v", err)
	}
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
func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
