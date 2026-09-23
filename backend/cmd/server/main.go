package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ai "hackalem/ai"

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
	frontendOrigin := os.Getenv("FRONTEND_ORIGIN")
	if strings.TrimSpace(frontendOrigin) == "" {
		frontendOrigin = httpapi.DefaultFrontendOrigin
	}
	aiConfig, err := ai.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("load AI configuration: %v", err)
	}
	aiService, err := ai.NewService(aiConfig, nil)
	if err != nil {
		log.Fatalf("initialize AI service: %v", err)
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

	combined := http.NewServeMux()
	ai.RegisterRoutes(combined, aiService)
	combined.Handle("/", httpapi.NewRouter(taskRepository, teamRepository, proposalRepository))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           httpapi.WithCORS(combined, frontendOrigin),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      aiConfig.Timeout + 15*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("server listening on %s", server.Addr)
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	case <-shutdownContext.Done():
		log.Printf("shutdown signal received")
		shutdownTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownTimeout); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
		if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server failed during shutdown: %v", err)
		}
	}
}
