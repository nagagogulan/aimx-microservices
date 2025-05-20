package base

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	errorlib "github.com/PecozQ/aimx-library/apperrors"
	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	middleware "github.com/PecozQ/aimx-library/middleware"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	//"github.com/gorilla/mux"
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

//		// Load the .env file from the correct path
//		err = godotenv.Load(envPath)
//		if err != nil {
//			fmt.Errorf("Error loading .env file")
//		}
//	}
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
		AllowOrigins:     []string{"http://54.251.96.179:3000", "http://localhost:3000", "http://13.229.196.7:3000"}, // Replace with your frontend's origin
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
	router.GET("/file/preview", gin.WrapF(httptransport.NewServer(
		endpoints.PreviewDataSetfile,
		decodePreviewFileRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// New endpoint for dataset upload with multipart streaming
	router.POST("/uploadsampledataset", gin.WrapF(httptransport.NewServer(
		endpoints.UploadSampleDataset,
		decodeUploadSampleDatasetRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// // New endpoint for chunking files to Kafka
	// router.POST("/chunkfile", gin.WrapF(httptransport.NewServer(
	// 	endpoints.ChunkFileToKafka,
	// 	decodeChunkFileRequest,
	// 	encodeResponse,
	// 	options...,
	// ).ServeHTTP))

	// New endpoint for chunking files with form data to Kafka
	router.POST("/chunkfile", gin.WrapF(httptransport.NewServer(
		endpoints.ChunkFileToKafka,
		decodeChunkFileRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	return r
}

// decodeUploadSampleDatasetRequest handles the multipart form data for dataset uploads
// It passes the file header directly to the service for streaming
func decodeUploadSampleDatasetRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	// Verify authentication
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, err // Unauthorized or invalid token
	}

	// Parse multipart form with a reasonable max memory
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	// Get the file header from the form
	// r.FormFile returns 3 values: file, header, error
	file, fileHeader, err := r.FormFile("fileName")
	if err != nil {
		return nil, fmt.Errorf("failed to get file from form: %w", err)
	}
	// Close the file when we're done
	defer file.Close()

	// Return the file header directly to be processed by the service
	// This allows streaming the file without loading it entirely into memory
	return fileHeader, nil
}

func decodeUploadRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	Org_Upload := r.FormValue("org_upload")
	if Org_Upload != "" && Org_Upload == "organization" {
		_, err := middleware.DecodeHeaderGetClaims(r)
		if err != nil {
			return nil, errorlib.ErrInvalidOrMissingJWT // Unauthorized or invalid token
		}
	}
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
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errorlib.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
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
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errorlib.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req model.DeleteFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
func decodePreviewFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errorlib.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var request model.OpenFileRequest
	filepath := strings.TrimSpace(r.URL.Query().Get("filepath"))
	request.FileURL = filepath
	fmt.Println("check path testttt", filepath)
	return request, nil
}

// // decodeChunkFileRequest decodes the request for the chunk file API
// func decodeChunkFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
// 	// Verify authentication
// 	_, err := middleware.DecodeHeaderGetClaims(r)
// 	if err != nil {
// 		return nil, errorlib.ErrInvalidOrMissingJWT // Unauthorized or invalid token
// 	}

// 	// Parse the request body
// 	var req dto.ChunkFileRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		return nil, fmt.Errorf("failed to decode request body: %w", err)
// 	}

// 	// Validate required fields
// 	if req.Name == "" {
// 		return nil, fmt.Errorf("name is required")
// 	}
// 	if req.UUID == "" {
// 		return nil, fmt.Errorf("uuid is required")
// 	}
// 	if req.FilePath == "" {
// 		return nil, fmt.Errorf("filepath is required")
// 	}

// 	return req, nil
// }

// decodeExtendedChunkFileRequest decodes the request for the extended chunk file API
func decodeChunkFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	// Verify authentication
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errorlib.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}

	// Parse the request body
	var req dto.ChunkFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("failed to decode request body: %w", err)
	}

	// Validate required fields
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.UUID == "" {
		return nil, fmt.Errorf("uuid is required")
	}
	if req.FilePath == "" {
		return nil, fmt.Errorf("filepath is required")
	}
	if req.FormData.Type == 0 {
		return nil, fmt.Errorf("formData is required")
	}

	return req, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
