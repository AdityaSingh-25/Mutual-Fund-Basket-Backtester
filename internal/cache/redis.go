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

// redisTimeout bounds every cache operation so a slow or unreachable Redis
// degrades the request (cache miss) instead of stalling it.
const redisTimeout = 200 * time.Millisecond

// opCtx returns a context carrying the standard Redis-operation timeout.
// Callers must defer the returned cancel func.
func opCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(Ctx, redisTimeout)
}

func InitRedis(redisURL string) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Invalid Redis URL:", err)
	}

	RDB = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(Ctx, 5*time.Second)
	defer cancel()
	if _, err = RDB.Ping(ctx).Result(); err != nil {
		log.Fatal("Redis connection failed:", err)
	}

	log.Println("Connected to Redis")
}

func SetBacktestResult(key string, result models.BacktestResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	ctx, cancel := opCtx()
	defer cancel()

	return RDB.Set(ctx, key, data, 24*time.Hour).Err()
}

func GetBacktestResult(key string) (*models.BacktestResult, error) {
	ctx, cancel := opCtx()
	defer cancel()

	data, err := RDB.Get(ctx, key).Result()
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

// GetBacktestRaw returns the cached result as the raw JSON bytes stored by
// SetBacktestResult, skipping the unmarshal/marshal round trip. The hot
// cache-hit path writes these bytes straight to the response. A miss returns
// redis.Nil, which callers treat as "not cached".
func GetBacktestRaw(key string) ([]byte, error) {
	ctx, cancel := opCtx()
	defer cancel()

	return RDB.Get(ctx, key).Bytes()
}
