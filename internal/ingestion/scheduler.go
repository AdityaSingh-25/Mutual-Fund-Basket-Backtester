package ingestion

import (
	"context"
	"log"
	"time"
)

// StartScheduler performs an initial NAV ingestion and then repeats it on the
// given interval in a background goroutine. The goroutine stops when ctx is
// cancelled, so it does not outlive a graceful server shutdown.
func StartScheduler(ctx context.Context, interval time.Duration) {
	go func() {
		log.Println("Running initial NAV ingestion...")
		FetchAndStoreNAV()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("NAV ingestion scheduler stopped")
				return
			case <-ticker.C:
				log.Println("Running scheduled NAV ingestion...")
				FetchAndStoreNAV()
			}
		}
	}()
}
