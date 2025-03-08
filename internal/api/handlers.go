package api

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"wireguard-vpn-client-creater/pkg/config"
	"wireguard-vpn-client-creater/pkg/database"
	"wireguard-vpn-client-creater/pkg/models"
	"wireguard-vpn-client-creater/pkg/security"
	"wireguard-vpn-client-creater/pkg/wireguard"
)

// Global IP blocker
var ipBlocker *security.IPBlocker

// InitIPBlocker - IP bloklash tizimini ishga tushirish
func InitIPBlocker() error {
	// Konfiguratsiyadan IP bloklash sozlamalarini olish
	ipBlockerConfig := config.Config.Security.IPBlocker

	// Agar IP bloklash o'chirilgan bo'lsa, hech narsa qilmaslik
	if !ipBlockerConfig.Enabled {
		return nil
	}

	// Log fayli uchun papkani yaratish
	logDir := strings.Split(ipBlockerConfig.LogFilePath, "/")
	if len(logDir) > 1 {
		logDirPath := strings.Join(logDir[:len(logDir)-1], "/")
		if err := os.MkdirAll(logDirPath, 0755); err != nil {
			return fmt.Errorf("log papkasini yaratishda xatolik: %v", err)
		}
	}

	// IP bloklash tizimini yaratish
	var err error
	ipBlocker, err = security.NewIPBlocker(
		time.Duration(ipBlockerConfig.BlockDuration)*time.Minute,
		ipBlockerConfig.MaxAttempts,
		ipBlockerConfig.LogFilePath,
	)

	return err
}

// TokenAuthMiddleware - API token autentifikatsiyasi uchun middleware
func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// IP manzilni olish
		clientIP := c.ClientIP()

		// IP bloklash tizimi ishga tushirilgan bo'lsa, IP manzilni tekshirish
		if ipBlocker != nil && config.Config.Security.IPBlocker.Enabled {
			// IP manzil bloklangan bo'lsa, so'rovni rad etish
			if ipBlocker.IsBlocked(clientIP) {
				remainingTime := ipBlocker.GetRemainingBlockTime(clientIP)
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": fmt.Sprintf("IP manzil bloklangan. Qolgan vaqt: %s", remainingTime.Round(time.Second)),
				})
				c.Abort()
				return
			}
		}

		// Authorization headerini tekshirish
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// IP bloklash tizimi ishga tushirilgan bo'lsa, muvaffaqiyatsiz urinishni qayd qilish
			if ipBlocker != nil && config.Config.Security.IPBlocker.Enabled {
				ipBlocker.RecordFailedAttempt(clientIP, c.Request.UserAgent(), c.Request.URL.Path)
			}

			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header topilmadi"})
			c.Abort()
			return
		}

		// Bearer token formatini tekshirish
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// IP bloklash tizimi ishga tushirilgan bo'lsa, muvaffaqiyatsiz urinishni qayd qilish
			if ipBlocker != nil && config.Config.Security.IPBlocker.Enabled {
				ipBlocker.RecordFailedAttempt(clientIP, c.Request.UserAgent(), c.Request.URL.Path)
			}

			c.JSON(http.StatusUnauthorized, gin.H{"error": "Noto'g'ri Authorization format. 'Bearer TOKEN' formatida bo'lishi kerak"})
			c.Abort()
			return
		}

		// Tokenni tekshirish
		token := parts[1]
		if token != config.Config.API.Token {
			// IP bloklash tizimi ishga tushirilgan bo'lsa, muvaffaqiyatsiz urinishni qayd qilish
			if ipBlocker != nil && config.Config.Security.IPBlocker.Enabled {
				ipBlocker.RecordFailedAttempt(clientIP, c.Request.UserAgent(), c.Request.URL.Path)
			}

			c.JSON(http.StatusUnauthorized, gin.H{"error": "Noto'g'ri token"})
			c.Abort()
			return
		}

		// Token to'g'ri bo'lsa, muvaffaqiyatsiz urinishlar sonini nolga tushirish
		if ipBlocker != nil && config.Config.Security.IPBlocker.Enabled {
			ipBlocker.ResetFailedAttempts(clientIP)
		}

		c.Next()
	}
}

