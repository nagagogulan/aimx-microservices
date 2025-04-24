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
	// Database config - you can make this dynamic via env vars
	DB, err := pgsql.InitDB(&pgsql.Config{
		// my local host
		// DBHost:     "localhost",
		// DBPort:     5432,
		// DBUser:     "postgres",
		// DBPassword: "password@123",
		// DBName:     "localDb",

		// rds
		DBHost:     "18.142.238.70",
        DBPort:     5432,
        DBUser:     "myappuser",
        DBPassword: "SmartWork@123",
        DBName:     "aimxdb",

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

	// Initialize repositories
	roleRepo := repository.NewRoleRepositoryService(DB)
	moduleRepo := repository.NewModuleRepositoryService(DB)
	permissionRepo := repository.NewPermissionRepositoryService(DB)
	rmpRepo := repository.NewRMPRepositoryService(DB)

	// Initialize services
	roleService := service.NewRoleService(roleRepo)
	moduleService := service.NewModuleService(moduleRepo)
	permissionService := service.NewPermissionService(permissionRepo)
	rmpService := service.NewRMPService(rmpRepo)

	// Set up endpoints
	endpoints := base.NewRoleEndpoints(roleService, moduleService, permissionService, rmpService)

	// Set up HTTP handler
	httpHandlers := base.MakeRoleHTTPHandler(endpoints)

	// Start HTTP server
	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(8083),
		Handler: service.CORS(http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`)),
	}

	fmt.Println("Info", "Role service HTTP server started", "port", 8083)
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
