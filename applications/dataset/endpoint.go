package base

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/go-kit/kit/endpoint"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"
)

type Endpoints struct {
	UploadDataSet      endpoint.Endpoint
	GetDataSetfile     endpoint.Endpoint
	DeleteDataSetfile  endpoint.Endpoint
	PreviewDataSetfile endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		UploadDataSet:      Middleware(makeUploadDataSet(s), commonlib.TimeoutMs),
		GetDataSetfile:     Middleware(makeGetDataSetfile(s), commonlib.TimeoutMs),
		DeleteDataSetfile:  Middleware(makeDeleteFileEndpoint(s), commonlib.TimeoutMs),
		PreviewDataSetfile: Middleware(MakeOpenFileEndpoint(s), commonlib.TimeoutMs),
	}
}

// Middlewares applies both error handling and timeout middleware to an endpoint...
func Middleware(endpoint endpoint.Endpoint, timeout time.Duration) endpoint.Endpoint {
	return service.ErrorHandlingMiddleware(service.TimeoutMiddleware(5 * timeout)(endpoint))
}

func makeUploadDataSet(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(model.UploadRequest)

		//path, err := s.UploadFile(ctx, req.FilePath)
		res, err := s.UploadFile(ctx, req)
		if err != nil {
			return nil, err
		}
		return model.UploadResponse{ID: res.ID, FilePath: res.FilePath}, nil
	}
}
func makeGetDataSetfile(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(model.GetFileRequest) // Assuming you have a struct for this request

		// Call the service to get the file content
		content, fileName, err := s.GetFile(ctx, req.FilePath)
		contentType := http.DetectContentType(content)
		if err != nil {
			return nil, err
		}
		return model.GetFileResponse{FileName: fileName, Content: content, ContentType: contentType}, nil
	}
}
func makeDeleteFileEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(model.DeleteFileRequest)

		err := s.DeleteFile(ctx, req)
		if err != nil {
			return nil, err
		}

		return model.DeleteFileResponse{
			Message: "File deleted successfully",
		}, nil
	}
}

//	func MakeOpenFileEndpoint(s service.Service) endpoint.Endpoint {
//		return func(ctx context.Context, request interface{}) (interface{}, error) {
//			req := request.(model.OpenFileRequest)
//			file, err := s.OpenFile(ctx, req.FileURL)
//			if err != nil {
//				return model.OpenFileResponse{Err: err.Error()}, nil
//			}
//			return model.OpenFileResponse{File: file}, nil
//		}
//	}
func MakeOpenFileEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(model.OpenFileRequest)
		file, err := s.OpenFile(ctx, req.FileURL)
		if err != nil {
			return model.OpenFileResponse{Err: err.Error()}, nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var lines []string
		for i := 0; i < 10 && scanner.Scan(); i++ {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return model.OpenFileResponse{Err: fmt.Sprintf("failed to read file: %v", err)}, nil
		}

		fileInfo, err := file.Stat()
		if err != nil {
			return model.OpenFileResponse{Err: fmt.Sprintf("failed to get file info: %v", err)}, nil
		}
		return model.OpenFileResponse{
			FileName:    fileInfo.Name(),
			FileSize:    fileInfo.Size(),
			FilePath:    file.Name(),
			FilePreview: strings.Join(lines, "\n"),
		}, nil
	}
}
