package base

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	errorlib "github.com/PecozQ/aimx-library/apperrors"
	commonlib "github.com/PecozQ/aimx-library/common"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"

	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	//"github.com/gorilla/mux"
)

func MakeHttpHandler(s service.Service) http.Handler {
	options := []httptransport.ServerOption{httptransport.ServerErrorEncoder(errorlib.EncodeError)}

	r := gin.New()
	endpoints := NewEndpoint(s)

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Next()
	})

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

	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	// id := r.FormValue("id")
	// if id == "" {
	// 	return nil, fmt.Errorf("id is required")
	// }

	req := model.UploadRequest{
		//ID:       id,
		FileName: header.Filename,
		Content:  bytes,
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
func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
