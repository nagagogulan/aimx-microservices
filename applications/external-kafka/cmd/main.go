package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

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

	// // Get the current working directory (from where the command is run)
	// dir, err := os.Getwd()
	// if err != nil {
	// 	fmt.Errorf("Error getting current working directory:", err)
	// }
	// fmt.Println("Current Working Directory:", dir)

	// // Construct the path to the .env file in the root directory
	// envPath := filepath.Join(dir, "./.env")

	// // Load the .env file from the correct path
	// err = godotenv.Load(envPath)
	// if err != nil {
	// 	fmt.Errorf("Error loading .env file")
	// }

	uri := os.Getenv("MONGO_URI") // replace with your MongoDB URI

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		fmt.Printf("Error connecting to MongoDB: %v", err)
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		fmt.Printf("Could not ping to MongoDB: %v", err)
	}

	fmt.Println("Successfully connected to MongoDB!")

	// Get a handle to a collection
	db := client.Database(os.Getenv("MONGO_DBNAME"))

	// Initialize form repository
	formRepo := repository.NewFormRepository(db)

	// Start the dataset chunk subscriber with form repository (processes chunks and creates forms)
	go worker.StartDatasetChunkSubscriber(formRepo)

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
	err = server.ListenAndServe()
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
