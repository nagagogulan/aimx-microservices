package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/PecozQ/aimx-library/database/pgsql"
	"github.com/PecozQ/aimx-library/domain/repository"
	base "whatsdare.com/fullstack/aimx/backend"
	com "whatsdare.com/fullstack/aimx/backend/common"
	"whatsdare.com/fullstack/aimx/backend/service"
)

func main() {

	DB, err := pgsql.InitDB(&pgsql.Config{
		DBHost:     "localhost",
		DBPort:     5432,
		DBUser:     "postgres",
		DBPassword: "SmartWork@123",
		DBName:     "mylocaldb",
	})
	if err != nil {
		log.Fatalf("Error initializing DB: %v", err)
	}

	// Ping the database to check if the connection is successful
	sqlDB, err := DB.DB() // Get the raw SQL database instance

	if err != nil {
		log.Fatalf("Error getting raw DB instance: %v", err)
	}

	// Attempt to ping the database to check if it's alive
	err = sqlDB.Ping()
	if err != nil {
		log.Fatalf("Failed to ping the database: %v", err)
	} else {
		fmt.Println("Database connection successful!")
	}
	err = pgsql.Migrate(DB)
	if err != nil {
		log.Fatalf("Could not migrate database: %v", err)
		return
	}

	// Close the DB connection when done (deferred)
	defer sqlDB.Close()

	userRepo := repository.NewUserserviceRepositoryService(DB)
	s := service.NewService(userRepo)
	httpHandlers := base.MakeHTTPHandler(s)

	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(com.HttpPort),
		Handler: http.TimeoutHandler(httpHandlers, time.Duration(com.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`),
	}

	fmt.Println("Info", "HTTP server started", "port", com.HttpPort)
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
