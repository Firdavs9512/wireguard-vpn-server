package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"wireguard-vpn-client-creater/pkg/database"
	"wireguard-vpn-client-creater/pkg/models"
	"wireguard-vpn-client-creater/pkg/wireguard"
)

// SetupRouter - API routerini sozlash
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Client yaratish uchun endpoint
	r.POST("/api/client", CreateClientHandler)

	// Barcha clientlarni olish
	r.GET("/api/clients", GetAllClientsHandler)

	// Client ma'lumotlarini olish
	r.GET("/api/client/:id", GetClientHandler)

	// Clientni o'chirish
	r.DELETE("/api/client/:id", DeleteClientHandler)

	// Client life_time vaqtini olish
	r.GET("/api/client/:id/lifetime", GetClientLifetimeHandler)

	// Client life_time vaqtini yangilash
	r.PUT("/api/client/:id/lifetime", UpdateClientLifetimeHandler)

	// Client traffic ma'lumotlarini olish
	r.GET("/api/client/:id/traffic", GetClientTrafficHandler)

	// Barcha clientlar traffic ma'lumotlarini olish
	r.GET("/api/clients/traffic", GetAllClientsTrafficHandler)

	return r
}

// CreateClientHandler - Yangi client yaratish uchun handler
func CreateClientHandler(c *gin.Context) {
	// Requestdan ma'lumotlarni olish
	var request struct {
		Description string            `json:"description"`
		LifeTime    int               `json:"life_time"` // Soniya, 0 = cheksiz
		Type        models.ClientType `json:"type"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		// Agar ma'lumotlar berilmagan bo'lsa, default qiymatlarni ishlatamiz
		request.Description = ""
		request.LifeTime = 0
		request.Type = models.ClientTypeNormal
	}

	// Type ni tekshirish
	if request.Type != models.ClientTypeNormal && request.Type != models.ClientTypeVIP {
		request.Type = models.ClientTypeNormal
	}

	// Server public key ni o'qish
	serverPublicKey, err := wireguard.GetServerPublicKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Client uchun key pair yaratish
	clientPrivateKey, clientPublicKey, err := wireguard.GenerateKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Preshared key yaratish
	presharedKey, err := wireguard.GeneratePresharedKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Client IP manzilini yaratish
	// Databasedagi ishlatilayotgan IP manzillarni olish
	var subnetPrefix string
	if request.Type == models.ClientTypeVIP {
		subnetPrefix = "10.77."
	} else {
		subnetPrefix = "10.7."
	}

	usedIPs, err := database.GetUsedIPAddresses(subnetPrefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "IP manzillarni olishda xatolik: " + err.Error()})
		return
	}

	clientIP, err := wireguard.FindAvailableIP(request.Type, usedIPs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Bo'sh IP manzil topilmadi: " + err.Error()})
		return
	}

	// Client konfiguratsiyasini yaratish
	configText, configData := wireguard.CreateClientConfig(clientPrivateKey, presharedKey, clientIP, serverPublicKey)

	// Serverda client konfiguratsiyasini saqlash
	err = wireguard.AddPeerToServer(clientPublicKey, clientIP, presharedKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ExpiresAt ni hisoblash
	var expiresAt *time.Time
	if request.LifeTime > 0 {
		expiry := time.Now().Add(time.Duration(request.LifeTime) * time.Second)
		expiresAt = &expiry
	}

	// Clientni databasega saqlash
	client := &models.WireguardClient{
		PublicKey:     clientPublicKey,
		PrivateKey:    clientPrivateKey,
		PresharedKey:  presharedKey,
		Address:       clientIP,
		Endpoint:      configData.Endpoint,
		DNS:           configData.DNS,
		AllowedIPs:    configData.AllowedIPs,
		ConfigText:    configText,
		LastConnected: time.Now(),
		Description:   request.Description,
		Active:        true,
		Type:          request.Type,
		LifeTime:      request.LifeTime,
		ExpiresAt:     expiresAt,
	}

	if err := database.SaveClient(client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Clientni databasega saqlashda xatolik: " + err.Error()})
		return
	}

	// Natijani qaytarish
	response := models.ClientResponse{
		Config: configText,
		Data:   configData,
	}

	c.JSON(http.StatusOK, response)
}

// GetAllClientsHandler - Barcha clientlarni olish uchun handler
func GetAllClientsHandler(c *gin.Context) {
	clients, err := database.GetAllClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Clientlarni olishda xatolik: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, clients)
}

// GetClientHandler - Client ma'lumotlarini olish uchun handler
func GetClientHandler(c *gin.Context) {
	id := c.Param("id")

	var client models.WireguardClient
	if err := database.DB.First(&client, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client topilmadi"})
		return
	}

	c.JSON(http.StatusOK, client)
}

// DeleteClientHandler - Clientni o'chirish uchun handler
func DeleteClientHandler(c *gin.Context) {
	id := c.Param("id")

	var client models.WireguardClient
	if err := database.DB.First(&client, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client topilmadi"})
		return
	}

	// Wireguard konfiguratsiyasidan peer ni o'chirish
	if err := wireguard.RemovePeerFromServer(client.PublicKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Peerni o'chirishda xatolik: " + err.Error()})
		return
	}

	// Databasedan to'liq o'chirish (hard delete)
	if err := database.DB.Unscoped().Delete(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Clientni databasedan o'chirishda xatolik: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client muvaffaqiyatli o'chirildi"})
}

// GetClientLifetimeHandler - Client life_time vaqtini olish uchun handler
func GetClientLifetimeHandler(c *gin.Context) {
	id := c.Param("id")

	var client models.WireguardClient
	if err := database.DB.First(&client, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client topilmadi"})
		return
	}

	// Client life_time va qolgan vaqtni hisoblash
	var remainingTime int64 = 0
	if client.ExpiresAt != nil {
		// Qolgan vaqtni soniyalarda hisoblash
		remainingTime = client.ExpiresAt.Unix() - time.Now().Unix()
		if remainingTime < 0 {
			remainingTime = 0
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             client.ID,
		"description":    client.Description,
		"type":           client.Type,
		"life_time":      client.LifeTime,
		"expires_at":     client.ExpiresAt,
		"remaining_time": remainingTime,
	})
}

// UpdateClientLifetimeHandler - Client life_time vaqtini yangilash uchun handler
func UpdateClientLifetimeHandler(c *gin.Context) {
	id := c.Param("id")

	var client models.WireguardClient
	if err := database.DB.First(&client, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client topilmadi"})
		return
	}

	// Requestdan yangi life_time ni olish
	var request struct {
		LifeTime int `json:"life_time"` // Soniya, 0 = cheksiz
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Noto'g'ri so'rov formati"})
		return
	}

	// Yangi ExpiresAt ni hisoblash
	var expiresAt *time.Time
	if request.LifeTime > 0 {
		expiry := time.Now().Add(time.Duration(request.LifeTime) * time.Second)
		expiresAt = &expiry
	} else {
		expiresAt = nil // Cheksiz muddat
	}

	// Clientni yangilash
	client.LifeTime = request.LifeTime
	client.ExpiresAt = expiresAt

	if err := database.UpdateClient(&client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Clientni yangilashda xatolik: " + err.Error()})
		return
	}

	// Qolgan vaqtni hisoblash
	var remainingTime int64 = 0
	if client.ExpiresAt != nil {
		remainingTime = client.ExpiresAt.Unix() - time.Now().Unix()
		if remainingTime < 0 {
			remainingTime = 0
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             client.ID,
		"description":    client.Description,
		"type":           client.Type,
		"life_time":      client.LifeTime,
		"expires_at":     client.ExpiresAt,
		"remaining_time": remainingTime,
		"message":        "Client life_time vaqti muvaffaqiyatli yangilandi",
	})
}

// GetClientTrafficHandler - Client traffic ma'lumotlarini olish uchun handler
func GetClientTrafficHandler(c *gin.Context) {
	id := c.Param("id")

	// Clientni databasedan olish
	var client models.WireguardClient
	if err := database.DB.First(&client, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client topilmadi"})
		return
	}

	// Client traffic ma'lumotlarini olish
	traffic, err := wireguard.GetClientTraffic(client.PublicKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Traffic ma'lumotlarini olishda xatolik: " + err.Error()})
		return
	}

	// Traffic ma'lumotlarini qaytarish
	c.JSON(http.StatusOK, gin.H{
		"id":               client.ID,
		"description":      client.Description,
		"public_key":       client.PublicKey,
		"address":          client.Address,
		"type":             client.Type,
		"latest_handshake": traffic.LatestHandshake,
		"bytes_received":   traffic.BytesReceived,
		"bytes_sent":       traffic.BytesSent,
		"allowed_ips":      traffic.AllowedIPs,
		"endpoint":         traffic.Endpoint,
		// Qo'shimcha ma'lumotlar
		"bytes_received_formatted": formatBytes(traffic.BytesReceived),
		"bytes_sent_formatted":     formatBytes(traffic.BytesSent),
		"total_traffic":            formatBytes(traffic.BytesReceived + traffic.BytesSent),
	})
}

// GetAllClientsTrafficHandler - Barcha clientlar traffic ma'lumotlarini olish uchun handler
func GetAllClientsTrafficHandler(c *gin.Context) {
	// Barcha clientlarni databasedan olish
	clients, err := database.GetAllClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Clientlarni olishda xatolik: " + err.Error()})
		return
	}

	// Barcha clientlar traffic ma'lumotlarini olish
	allTraffic, err := wireguard.GetAllClientsTraffic()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Traffic ma'lumotlarini olishda xatolik: " + err.Error()})
		return
	}

	// Har bir client uchun traffic ma'lumotlarini birlashtirish
	var result []gin.H
	for _, client := range clients {
		// Client traffic ma'lumotlarini olish
		traffic, exists := allTraffic[client.PublicKey]
		if !exists {
			// Agar traffic ma'lumotlari topilmasa, bo'sh ma'lumotlar bilan davom etish
			traffic = &wireguard.ClientTraffic{
				PublicKey:       client.PublicKey,
				LatestHandshake: "Hech qachon",
				BytesReceived:   0,
				BytesSent:       0,
				AllowedIPs:      client.Address,
				Endpoint:        "Mavjud emas",
			}
		}

		// Client va traffic ma'lumotlarini birlashtirish
		result = append(result, gin.H{
			"id":               client.ID,
			"description":      client.Description,
			"public_key":       client.PublicKey,
			"address":          client.Address,
			"type":             client.Type,
			"latest_handshake": traffic.LatestHandshake,
			"bytes_received":   traffic.BytesReceived,
			"bytes_sent":       traffic.BytesSent,
			"allowed_ips":      traffic.AllowedIPs,
			"endpoint":         traffic.Endpoint,
			// Qo'shimcha ma'lumotlar
			"bytes_received_formatted": formatBytes(traffic.BytesReceived),
			"bytes_sent_formatted":     formatBytes(traffic.BytesSent),
			"total_traffic":            formatBytes(traffic.BytesReceived + traffic.BytesSent),
		})
	}

	c.JSON(http.StatusOK, result)
}

// formatBytes - Baytlarni odam o'qiy oladigan formatga o'zgartirish
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
