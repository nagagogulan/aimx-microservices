package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/database/pgsql"
	"github.com/PecozQ/aimx-library/domain/repository"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	// 	// DBHost:     "localhost",
	// 	// DBPort:     5432,
	// 	// DBUser:     "postgres",
	// 	// DBPassword: "password@123",
	// 	// DBName:     "localDb",

	// 	// build dev
	// 	// DBHost:     "localhost",
	// 	// DBPort:     5432,
	// 	// DBUser:     "postgres",
	// 	// DBPassword: "Admin",
	// 	// DBName:     "mylocaldb",
	// })
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

	uri := os.Getenv("MONGO_URI") // replace with your MongoDB URI

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Could not ping to MongoDB: %v", err)
	}

	fmt.Println("Successfully connected to MongoDB!")

	// Get a handle to a collection
	db := client.Database(os.Getenv("MONGO_DBNAME"))
	//collection := db.Collection("templates")

	userRepo := repository.NewUserCRUDRepository(DB)
	generalSettingRepo := repository.NewGeneralSettingRepository(DB)
	orgRepo := repository.NewOrganizationRepositoryService(DB)
	orgSettingRepo := repository.NewOrganizationSettingRepository(DB)
	formRepo := repository.NewFormRepository(db)

	s := service.NewService(userRepo, generalSettingRepo, orgRepo, orgSettingRepo, formRepo)
	httpHandlers := base.MakeHTTPHandler(s)

	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(8085),
		Handler: service.CORS(http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`)),
	}

	fmt.Println("HTTP server started on port", 8085)
	log.Fatal(httpServer.ListenAndServe())
}
