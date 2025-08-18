package main

import (
	"context"
	"log"

	"backend/internal/server"
)

func main() {
	// Create context for the application
	ctx := context.Background()

	// Initialize the app with all dependencies wired
	app, cleanup, err := server.InitializeApp(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	defer cleanup()

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatalf("Failed to run app: %v", err)
	}
}
