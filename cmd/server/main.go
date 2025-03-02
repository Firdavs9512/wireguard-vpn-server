package main

import (
	"log"

	"wireguard-vpn-client-creater/internal/api"
	"wireguard-vpn-client-creater/pkg/config"
	"wireguard-vpn-client-creater/pkg/database"
)

func main() {
	// Databaseni ishga tushirish
	_, err := database.InitDB(config.DatabasePath)
	if err != nil {
		log.Fatalf("Database initializatsiyasida xatolik: %v", err)
	}

	// API routerini sozlash
	r := api.SetupRouter()

	// Serverni ishga tushirish
	log.Println("Server started on :8080")
	r.Run(":8080")
}
