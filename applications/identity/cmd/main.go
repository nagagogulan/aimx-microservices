package main

import (
	"fmt"
	"log"

	"github.com/PecozQ/aimx-library/database/pgsql"
)

func main() {
	// Initialize the database connection using the configuration...
	// DB, err := pgsql.InitDB(&pgsql.Config{
	// 	DBHost:     os.Getenv("DB_HOST"),
	// 	DBPort:     dbPort,
	// 	DBUser:     os.Getenv("DB_USER"),
	// 	DBPassword: os.Getenv("DB_PASSWORD"),
	// 	DBName:     os.Getenv("DB_NAME"),
	// })

	DB, err := pgsql.InitDB(&pgsql.Config{
		DBHost:     "gp2-backend-rds.cxc4ic4mib6i.us-east-1.rds.amazonaws.com",
		DBPort:     5432,
		DBUser:     "postgres",
		DBPassword: "SmartWork123",
		DBName:     "postgres",
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

	// Close the DB connection when done (deferred)
	defer sqlDB.Close()
}
