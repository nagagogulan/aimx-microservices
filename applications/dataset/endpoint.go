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
				return model.OpenFileResponse{Err: fmt.Sprintf("failed to open zip file: %v", err)}, nil
			}
			defer reader.Close()

			nodeMap := make(map[string]*model.FileNode)

			// Recursive folder creation - use pointers for Children
			var ensurePath func(path string) *model.FileNode
			ensurePath = func(path string) *model.FileNode {
				path = filepath.ToSlash(path)
				if node, exists := nodeMap[path]; exists {
					return node
				}

				node := &model.FileNode{
					Name:     filepath.Base(path),
					Type:     "folder",
					Children: []*model.FileNode{}, // <-- slice of pointers!
				}
				nodeMap[path] = node

				parentPath := filepath.ToSlash(filepath.Dir(path))
				if parentPath != "." && parentPath != path {
					parent := ensurePath(parentPath)
					parent.Children = append(parent.Children, node) // append pointer
				}
				return node
			}

			for _, file := range reader.File {
				fullPath := filepath.ToSlash(file.Name)
				if fullPath == "" {
					continue
				}

				isDir := file.FileInfo().IsDir()
				if isDir {
					fullPath = strings.TrimSuffix(fullPath, "/")
				}

				parentPath := filepath.ToSlash(filepath.Dir(fullPath))
				if parentPath != "." {
					ensurePath(parentPath) // ensure all parent folders exist
				}

				node := &model.FileNode{
					Name:     filepath.Base(fullPath),
					Type:     "folder",
					Children: []*model.FileNode{}, // slice of pointers!
				}
				if !isDir {
					node.Type = "file"
				}
				nodeMap[fullPath] = node

				if !isDir {
					rc, err := file.Open()
					if err != nil {
						continue
					}
					content, _ := io.ReadAll(rc)
					rc.Close()

					ext := strings.ToLower(filepath.Ext(file.Name))
					switch ext {
					case ".jpg", ".jpeg":
						node.Type = "image"
						node.Preview = []string{"data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(content)}
					case ".png":
						node.Type = "image"
						node.Preview = []string{"data:image/png;base64," + base64.StdEncoding.EncodeToString(content)}
					case ".gif":
						node.Type = "image"
						node.Preview = []string{"data:image/gif;base64," + base64.StdEncoding.EncodeToString(content)}
					case ".csv":
						lines := strings.Split(string(content), "\n")
						for i := range lines {
							lines[i] = strings.TrimRight(lines[i], "\r\n")
						}
						if len(lines) > 10 {
							lines = lines[:10]
						}
						node.Type = "csv"
						node.Preview = lines
					case ".xlsx":
						tmp, err := os.CreateTemp("", "*.xlsx")
						if err != nil {
							node.Preview = []string{"Error: cannot preview xlsx"}
							break
						}
						defer os.Remove(tmp.Name())
						tmp.Write(content)
						tmp.Close()

						excel, err := excelize.OpenFile(tmp.Name())
						if err == nil {
							defer excel.Close()
							sheet := excel.GetSheetName(0)
							rows, _ := excel.GetRows(sheet)
							for i := 0; i < len(rows) && i < 10; i++ {
								node.Preview = append(node.Preview, strings.Join(rows[i], "\t"))
							}
							node.Type = "xlsx"
						}
					default:
						node.Preview = []string{"Not supported"}
					}
				}

				// Attach file node to parent folder pointer (only files here)
				if parent, ok := nodeMap[parentPath]; ok && fullPath != parentPath {
					parent.Children = append(parent.Children, node) // append pointer
				}
			}

			// Return only root nodes (whose parent is ".")
			var result []*model.FileNode
			for path, node := range nodeMap {
				parentPath := filepath.ToSlash(filepath.Dir(path))
				if parentPath == "." || nodeMap[parentPath] == nil {
					result = append(result, node)
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
