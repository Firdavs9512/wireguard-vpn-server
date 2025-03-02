package api

import (
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
	clientIP := wireguard.GenerateClientIP()

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

	// Databasedan o'chirish
	if err := database.DB.Delete(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Clientni databasedan o'chirishda xatolik: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client muvaffaqiyatli o'chirildi"})
}
