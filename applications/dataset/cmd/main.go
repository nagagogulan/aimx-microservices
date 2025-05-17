package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/PecozQ/aimx-library/common"
	"github.com/joho/godotenv"
	base "whatsdare.com/fullstack/aimx/backend"
	"whatsdare.com/fullstack/aimx/backend/service"
	"whatsdare.com/fullstack/aimx/backend/worker"
)

func main() {
	// Create empty service (if dependencies are needed later, inject here)
	s := service.NewService()

	// Start the dataset path worker (processes file paths and sends chunks)
	go worker.StartDatasetPathWorker()

	// Get the current working directory (from where the command is run)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting current working directory:", err)
	}
	fmt.Println("Current Working Directory:", dir)

	// Construct the path to the .env file in the root directory
	envPath := filepath.Join(dir, "./.env")

	// Load the .env file from the correct path
	err = godotenv.Load(envPath)
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

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
		log.Fatalf("HTTP server failed: %v", err)
	}
}
