package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	// Gin is now used in the subscriber package (transport.go)
	// "github.com/gin-gonic/gin"

	subscriber "whatsdare.com/fullstack/aimx/backend" // Alias for the subscriber package
	"whatsdare.com/fullstack/aimx/backend/service"
	"whatsdare.com/fullstack/aimx/backend/worker"
)

const (
	defaultHTTPPort  = "8090"      // Choose a port for this service
	defaultUploadDir = "./uploads" // Default directory for uploaded files
)

func main() {
	flag.Parse()

	go worker.StartFileChunkWorker()

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.NewSyncLogger(logger)
		logger = level.NewFilter(logger, level.AllowDebug()) // Adjust log level as needed
		logger = log.With(logger,
			"svc", "subscriber",
			"ts", log.DefaultTimestampUTC,
			"caller", log.DefaultCaller,
		)
	}

	level.Info(logger).Log("msg", "subscriber service with Gin starting")
	defer level.Info(logger).Log("msg", "subscriber service with Gin ended")

	// Create the upload service
	var uploadSvc service.UploadService
	{
		uploadSvc = service.NewUploadService(log.With(logger, "component", "UploadService"), "dockets")
		// If you add middleware for the service, wrap it here:
		// uploadSvc = service.NewLoggingMiddleware(log.With(logger, "component", "LoggingMiddleware"))(uploadSvc)
	}

	// Create Gin HTTP server (router)
	// The NewGinServer function is defined in applications/subscriber/transport.go
	ginRouter := subscriber.NewGinServer(uploadSvc, logger)

	errs := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(8088),
		Handler:      ginRouter, // Use the Gin router as the handler
		ReadTimeout:  1 * time.Hour,
		WriteTimeout: 1 * time.Hour,
		IdleTimeout:  1 * time.Hour,
	}

	fmt.Println("Info", "Role service HTTP server started", "port", 8088)
	err := server.ListenAndServe()
	if err != nil {
		// log.Fatalf("HTTP server failed: %v", err)
	}

	// go func() {
	// 	// level.Info(logger).Log("transport", "HTTP (Gin)", "addr", *httpAddr, "upload_dir", *uploadDir)
	// 	server := &http.Server{
	// 		Addr:   ":" + strconv.Itoa(8089),
	// 		Handler: ginRouter, // Use the Gin router as the handler
	// 		ReadTimeout:  1 * time.Hour,
	// 		WriteTimeout: 1 * time.Hour,
	// 		IdleTimeout:  1 * time.Hour,
	// 	}
	// 	errs <- server.ListenAndServe()
	// }()

	level.Error(logger).Log("exit", <-errs)
}
