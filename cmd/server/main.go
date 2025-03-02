package main

import (
	"flag"
	"fmt"
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
				if err := database.DeleteExpiredClients(); err != nil {
					log.Printf("Muddati o'tgan clientlarni tekshirishda xatolik: %v", err)
				}
			}
		}
	}()
}

func main() {
	// Konfiguratsiya fayli yo'lini olish
	configPath := flag.String("config", "/etc/wireguard/server.yaml", "Konfiguratsiya fayli yo'li")
	createConfig := flag.Bool("create-config", false, "Default konfiguratsiya faylini yaratish")
	flag.Parse()

	// Default konfiguratsiya faylini yaratish
	if *createConfig {
		if err := config.CreateDefaultConfig(*configPath); err != nil {
			log.Fatalf("Default konfiguratsiya faylini yaratishda xatolik: %v", err)
		}
		log.Printf("Default konfiguratsiya fayli yaratildi: %s", *configPath)
		return
	}

	// Konfiguratsiya faylini o'qish
	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("Konfiguratsiya faylini o'qishda xatolik: %v", err)
	}
	log.Printf("Konfiguratsiya fayli o'qildi: %s", *configPath)

	// Databaseni ishga tushirish
	_, err := database.InitDB(config.Config.Database.Path)
	if err != nil {
		log.Fatalf("Database initializatsiyasida xatolik: %v", err)
	}

	// Muddati o'tgan clientlarni tekshirish schedulerini ishga tushirish
	startExpirationChecker()

	// Dastur ishga tushganda bir marta tekshirish
	if err := database.DeleteExpiredClients(); err != nil {
		log.Printf("Muddati o'tgan clientlarni tekshirishda xatolik: %v", err)
	}

	// API routerini sozlash
	r := api.SetupRouter()

	// Serverni ishga tushirish
	serverAddr := fmt.Sprintf(":%d", config.Config.API.Port)
	log.Printf("Server started on %s", serverAddr)
	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("Server ishga tushirishda xatolik: %v", err)
	}
}
