package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	// client, err := mongo.InitDB(&mongo.Config{
	// 	ConnectionURI:     "mongodb+srv://karthikyoki999:SmartWork123@goperla.qvnqj.mongodb.net",
	// 	ConnectionOptions: "retryWrites=true&w=majority&tls=true&tlsAllowInvalidCertificates=false",
	// 	DBName:            "0a8952d0-305f-4de9-ab9a-e6cdc5192ee7",
	// })
	// clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	// Connect to MongoDB
	// client, err := mongo.Connect(context.Background(), clientOptions)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// if err != nil {
	// 	log.Fatalf("Error initializing DB: %v", err)
	// }

	// Connect to MongoDB with a context and timeout
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	// for i := 0; i < 5; i++ {
	// 	Ping the MongoDB server to check the connection
	// 	err = client.Ping(ctx, nil)
	// 	if err == nil {
	// 		fmt.Println("Successfully connected to MongoDB!")
	// 		break
	// 	}

	// 	Wait before retrying
	// 	log.Printf("Ping failed: %v, retrying... (%d/5)", err, i+1)
	// 	time.Sleep(2 * time.Second) // Retry delay
	// }

	// Optionally, check if you can select a database
	// db := client.Database("0a8952d0-305f-4de9-ab9a-e6cdc5192ee7")
	// fmt.Printf("Using database: %s\n", db.Name())

	// Ensure the MongoDB client disconnects once done
	// defer client.Disconnect(context.Background())
	// Set MongoDB URI

	DB, err := pgsql.InitDB(&pgsql.Config{
		DBHost:     "18.142.238.70",
		DBPort:     5432,
		DBUser:     "myappuser",
		DBPassword: "SmartWork@123",
		DBName:     "aimxdb",
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

	uri := "mongodb+srv://nithiyavelmurugan:W5Tzzye77q3KXOrm@cluster0.tpgmrey.mongodb.net/?retryWrites=true&tls=true&w=majority&appName=Cluster0" // replace with your MongoDB URI

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
	db := client.Database("mydb")
	//collection := db.Collection("templates")

	templateRepo := repository.NewTemplateRepository(db)
	formRepo := repository.NewFormRepository(db)
	formTypeRepo := repository.NewFormTypeRepo(db)
	organizationRepo := repository.NewOrganizationRepositoryService(DB)

	s := service.NewService(templateRepo, formRepo, formTypeRepo, organizationRepo)
	httpHandlers := base.MakeHttpHandler(s)

	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(8585),
		Handler: service.CORS(http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`)),
	}

	fmt.Println("Info", "HTTP server started", "port", common.HttpPort)
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
