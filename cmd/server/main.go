package main

import (
	"MFBasketBacktester/config"
	"fmt"
)

func main() {
	cfg := config.LoadConfig()

	fmt.Println("Config Loaded:")
	fmt.Println("DB:", cfg.DBUrl)
	fmt.Println("Redis:", cfg.RedisUrl)
	fmt.Println("Port:", cfg.Port)
}
