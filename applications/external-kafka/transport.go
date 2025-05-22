package subscriber

import (
	"errors"
	"fmt"
	"net/http"

	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"whatsdare.com/fullstack/aimx/backend/service"
)

const (
	maxUploadSizeGin = 20 * 1024 * 1024 * 1024 // 20 GB, adjust as needed
	fileFormFieldGin = "uploadFile"            // The form field name for the file
)

var (
	// ErrNoFileGin is returned when the expected file is not found in the form.
	ErrNoFileGin = errors.New("no file provided in the request or incorrect form field name")
	// ErrFileTooLargeGin is returned when the uploaded file exceeds the maximum allowed size.
	ErrFileTooLargeGin = errors.New("uploaded file is too large")
)

// NewGinServer creates a new Gin HTTP server.
// The UploadService is passed here to be used by the handlers.
func NewGinServer(uploadSvc service.UploadService, logger log.Logger) *gin.Engine {
	// gin.SetMode(gin.ReleaseMode) // Uncomment for production
	router := gin.Default() // Default includes logger and recovery middleware

	// Middleware to set logger for handlers
	router.Use(func(c *gin.Context) {
		c.Set("logger", logger)
		c.Next()
	})

	// Set a higher limit for multipart forms (this is for the sum of all parts, not just one file)
	// Gin's default is 32MB. For very large files, the file itself is streamed.
	// This limit is more for other form fields or smaller files if not streamed.
	// For streaming a single large file, this might not be the primary bottleneck,
	// but good to be aware of. The actual file size check is done separately.
	router.MaxMultipartMemory = 64 << 20 // 64 MB

	// Define the upload route
	// Group routes if you plan to have more under a common path, e.g., /api/v1
	uploadGroup := router.Group(fmt.Sprintf("%s/%s/%s", commonlib.BasePath, commonlib.Version, "extkafka")) // Or just router.POST("/upload", ...)
	{
		uploadGroup.POST("/upload", makeUploadFileGinHandler(uploadSvc, logger))

		// Test endpoint for Kong
		uploadGroup.GET("/test", makeTestKongGinHandler(uploadSvc, logger))
	}

	// Add other routes and handlers here
	// router.GET("/health", func(c *gin.Context) {
	// 	c.JSON(http.StatusOK, gin.H{"status": "ok"})
	// })

	return router
}

// makeUploadFileGinHandler creates a Gin handler function for file uploads.
func makeUploadFileGinHandler(svc service.UploadService, logger log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := c.MustGet("logger").(log.Logger) // Retrieve logger from context

		// Source
		fileHeader, err := c.FormFile(fileFormFieldGin)
		if err != nil {
			level.Error(log).Log("method", "FormFile", "err", err, "field", fileFormFieldGin)
			if errors.Is(err, http.ErrMissingFile) {
				c.JSON(http.StatusBadRequest, gin.H{"error": ErrNoFileGin.Error(), "code": http.StatusBadRequest})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from form: " + err.Error(), "code": http.StatusBadRequest})
			return
		}

		// Check file size
		if fileHeader.Size > maxUploadSizeGin {
			level.Error(log).Log("method", "checkFileSize", "err", "file too large", "size", fileHeader.Size, "max_size", maxUploadSizeGin)
			c.JSON(http.StatusBadRequest, gin.H{"error": ErrFileTooLargeGin.Error(), "code": http.StatusBadRequest})
			return
		}

		// Call the service method
		// The service's UploadFile method expects a context and *multipart.FileHeader
		// Gin's context `c` can be used, or `c.Request.Context()`
		resp, err := svc.UploadFile(c.Request.Context(), fileHeader)
		if err != nil {
			level.Error(log).Log("method", "UploadFile", "err", err)
			// Determine appropriate status code based on error type if possible
			// For now, using InternalServerError for service-level errors
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file: " + err.Error(), "code": http.StatusInternalServerError})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

// makeTestKongGinHandler creates a Gin handler function for the test endpoint
func makeTestKongGinHandler(svc service.UploadService, logger log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := c.MustGet("logger").(log.Logger) // Retrieve logger from context

		// Call the service method
		resp, err := svc.TestKong(c.Request.Context())
		if err != nil {
			level.Error(log).Log("method", "TestKong", "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute test endpoint: " + err.Error(), "code": http.StatusInternalServerError})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

// Note: The old Go-kit specific EncodeResponse, encodeError, DecodeUploadFileRequest are no longer needed
// as Gin handles request parsing and response writing differently.