// SetupRouter - API routerini sozlash
func SetupRouter() *gin.Engine {
	// Debug rejimini tekshirish
	if !config.Config.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// API routerlari
	api := r.Group("/api")
	api.Use(TokenAuthMiddleware()) // Barcha API so'rovlari uchun token autentifikatsiyasi

	// Client API endpointlari
	api.POST("/client", CreateClientHandler)
	api.GET("/clients", GetAllClientsHandler)
	api.GET("/client/:id", GetClientHandler)
	api.DELETE("/client/:id", DeleteClientHandler)
	api.GET("/client/:id/lifetime", GetClientLifetimeHandler)
	api.PUT("/client/:id/lifetime", UpdateClientLifetimeHandler)
	api.GET("/client/:id/traffic", GetClientTrafficHandler)
	api.GET("/clients/traffic", GetAllClientsTrafficHandler)

	// Server holati API endpointi
	api.GET("/server/status", GetServerStatusHandler)
	api.GET("/health", GetHealthHandler)

	return r
}

// CreateClientHandler - yangi client yaratish
func CreateClientHandler(c *gin.Context) {
	// Request bodyni o'qish
	var req struct {
		Description string `json:"description"`
		LifeTime    int    `json:"life_time"`
		Type        string `json:"type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Noto'g'ri so'rov formati"})
		return
	}

	// Type ni tekshirish
	if req.Type != "" && req.Type != "normal" && req.Type != "vip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Type faqat 'normal' yoki 'vip' bo'lishi mumkin"})
		return
	}

	// Default qiymatlarni o'rnatish
	if req.Type == "" {
		req.Type = "normal"
	}

	// ClientType ga o'zgartirish
	var clientType models.ClientType
	if req.Type == "vip" {
		clientType = models.ClientTypeVIP
	} else {
		clientType = models.ClientTypeNormal
	}

	// Ishlatilgan IP manzillarni olish
	usedIPs, err := database.GetUsedIPAddresses("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ishlatilgan IP manzillarni olishda xatolik: %v", err)})
		return
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
	clientIP, err := wireguard.FindAvailableIP(clientType, usedIPs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Bo'sh IP manzil topilmadi: " + err.Error()})
		return
	}

	// Client obyektini yaratish
	client := &models.WireguardClient{
		PublicKey:    clientPublicKey,
		PrivateKey:   clientPrivateKey,
		PresharedKey: presharedKey,
		Address:      clientIP,
		Description:  req.Description,
		Active:       true,
		Type:         clientType,
		LifeTime:     req.LifeTime,
		Endpoint:     fmt.Sprintf("%s:%d", config.Config.Server.IP, config.Config.Server.Port),
		DNS:          config.Config.Wireguard.DNS,
		AllowedIPs:   config.Config.Wireguard.AllowedIPs,
	}

	// ExpiresAt ni hisoblash
	if req.LifeTime > 0 {
		expiresAt := time.Now().Add(time.Duration(req.LifeTime) * time.Second)
		client.ExpiresAt = &expiresAt
	}

	// Serverda client konfiguratsiyasini saqlash
	err = wireguard.AddPeerToServer(clientPublicKey, clientIP, presharedKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Clientni databasega saqlash
	if err := database.SaveClient(client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Clientni saqlashda xatolik: %v", err)})
		return
	}

	// Client konfiguratsiyasini yaratish
	configText, _ := wireguard.CreateClientConfig(clientPrivateKey, presharedKey, clientIP, serverPublicKey)

	// Natijani qaytarish
	c.JSON(http.StatusOK, gin.H{
		"config": configText,
		"data":   client,
	})
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

// GetClientTrafficHandler - Client traffic ma'lumotlarini olish
func GetClientTrafficHandler(c *gin.Context) {
	// Client ID ni olish
	clientID := c.Param("id")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID ko'rsatilmagan"})
		return
	}

	// ID ni uint ga o'zgartirish
	id, err := strconv.ParseUint(clientID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Noto'g'ri ID formati"})
		return
	}

	// Clientni databasedan olish
	client, err := database.GetClientByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client topilmadi"})
		return
	}

	// Client traffic ma'lumotlarini olish
	traffic, err := wireguard.GetClientTraffic(client.PublicKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Traffic ma'lumotlarini olishda xatolik: %v", err)})
		return
	}

	// Natijani qaytarish
	c.JSON(http.StatusOK, gin.H{
		"client_id":                client.ID,
		"description":              client.Description,
		"public_key":               client.PublicKey,
		"address":                  client.Address,
		"latest_handshake":         traffic.LatestHandshake,
		"bytes_received":           traffic.BytesReceived,
		"bytes_sent":               traffic.BytesSent,
		"bytes_received_formatted": traffic.BytesReceivedFormatted,
		"bytes_sent_formatted":     traffic.BytesSentFormatted,
		"allowed_ips":              traffic.AllowedIPs,
	})
}

// GetAllClientsTrafficHandler - Barcha clientlar traffic ma'lumotlarini olish
func GetAllClientsTrafficHandler(c *gin.Context) {
	// Barcha clientlarni databasedan olish
	clients, err := database.GetAllClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Clientlarni olishda xatolik: %v", err)})
		return
	}

	// Barcha clientlar traffic ma'lumotlarini olish
	trafficList, err := wireguard.GetAllClientsTraffic()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Traffic ma'lumotlarini olishda xatolik: %v", err)})
		return
	}

	// Traffic ma'lumotlarini client ma'lumotlari bilan birlashtirish
	var result []gin.H
	for _, client := range clients {
		// Client uchun traffic ma'lumotlarini topish
		var clientTraffic *wireguard.ClientTraffic
		for _, traffic := range trafficList {
			if traffic.PublicKey == client.PublicKey {
				clientTraffic = traffic
				break
			}
		}

		// Agar traffic ma'lumotlari topilmagan bo'lsa, default qiymatlarni ishlatish
		if clientTraffic == nil {
			// Yangi client uchun default traffic ma'lumotlari
			result = append(result, gin.H{
				"client_id":                client.ID,
				"description":              client.Description,
				"public_key":               client.PublicKey,
				"address":                  client.Address,
				"latest_handshake":         time.Time{},
				"bytes_received":           int64(0),
				"bytes_sent":               int64(0),
				"bytes_received_formatted": "0 B",
				"bytes_sent_formatted":     "0 B",
				"allowed_ips":              client.Address,
			})
		} else {
			// Traffic ma'lumotlari bilan client ma'lumotlarini birlashtirish
			result = append(result, gin.H{
				"client_id":                client.ID,
				"description":              client.Description,
				"public_key":               client.PublicKey,
				"address":                  client.Address,
				"latest_handshake":         clientTraffic.LatestHandshake,
				"bytes_received":           clientTraffic.BytesReceived,
				"bytes_sent":               clientTraffic.BytesSent,
				"bytes_received_formatted": clientTraffic.BytesReceivedFormatted,
				"bytes_sent_formatted":     clientTraffic.BytesSentFormatted,
				"allowed_ips":              clientTraffic.AllowedIPs,
			})
		}
	}

	// Natijani qaytarish
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

// GetServerStatusHandler - Server holatini olish uchun handler
func GetServerStatusHandler(c *gin.Context) {
	// Server holatini olish
	status, err := wireguard.GetServerStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Server holatini olishda xatolik: %v", err)})
		return
	}

	// Databasedan clientlar sonini olish
	clients, err := database.GetAllClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Clientlarni olishda xatolik: %v", err)})
		return
	}

	// Natijani qaytarish
	c.JSON(http.StatusOK, gin.H{
		"server": status,
		"database": gin.H{
			"total_clients":    len(clients),
			"active_clients":   status.ActiveClients,
			"inactive_clients": len(clients) - status.ActiveClients,
		},
		"system": gin.H{
			"uptime": status.Uptime,
		},
		"traffic": gin.H{
			"total_bytes_received":           status.TotalBytesReceived,
			"total_bytes_sent":               status.TotalBytesSent,
			"total_traffic":                  status.TotalTraffic,
			"total_bytes_received_formatted": status.TotalBytesReceivedFormatted,
			"total_bytes_sent_formatted":     status.TotalBytesSentFormatted,
			"total_traffic_formatted":        status.TotalTrafficFormatted,
		},
	})
}

func GetHealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
