package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"wireguard-vpn-client-creater/pkg/models"
	"wireguard-vpn-client-creater/pkg/wireguard"
)

// SetupRouter - API routerini sozlash
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Client yaratish uchun endpoint
	r.POST("/api/client", CreateClientHandler)

	return r
}

// CreateClientHandler - Yangi client yaratish uchun handler
func CreateClientHandler(c *gin.Context) {
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

	// Natijani qaytarish
	response := models.ClientResponse{
		Config: configText,
		Data:   configData,
	}

	c.JSON(http.StatusOK, response)
}
