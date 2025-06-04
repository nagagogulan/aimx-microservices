package base

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/go-kit/kit/endpoint"
	"github.com/xuri/excelize/v2"
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
		req, ok := request.(model.OpenFileRequest)
		if !ok {
			return model.OpenFileResponse{Err: "invalid request type"}, nil
		}

		file, err := s.OpenFile(ctx, req.FileURL)
		if err != nil {
			return model.OpenFileResponse{Err: err.Error()}, nil
		}
		defer file.Close()

		filePath := file.Name()
		ext := strings.ToLower(filepath.Ext(filePath))
		fileInfo, err := file.Stat()
		if err != nil {
			return model.OpenFileResponse{Err: fmt.Sprintf("failed to get file info: %v", err)}, nil
		}

		switch ext {
		case ".csv":
			file.Seek(0, io.SeekStart)
			scanner := bufio.NewScanner(file)
			var preview []string
			for i := 0; i < 10 && scanner.Scan(); i++ {
				preview = append(preview, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				return model.OpenFileResponse{Err: fmt.Sprintf("failed to read csv file: %v", err)}, nil
			}
			return model.OpenFileResponse{
				FileName:    fileInfo.Name(),
				FileSize:    fileInfo.Size(),
				FilePath:    filePath,
				FileType:    "csv",
				FilePreview: strings.Join(preview, "\n"),
			}, nil
		case ".docx":
			file.Seek(0, io.SeekStart)
			scanner := bufio.NewScanner(file)
			var preview []string
			for i := 0; i < 10 && scanner.Scan(); i++ {
				preview = append(preview, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				return model.OpenFileResponse{Err: fmt.Sprintf("failed to read csv file: %v", err)}, nil
			}
			return model.OpenFileResponse{
				FileName:    fileInfo.Name(),
				FileSize:    fileInfo.Size(),
				FilePath:    filePath,
				FileType:    "docx",
				FilePreview: strings.Join(preview, "\n"),
			}, nil

		case ".xlsx":
			excelFile, err := excelize.OpenFile(filePath)
			if err != nil {
				return model.OpenFileResponse{Err: fmt.Sprintf("failed to read xlsx file: %v", err)}, nil
			}
			defer excelFile.Close()

			sheetName := excelFile.GetSheetName(0)
			rows, err := excelFile.GetRows(sheetName)
			if err != nil {
				return model.OpenFileResponse{Err: fmt.Sprintf("failed to read sheet rows: %v", err)}, nil
			}

			var preview []string
			for i := 0; i < len(rows) && i < 10; i++ {
				preview = append(preview, strings.Join(rows[i], "\t"))
			}

			return model.OpenFileResponse{
				FileName:    fileInfo.Name(),
				FileSize:    fileInfo.Size(),
				FilePath:    filePath,
				FileType:    "xlsx",
				FilePreview: strings.Join(preview, "\n"),
			}, nil
		case ".zip":
			fmt.Println("Info: file format is zip")
			reader, err := zip.OpenReader(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to open zip file: %w", err)
			}
			defer reader.Close()

			nodeMap := make(map[string]*model.FileNode)

			for _, file := range reader.File {
				path := strings.ReplaceAll(file.Name, "\\", "/")
				if path == "" {
					continue
				}

				if file.FileInfo().IsDir() {
					dirPath := strings.TrimSuffix(path, "/")
					if _, exists := nodeMap[dirPath]; !exists {
						nodeMap[dirPath] = &model.FileNode{
							Name:     filepath.Base(dirPath),
							Type:     "folder",
							Children: []model.FileNode{},
						}
					}
					continue
				}

				// Ensure folder hierarchy exists
				dir := filepath.Dir(path)
				if dir != "." {
					parts := strings.Split(dir, "/")
					currentPath := ""
					for _, part := range parts {
						if currentPath != "" {
							currentPath += "/"
						}
						currentPath += part
						if _, exists := nodeMap[currentPath]; !exists {
							nodeMap[currentPath] = &model.FileNode{
								Name:     part,
								Type:     "folder",
								Children: []model.FileNode{},
							}
						}
					}
				}

				fileName := filepath.Base(path)
				ext := strings.ToLower(filepath.Ext(fileName))

				fileNode := model.FileNode{
					Name: fileName,
					Type: "file",
				}

				rc, err := file.Open()
				if err == nil {
					content, _ := io.ReadAll(rc)
					rc.Close()

					switch ext {
					case ".jpg", ".jpeg":
						fileNode.Type = "image"
						fileNode.Preview = []string{"data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(content)}
					case ".png":
						fileNode.Type = "image"
						fileNode.Preview = []string{"data:image/png;base64," + base64.StdEncoding.EncodeToString(content)}
					case ".gif":
						fileNode.Type = "image"
						fileNode.Preview = []string{"data:image/gif;base64," + base64.StdEncoding.EncodeToString(content)}
					case ".csv":
						lines := strings.Split(string(content), "\n")
						if len(lines) > 10 {
							lines = lines[:10]
						}
						fileNode.Type = "csv"
						fileNode.Preview = lines
					case ".xlsx":
						tempFile, err := os.CreateTemp("", "*.xlsx")
						if err != nil {
							return model.OpenFileResponse{Err: "failed to create temp file for xlsx"}, nil
						}
						defer os.Remove(tempFile.Name())

						if _, err := tempFile.Write(content); err != nil {
							return model.OpenFileResponse{Err: "failed to write temp xlsx content"}, nil
						}
						tempFile.Close()

						excelFile, err := excelize.OpenFile(tempFile.Name())
						if err != nil {
							return model.OpenFileResponse{Err: fmt.Sprintf("failed to read xlsx file: %v", err)}, nil
						}
						defer excelFile.Close()

						sheetName := excelFile.GetSheetName(0)
						rows, err := excelFile.GetRows(sheetName)
						if err != nil {
							return model.OpenFileResponse{Err: fmt.Sprintf("failed to read sheet rows: %v", err)}, nil
						}

						var preview []string
						for i := 0; i < len(rows) && i < 10; i++ {
							preview = append(preview, strings.Join(rows[i], "\t"))
						}
						fileNode.Type = "xlsx"
						fileNode.Preview = preview
					default:
						fileNode.Preview = []string{"Not supported"}
					}
				}

				fullFilePath := path
				nodeMap[fullFilePath] = &fileNode

				// Add file to parent folder
				if dir != "." {
					parent := nodeMap[dir]
					parent.Children = append(parent.Children, fileNode)
				}
			}

			// Now build the tree
			result := []model.FileNode{}
			for path, node := range nodeMap {
				if strings.Count(path, "/") == 0 {
					result = append(result, *node)
				} else {
					parentPath := filepath.Dir(path)
					if parent, exists := nodeMap[parentPath]; exists {
						parent.Children = append(parent.Children, *node)
					}
				}
			}

			return model.OpenFileResponse{
				FileName:  fileInfo.Name(),
				FileSize:  fileInfo.Size(),
				FilePath:  filePath,
				FileType:  "zip",
				Structure: result,
			}, nil

		default:
			return model.OpenFileResponse{Err: fmt.Sprintf("unsupported file type: %s", ext)}, nil
		}

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
