// Demo is an integration stand, not the full platform backend.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"hackalem/ai"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "integration stand listen address")
	flag.Parse()
	cfg, err := ai.LoadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	service, err := ai.NewService(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              *addr,
		Handler:           ai.NewHandler(service),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      cfg.Timeout + 15*time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	shutdown := make(chan struct{})
	go func() {
		<-ctx.Done()
		deadline, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(deadline); err != nil {
			_ = server.Close()
		}
		close(shutdown)
	}()
	mode := "live configured; verify X-AI-Mode on each response"
	if cfg.ForceFallback || cfg.APIKey == "" || cfg.Model == "" {
		mode = "fallback"
	}
	log.Printf("AI integration stand listening on %s (%s)", *addr, mode)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-shutdown
}
