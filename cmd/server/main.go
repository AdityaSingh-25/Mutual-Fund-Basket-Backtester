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

	db.InitDB(cfg.DBUrl)
	cache.InitRedis(cfg.RedisUrl)

	// Keep NAV data fresh; AMFI publishes once per business day.
	ingestion.StartScheduler(24 * time.Hour)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("Graceful shutdown failed:", err)
	}
	log.Println("Server stopped")
}
