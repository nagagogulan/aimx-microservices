package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/database/pgsql"
	"github.com/PecozQ/aimx-library/domain/repository"
	base "whatsdare.com/fullstack/aimx/backend"
	"whatsdare.com/fullstack/aimx/backend/service"
)

func main() {

	// DB, err := pgsql.InitDB(&pgsql.Config{
	// 	// my local host
	// 	// DBHost:     "localhost",
	// 	// DBPort:     5432,
	// 	// DBUser:     "postgres",
	// 	// DBPassword: "password@123",
	// 	// DBName:     "localDb",

	// 	// rds
	// 	DBHost:     "13.229.196.7",
	// 	DBPort:     5432,
	// 	DBUser:     "myappuser",
	// 	DBPassword: "SmartWork@123",
	// 	DBName:     "aimxdb",

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

	if err := pgsql.Migrate(DB); err != nil {
		log.Fatalf("Could not migrate database: %v", err)
	}
	defer sqlDB.Close()

	tempUserRepo := repository.NewUserserviceRepositoryService(DB)
	orgRepo := repository.NewOrganizationRepositoryService(DB)
	userRepo := repository.NewUserCRUDRepository(DB)
	roleRepo := repository.NewRoleRepositoryService(DB)

	s := service.NewService(tempUserRepo, orgRepo, userRepo, roleRepo)

	httpHandlers := base.MakeHTTPHandler(s)

	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(8081),
		Handler: http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`),
	}

	fmt.Println("Info", "v.0.0.1 => HTTP server started", "port", 8081)
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
