package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Default per-IP rate-limit settings: a burst of 10 requests, then 5 sustained
// requests per second. Override via RATE_LIMIT_RPS / RATE_LIMIT_BURST; set
// RATE_LIMIT_RPS to 0 to disable rate limiting (e.g. for load testing).
const (
	defaultRateLimitRPS   = 5
	defaultRateLimitBurst = 10
)

type Config struct {
	DBUrl          string
	RedisUrl       string
	Port           string
	RateLimitRPS   float64
	RateLimitBurst float64
}

// Validate reports whether the required configuration is present. DB_URL and
// REDIS_URL are mandatory, and PORT has a default.
func (c *Config) Validate() error {
	var missing []string
	if c.DBUrl == "" {
		missing = append(missing, "DB_URL")
	}
	if c.RedisUrl == "" {
		missing = append(missing, "REDIS_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s",
			strings.Join(missing, ", "))
	}
	return nil
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return &Config{
		DBUrl:          getEnv("DB_URL", ""),
		RedisUrl:       getEnv("REDIS_URL", ""),
		Port:           getEnv("PORT", "8080"),
		RateLimitRPS:   getEnvFloat("RATE_LIMIT_RPS", defaultRateLimitRPS),
		RateLimitBurst: getEnvFloat("RATE_LIMIT_BURST", defaultRateLimitBurst),
	}
}

func getEnv(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// getEnvFloat reads a non-negative float from the environment, falling back to
// defaultVal when the variable is unset or invalid.
func getEnvFloat(key string, defaultVal float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil && v >= 0 {
		return v
	}
	return defaultVal
}
