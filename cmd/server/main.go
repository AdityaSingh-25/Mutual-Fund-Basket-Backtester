package main

import (
	"log"

	"MFBasketBacktester/config"
	"MFBasketBacktester/internal/cache"
	"MFBasketBacktester/internal/db"
)

func main() {
	cfg := config.LoadConfig()

	db.InitDB(cfg.DBUrl)

	cache.InitRedis(cfg.RedisUrl)

	log.Println("Server setup complete")
}

// package main

// import (
// 	"log"

// 	"MFBasketBacktester/config"
// 	"MFBasketBacktester/internal/cache"
// 	"MFBasketBacktester/internal/db"
// 	"MFBasketBacktester/internal/models"
// )

// func main() {
// 	cfg := config.LoadConfig()

// 	db.InitDB(cfg.DBUrl)

// 	cache.InitRedis(cfg.RedisUrl)

// 	result := models.BacktestResult{
// 		CAGR:       14.2,
// 		XIRR:       13.8,
// 		Drawdown:   22.1,
// 		FinalValue: 152340,
// 	}

// 	err := cache.SetBacktestResult("test_basket", result)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	cachedResult, err := cache.GetBacktestResult("test_basket")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	log.Println("Cached Result:", cachedResult)

// 	log.Println("Server setup complete")
// }
