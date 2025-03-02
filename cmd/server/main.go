package main

import (
	"log"
	"time"

	"wireguard-vpn-client-creater/internal/api"
	"wireguard-vpn-client-creater/pkg/config"
	"wireguard-vpn-client-creater/pkg/database"
)

// Muddati o'tgan clientlarni tekshirish uchun scheduler
func startExpirationChecker() {
	ticker := time.NewTicker(15 * time.Minute) // Har 15 minutda bir marta tekshirish
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("Muddati o'tgan clientlarni tekshirish...")
				if err := database.DeactivateExpiredClients(); err != nil {
					log.Printf("Muddati o'tgan clientlarni tekshirishda xatolik: %v", err)
				}
			}
		}
	}()
}

func main() {
	// Databaseni ishga tushirish
	_, err := database.InitDB(config.DatabasePath)
	if err != nil {
		log.Fatalf("Database initializatsiyasida xatolik: %v", err)
	}

	// Muddati o'tgan clientlarni tekshirish schedulerini ishga tushirish
	startExpirationChecker()

	// Dastur ishga tushganda bir marta tekshirish
	if err := database.DeactivateExpiredClients(); err != nil {
		log.Printf("Muddati o'tgan clientlarni tekshirishda xatolik: %v", err)
	}

	// API routerini sozlash
	r := api.SetupRouter()

	// Serverni ishga tushirish
	log.Println("Server started on :8080")
	r.Run(":8080")
}
