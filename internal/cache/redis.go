package cache

import (
	"MFBasketBacktester/internal/models"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RDB *redis.Client
	Ctx = context.Background()
)

func InitRedis(redisURL string) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Invalid Redis URL:", err)
	}

	RDB = redis.NewClient(opt)

	_, err = RDB.Ping(Ctx).Result()
	if err != nil {
		log.Fatal("Redis connection failed:", err)
	}

	log.Println("Connected to Redis")
}

func SetBacktestResult(key string, result models.BacktestResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return RDB.Set(
		Ctx,
		key,
		data,
		24*time.Hour,
	).Err()
}

func GetBacktestResult(key string) (*models.BacktestResult, error) {
	data, err := RDB.Get(Ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var result models.BacktestResult

	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
