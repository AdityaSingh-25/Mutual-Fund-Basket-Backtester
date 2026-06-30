package db

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// Connection-pool defaults. The Go database/sql defaults are unbounded
// MaxOpenConns and only 2 idle connections, which under load either exhausts
// Postgres (default max_connections is 100) or churns connections on every
// query. These bounded defaults keep the pool warm and protect Postgres; each
// is overridable via the matching environment variable for load tuning.
const (
	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 25
	defaultConnMaxLifetime = 5 * time.Minute
	defaultConnMaxIdleTime = 5 * time.Minute
)

func InitDB(dbURL string) {
	var err error

	DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	DB.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", defaultMaxOpenConns))
	DB.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", defaultMaxIdleConns))
	DB.SetConnMaxLifetime(defaultConnMaxLifetime)
	DB.SetConnMaxIdleTime(defaultConnMaxIdleTime)

	err = DB.Ping()
	if err != nil {
		log.Fatal("Database unreachable:", err)
	}

	log.Println("Connected to Postgres")
}

// envInt reads a positive integer from the environment, falling back to def
// when the variable is unset or invalid.
func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}
