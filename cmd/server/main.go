package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"MFBasketBacktester/config"
	"MFBasketBacktester/internal/api"
	"MFBasketBacktester/internal/cache"
	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/ingestion"
)

func main() {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatal("Configuration error: ", err)
	}
	if cfg.ClaudeAPIKey == "" {
		log.Println("Warning: CLAUDE_API_KEY not set; the /summary endpoint will be unavailable")
	}

	db.InitDB(cfg.DBUrl)
	cache.InitRedis(cfg.RedisUrl)

	// Keep NAV data fresh; AMFI publishes once per business day. The scheduler
	// goroutine is cancelled on shutdown so it does not outlive the process.
	schedulerCtx, stopScheduler := context.WithCancel(context.Background())
	ingestion.StartScheduler(schedulerCtx, 24*time.Hour)

	router := api.NewRouter(cfg)

	addr := ":" + cfg.Port
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// Run the server in the background so main can wait for a shutdown signal.
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server failed:", err)
		}
	}()

	// Block until an interrupt or termination signal arrives.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
	stopScheduler()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("Graceful shutdown failed:", err)
	}
	log.Println("Server stopped")
}
