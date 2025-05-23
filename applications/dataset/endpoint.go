package base

import (
	"bufio"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/go-kit/kit/endpoint"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"
)

type Endpoints struct {
	UploadDataSet        endpoint.Endpoint
	UploadSampleDataset  endpoint.Endpoint // New endpoint for multipart upload
	GetDataSetfile       endpoint.Endpoint
	DeleteDataSetfile    endpoint.Endpoint
	PreviewDataSetfile   endpoint.Endpoint
	ChunkFileToKafka     endpoint.Endpoint // New endpoint for chunking files with form data
	GetAllSampleDatasets endpoint.Endpoint // New endpoint to get all sample datasets
	TestKongEndpoint     endpoint.Endpoint // Test endpoint for Kong
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		UploadDataSet:        Middleware(makeUploadDataSet(s), commonlib.TimeoutMs),
		UploadSampleDataset:  Middleware(makeUploadSampleDatasetEndpoint(s), commonlib.TimeoutMs),
		GetDataSetfile:       Middleware(makeGetDataSetfile(s), commonlib.TimeoutMs),
		DeleteDataSetfile:    Middleware(makeDeleteFileEndpoint(s), commonlib.TimeoutMs),
		PreviewDataSetfile:   Middleware(MakeOpenFileEndpoint(s), commonlib.TimeoutMs),
		ChunkFileToKafka:     Middleware(makeChunkFileToKafkaEndpoint(s), commonlib.TimeoutMs),
		GetAllSampleDatasets: Middleware(makeGetAllSampleDatasetsEndpoint(s), commonlib.TimeoutMs),
		TestKongEndpoint:     Middleware(makeTestKongEndpoint(s), commonlib.TimeoutMs),
	}
}

// Middlewares applies both error handling and timeout middleware to an endpoint...
func Middleware(endpoint endpoint.Endpoint, timeout time.Duration) endpoint.Endpoint {
	return service.ErrorHandlingMiddleware(service.TimeoutMiddleware(5 * timeout)(endpoint))
}

// makeUploadSampleDatasetEndpoint creates an endpoint for the UploadDataset method
func makeUploadSampleDatasetEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// The request is expected to be *multipart.FileHeader directly from the transport layer
		fileHeader, ok := request.(*multipart.FileHeader)
		if !ok {
			return nil, fmt.Errorf("invalid request type: expected *multipart.FileHeader")
		}

		return s.UploadDataset(ctx, fileHeader)
	}
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

// // makeChunkFileToKafkaEndpoint creates an endpoint for the ChunkFileToKafka method
// func makeChunkFileToKafkaEndpoint(s service.Service) endpoint.Endpoint {
// 	return func(ctx context.Context, request interface{}) (interface{}, error) {
// 		req, ok := request.(dto.ChunkFileRequest)
// 		if !ok {
// 			return nil, fmt.Errorf("invalid request type: expected dto.ChunkFileRequest")
// 		}

// 		return s.ChunkFileToKafka(ctx, req)
// 	}
// }

// makeExtendedChunkFileToKafkaEndpoint creates an endpoint for the ExtendedChunkFileToKafka method
func makeChunkFileToKafkaEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(dto.ChunkFileRequest)
		if !ok {
			return nil, fmt.Errorf("invalid request type: expected dto.ChunkFileRequest")
		}

		return s.ChunkFileToKafka(ctx, req)
	}
}

// makeGetAllSampleDatasetsEndpoint creates an endpoint for the GetAllSampleDatasets method
func makeGetAllSampleDatasetsEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// No request processing needed for this endpoint
		datasets, err := s.GetAllSampleDatasets(ctx)
		if err != nil {
			return nil, err
		}
		return datasets, nil
	}
}

// makeTestKongEndpoint creates an endpoint for the TestKong method
func makeTestKongEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// No request processing needed for this endpoint
		return s.TestKong(ctx)
	}
}
