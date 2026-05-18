// Command backfill proactively loads full NAV history for every fund used in
// a basket, so backtests need not fetch it lazily on the first request.
//
// It is idempotent: funds that already have history are skipped.
package main

import (
	"log"
	"time"

	"MFBasketBacktester/config"
	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/ingestion"
	"MFBasketBacktester/migrations"
)

func main() {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatal("Configuration error: ", err)
	}

	db.InitDB(cfg.DBUrl)
	if err := db.RunMigrations(migrations.Files); err != nil {
		log.Fatal("Migration error: ", err)
	}

	fundIDs, err := db.BasketFundIDs()
	if err != nil {
		log.Fatal("Could not list basket funds: ", err)
	}
	if len(fundIDs) == 0 {
		log.Println("No funds in any basket; nothing to backfill")
		return
	}
	log.Printf("Backfilling NAV history for %d basket fund(s)...", len(fundIDs))

	var succeeded, failed int
	for _, id := range fundIDs {
		if err := ingestion.EnsureHistory(id); err != nil {
			log.Printf("fund %d: %v", id, err)
			failed++
		} else {
			succeeded++
		}
		time.Sleep(200 * time.Millisecond) // be polite to api.mfapi.in
	}

	log.Printf("Backfill complete: %d succeeded, %d failed", succeeded, failed)
}
