package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/database/pgsql"
	"github.com/PecozQ/aimx-library/domain/repository"
	base "whatsdare.com/fullstack/aimx/backend"
	"whatsdare.com/fullstack/aimx/backend/service"
)

func main() {
	DB, err := pgsql.InitDB(&pgsql.Config{
		// my local host
		DBHost:     "18.142.238.70",
		DBPort:     5432,
		DBUser:     "myappuser",
		DBPassword: "SmartWork@123",
		DBName:     "aimxdb",

		// rds
		// DBHost:     "18.142.238.70",
		// DBPort:     5432,
		// DBUser:     "myappuser",
		// DBPassword: "SmartWork@123",
		// DBName:     "aimxdb",

		// build dev
		// DBHost:     "localhost",
		// DBPort:     5432,
		// DBUser:     "postgres",
		// DBPassword: "Admin",
		// DBName:     "mylocaldb",
	})
	if err != nil {
		log.Fatalf("Error initializing DB: %v", err)
	}

	sqlDB, _ := DB.DB()
	defer sqlDB.Close()

	userRepo := repository.NewUserCRUDRepository(DB)
	s := service.NewService(userRepo)
	endpoints := base.NewEndpoint(s)                // 💡 create Endpoints
	httpHandlers := base.MakeHTTPHandler(endpoints) // ✅ pass Endpoints to HTTP handler

	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(8085),
		Handler: service.CORS(http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`)),
	}

	fmt.Println("HTTP server started on port", 8085)
	log.Fatal(httpServer.ListenAndServe())
}
