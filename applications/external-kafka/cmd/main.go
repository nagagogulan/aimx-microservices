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

	"github.com/PecozQ/aimx-library/database/pgsql"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

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

	dbPortStr := os.Getenv("DBPORT")
	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		fmt.Printf("Invalid DBPORT value: %v\n", err)
		return
	}
	DB, err := pgsql.InitDB(&pgsql.Config{
		DBHost:     os.Getenv("DBHOST"),
		DBPort:     dbPort,
		DBUser:     os.Getenv("DBUSER"),
		DBPassword: os.Getenv("DBPASSWORD"),
		DBName:     os.Getenv("DBNAME"),
	})
	if err != nil {
		fmt.Println("Error initializing DB: %v", err)
	}

	// Ping the database to check if the connection is successful
	sqlDB, err := DB.DB() // Get the raw SQL database instance

	if err != nil {
		fmt.Println("Error getting raw DB instance: %v", err)
	}

	// Attempt to ping the database to check if it's alive
	err = sqlDB.Ping()
	if err != nil {
		fmt.Println("Failed to ping the database: %v", err)
	} else {
		fmt.Println("Database connection successful!")
	}
	err = pgsql.Migrate(DB)
	if err != nil {
		fmt.Println("Could not migrate database: %v", err)
		return
	}

	// Close the DB connection when done (deferred)
	defer sqlDB.Close()

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

	// Initialize repositories
	formRepo := repository.NewFormRepository(db)
	sampleDatasetRepo := repository.NewSampleDatasetRepository(DB)
	userRepo := repository.NewUserCRUDRepository(DB)
	docketMetricsRepo := repository.NewDocketMetricsRepository(db)
	docketStatusRepo := repository.NewDocketStatusRepositoryService(DB)

	// Start the dataset chunk subscriber with form repository (processes chunks and creates forms)
	go worker.StartDatasetChunkSubscriber(formRepo, sampleDatasetRepo, userRepo)

	go worker.StartFileChunkWorker()

	try {
		go worker.StartDocketStatusResultSubscriber(docketMetricsRepo, docketStatusRepo)
	} catch(e) {
		console.log("e", e)
	}

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
		uploadSvc = service.NewUploadService(log.With(logger, "component", "UploadService"), "shared/dockets")
		// If you add middleware for the service, wrap it here:
		// uploadSvc = service.NewLoggingMiddleware(log.With(logger, "component", "LoggingMiddleware"))(uploadSvc)
	}

	// Initialize repositories
	// docketStatusRepo := repository.NewDocketStatusRepositoryService(DB)

	// Initialize services
	// statusSvc := service.NewStatusService(docketStatusRepo, logger)

	// Start the docket status worker in a goroutine
	// go func() {
	// 	worker.StartDocketStatusWorker(statusSvc)
	// }()

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

	level.Error(logger).Log("exit", <-errs)
}
