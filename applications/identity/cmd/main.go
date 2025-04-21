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
		// DBHost:     "localhost",
		// DBPort:     5432,
		// DBUser:     "postgres",
		// DBPassword: "password@123",
		// DBName:     "localDb",
		DBHost:     "18.142.238.70",
		DBPort:     5432,
		DBUser:     "myappuser",
		DBPassword: "SmartWork@123",
		DBName:     "aimxdb",
	})
	if err != nil {
		log.Fatalf("Error initializing DB: %v", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Error getting raw DB instance: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		log.Fatalf("Failed to ping the database: %v", err)
	} else {
		fmt.Println("Database connection successful!")
	}
	// Call Migration from Library
	err = pgsql.Migrate(DB)
	if err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// Optional: migrate your role/module/permission/RMP tables here manually if needed
	// err = DB.AutoMigrate(&model.Role{}, &model.Module{}, &model.Permission{}, &model.RoleModulePermission{})

	defer sqlDB.Close()

	userRepo := repository.NewUserserviceRepositoryService(DB)
	s := service.NewService(userRepo)
	httpHandlers := base.MakeHTTPHandler(s)

	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(common.HttpPort),
		Handler: service.CORS(http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`)),
	}

	fmt.Println("Info", "HTTP server started", "port", common.HttpPort)
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
