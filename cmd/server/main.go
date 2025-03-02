package main

import (
	"log"

	"wireguard-vpn-client-creater/internal/api"
)

func main() {
	// API routerini sozlash
	r := api.SetupRouter()

	// Serverni ishga tushirish
	log.Println("Server started on :8080")
	r.Run(":8080")
}
