package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl        string
	RedisUrl     string
	ClaudeAPIKey string
	Port         string
}

// Validate reports whether the required configuration is present. DB_URL and
// REDIS_URL are mandatory; CLAUDE_API_KEY is optional (only /summary needs it)
// and PORT has a default.
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
		DBUrl:        getEnv("DB_URL", ""),
		RedisUrl:     getEnv("REDIS_URL", ""),
		ClaudeAPIKey: getEnv("CLAUDE_API_KEY", ""),
		Port:         getEnv("PORT", "8080"),
	}
}

func getEnv(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
