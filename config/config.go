package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl        string
	RedisUrl     string
	ClaudeAPIKey string
	Port         string
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
