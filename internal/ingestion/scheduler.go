package ingestion

import (
	"log"
	"time"
)

// StartScheduler performs an initial NAV ingestion and then repeats it on the
// given interval in a background goroutine.
func StartScheduler(interval time.Duration) {
	go func() {
		log.Println("Running initial NAV ingestion...")
		FetchAndStoreNAV()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("Running scheduled NAV ingestion...")
			FetchAndStoreNAV()
		}
	}()
}
