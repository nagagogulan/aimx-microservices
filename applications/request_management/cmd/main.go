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
	"github.com/PecozQ/aimx-library/database/pgsql"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/joho/godotenv"
	base "whatsdare.com/fullstack/aimx/backend"
	"whatsdare.com/fullstack/aimx/backend/service"
)

func main() {
	// DB, err := pgsql.InitDB(&pgsql.Config{
	// 	// my local host
	// 	DBHost:     "13.229.196.7",
	// 	DBPort:     5432,
	// 	DBUser:     "myappuser",
	// 	DBPassword: "SmartWork@123",
	// 	DBName:     "aimxdb",

	// 	// rds
	// 	// DBHost:     "18.142.238.70",
	// 	// DBPort:     5432,
	// 	// DBUser:     "myappuser",
	// 	// DBPassword: "SmartWork@123",
	// 	// DBName:     "aimxdb",

	// 	// build dev
	// 	// DBHost:     "localhost",
	// 	// DBPort:     5432,
	// 	// DBUser:     "postgres",
	// 	// DBPassword: "Admin",
	// 	// DBName:     "mylocaldb",
	// })

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

	requestRepo := repository.NewRequestRepository(DB)
	orgSettingRepo := repository.NewOrganizationSettingRepository(DB)

	s := service.NewRequestService(requestRepo, orgSettingRepo)
	endpoints := base.NewEndpoint(s)                // 💡 create Endpoints
	httpHandlers := base.MakeHTTPHandler(endpoints) // ✅ pass Endpoints to HTTP handler

	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(8087),
		Handler: service.CORS(http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`)),
	}

	fmt.Println("HTTP server started on port", 8087)
	log.Fatal(httpServer.ListenAndServe())
}
