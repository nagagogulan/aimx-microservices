package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/database/pgsql"
	"github.com/PecozQ/aimx-library/domain/repository"
	base "whatsdare.com/fullstack/aimx/backend"
	"whatsdare.com/fullstack/aimx/backend/service"
	"whatsdare.com/fullstack/aimx/backend/worker"
)

func main() {

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

	// Initialize the SampleDatasetRepositoryService
	sampleDatasetRepo := repository.NewSampleDatasetRepository(DB)

	// Create service with dependencies
	s := service.NewService(sampleDatasetRepo)

	// Start the dataset path worker (processes file paths and sends chunks)
	go worker.StartDatasetPathWorker()

	// Create HTTP handlers
	httpHandlers := base.MakeHttpHandler(s)

	// Set up HTTP server
	httpServer := http.Server{

		Addr:    ":" + strconv.Itoa(8084),
		Handler: http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`),
	}

	fmt.Println("Info", "HTTP server started", "port", 8084)

	// Start HTTP server
	err = httpServer.ListenAndServe()
	if err != nil {
		fmt.Println("HTTP server failed: %v", err)
	}
}
