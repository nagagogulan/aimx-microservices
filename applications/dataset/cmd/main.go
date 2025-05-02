package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/PecozQ/aimx-library/common"
	base "whatsdare.com/fullstack/aimx/backend"
	"whatsdare.com/fullstack/aimx/backend/service"
)

func main() {
	// Create empty service (if dependencies are needed later, inject here)
	s := service.NewService()

	// Create HTTP handlers
	httpHandlers := base.MakeHttpHandler(s)

	// Set up HTTP server
	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(8084),
		Handler: http.TimeoutHandler(httpHandlers, time.Duration(common.ServerTimeout)*time.Millisecond, `{"Error":"Server Execution Timeout"}`),
	}

	fmt.Println("Info", "HTTP server started", "port", 8084)

	// Start HTTP server
	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
