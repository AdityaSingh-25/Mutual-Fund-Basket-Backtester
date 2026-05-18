// Package testsupport provides shared helpers for integration tests that need
// a real Postgres and/or Redis instance. Tests call RequireDB / RequireRedis,
// which skip the test when the corresponding TEST_* environment variable is
// unset — so a plain `go test ./...` with no infrastructure still passes.
package testsupport

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"MFBasketBacktester/internal/cache"
	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/migrations"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var (
	dbOnce    sync.Once
	dbErr     error
	redisOnce sync.Once
	redisErr  error
	schemeSeq atomic.Int64
)

// RequireDB points the db package at the database named by TEST_DB_URL and
// ensures the schema exists. The calling test is skipped when TEST_DB_URL is
// not set.
func RequireDB(t testing.TB) {
	t.Helper()

	url := os.Getenv("TEST_DB_URL")
	if url == "" {
		t.Skip("TEST_DB_URL not set; skipping database integration test")
	}

	dbOnce.Do(func() {
		conn, err := sql.Open("postgres", url)
		if err != nil {
			dbErr = err
			return
		}
		if err := conn.Ping(); err != nil {
			dbErr = fmt.Errorf("ping test database: %w", err)
			return
		}
		db.DB = conn
		dbErr = db.RunMigrations(migrations.Files)
	})

	if dbErr != nil {
		t.Fatalf("test database setup failed: %v", dbErr)
	}
}

// RequireRedis points the cache package at the Redis named by TEST_REDIS_URL.
// The calling test is skipped when TEST_REDIS_URL is not set.
func RequireRedis(t testing.TB) {
	t.Helper()

	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set; skipping Redis integration test")
	}

	redisOnce.Do(func() {
		opt, err := redis.ParseURL(url)
		if err != nil {
			redisErr = err
			return
		}
		rdb := redis.NewClient(opt)
		if err := rdb.Ping(cache.Ctx).Err(); err != nil {
			redisErr = fmt.Errorf("ping test redis: %w", err)
			return
		}
		cache.RDB = rdb
	})

	if redisErr != nil {
		t.Fatalf("test redis setup failed: %v", redisErr)
	}
}

// InsertFund inserts a fund with a unique, collision-free scheme code and
// returns its id. The fund and any rows referencing it are removed when the
// test finishes.
func InsertFund(t testing.TB, name string) int {
	t.Helper()

	code := 90_000_000 + schemeSeq.Add(1)

	var id int
	err := db.DB.QueryRow(`
		INSERT INTO funds (scheme_code, scheme_name)
		VALUES ($1, $2)
		RETURNING id
	`, code, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert test fund: %v", err)
	}

	t.Cleanup(func() {
		db.DB.Exec(`DELETE FROM nav WHERE fund_id = $1`, id)
		db.DB.Exec(`DELETE FROM basket_items WHERE fund_id = $1`, id)
		db.DB.Exec(`DELETE FROM funds WHERE id = $1`, id)
	})

	return id
}

// InsertNAV adds one NAV record for a fund. date must be YYYY-MM-DD. The row
// is removed by the owning fund's cleanup.
func InsertNAV(t testing.TB, fundID int, date string, nav float64) {
	t.Helper()

	_, err := db.DB.Exec(`
		INSERT INTO nav (fund_id, nav, date)
		VALUES ($1, $2, $3)
		ON CONFLICT (fund_id, date) DO NOTHING
	`, fundID, nav, date)
	if err != nil {
		t.Fatalf("insert test nav: %v", err)
	}
}

// InsertBasket creates an empty basket and returns its id. The basket and its
// items are removed when the test finishes.
func InsertBasket(t testing.TB, name string) int {
	t.Helper()

	id, err := db.InsertBasket(name)
	if err != nil {
		t.Fatalf("insert test basket: %v", err)
	}

	t.Cleanup(func() {
		db.DB.Exec(`DELETE FROM basket_items WHERE basket_id = $1`, id)
		db.DB.Exec(`DELETE FROM baskets WHERE id = $1`, id)
	})

	return id
}
