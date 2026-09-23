package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"backend/internal/database"
	"backend/internal/httpapi"
	"backend/internal/repository"
	"backend/internal/seed"
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

	if err := database.InitSchema(db); err != nil {
		log.Fatalf("initialize database schema: %v", err)
	}

	taskRepository := repository.NewTaskRepository(db)
	teamRepository := repository.NewTeamRepository(db)
	proposalRepository := repository.NewProposalRepository(db)

	if strings.EqualFold(strings.TrimSpace(os.Getenv("SEED_DEMO_DATA")), "true") {
		if err := seed.Run(context.Background(), db); err != nil {
			log.Fatalf("seed demo data: %v", err)
		}
		log.Printf("demo data seeded")
	}

	server := &http.Server{
		Addr:              ":8080",
		Handler:           httpapi.NewRouter(taskRepository, teamRepository, proposalRepository),
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
