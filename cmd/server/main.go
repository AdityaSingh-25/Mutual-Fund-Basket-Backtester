package main

import (
	"log"

	"MFBasketBacktester/config"
	"MFBasketBacktester/internal/db"
)

func main() {
	cfg := config.LoadConfig()

	db.InitDB(cfg.DBUrl)

	basketID, err := db.InsertBasket("Test Basket")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Inserted Basket ID:", basketID)

	basket, err := db.GetBasket(basketID)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Fetched Basket:", basket.Name)
}
