package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"backend/internal/database"
	"backend/internal/httpapi"
)

func main() {
	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = "./data/app.db"
	}

	db, err := database.Open(databasePath)
	if err != nil {
		log.Fatalf("initialize database: %v", err)
	}
	defer db.Close()

	server := &http.Server{
		Addr:              ":8080",
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
