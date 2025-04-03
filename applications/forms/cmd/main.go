package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PecozQ/aimx-library/database/mongo"
)

func main() {
	client, err := mongo.InitDB(&mongo.Config{
		ConnectionURI:     "mongodb+srv://karthikyoki999:SmartWork123@goperla.qvnqj.mongodb.net",
		ConnectionOptions: "retryWrites=true&w=majority&tls=true&tlsAllowInvalidCertificates=false",
		DBName:            "0a8952d0-305f-4de9-ab9a-e6cdc5192ee7",
	})
	if err != nil {
		log.Fatalf("Error initializing DB: %v", err)
	}

	// Connect to MongoDB with a context and timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i := 0; i < 5; i++ {
		// Ping the MongoDB server to check the connection
		err = client.Ping(ctx, nil)
		if err == nil {
			fmt.Println("Successfully connected to MongoDB!")
			break
		}

		// Wait before retrying
		log.Printf("Ping failed: %v, retrying... (%d/5)", err, i+1)
		time.Sleep(2 * time.Second) // Retry delay
	}

	// Optionally, check if you can select a database
	// db := client.Database("0a8952d0-305f-4de9-ab9a-e6cdc5192ee7")
	// fmt.Printf("Using database: %s\n", db.Name())

	// Ensure the MongoDB client disconnects once done
	defer client.Disconnect(context.Background())
}
