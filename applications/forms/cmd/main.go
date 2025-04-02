package main

import (
	"log"

	"github.com/PecozQ/aimx-library/database/mongo"
)

func main() {
	_, err := mongo.InitDB(&mongo.Config{
		DBHost:     "127.0.0.1",
		DBPort:     27017,
		DBUser:     "root",
		DBPassword: "root",
		DBName:     "test",
	})
	if err != nil {
		log.Fatalf("Error initializing DB: %v", err)
	}
}
