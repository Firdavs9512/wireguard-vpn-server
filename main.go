package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	DNS                 = "1.1.1.1, 8.8.8.8"
	AllowedIPs          = "0.0.0.0/0, ::/0"
	Endpoint            = "192.168.1.151:51820"
	PersistentKeepalive = 25
	InterfaceName       = "wg0"
	ServerPublicKeyPath = "/etc/wireguard/server_public.key"
	ServerIP            = "192.168.1.151"
)

// WireguardConfig - Wireguard konfiguratsiya ma'lumotlari
type WireguardConfig struct {
	Endpoint   string `json:"endpoint"`
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	DNS        string `json:"dns"`
	AllowedIPs string `json:"allowed_ips"`
}

// ClientResponse - API response strukturasi
type ClientResponse struct {
	Config string          `json:"config"`
	Data   WireguardConfig `json:"data"`
}

// Server public key ni o'qish
func getServerPublicKey() (string, error) {
	// Server public key ni o'qish
	publicKeyBytes, err := os.ReadFile(ServerPublicKeyPath)
	if err != nil {
		return "", fmt.Errorf("server public key o'qishda xatolik: %v", err)
	}
	return strings.TrimSpace(string(publicKeyBytes)), nil
}

// Yangi client uchun private va public key yaratish
func generateKeyPair() (string, string, error) {
	// Vaqtinchalik papka yaratish
	tempDir, err := os.MkdirTemp("", "wg-keys")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tempDir)

	// Private key yaratish
	cmd := exec.Command("wg", "genkey")
	privateKeyBytes, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("private key yaratishda xatolik: %v", err)
	}
	privateKey := strings.TrimSpace(string(privateKeyBytes))

	// Public key yaratish
	cmd = exec.Command("wg", "pubkey")
	cmd.Stdin = strings.NewReader(privateKey)
	publicKeyBytes, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("public key yaratishda xatolik: %v", err)
	}
	publicKey := strings.TrimSpace(string(publicKeyBytes))

	return privateKey, publicKey, nil
}

// Yangi IP manzil yaratish
func generateClientIP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("10.7.0.%d/32", rand.Intn(250)+2) // 10.0.0.2 - 10.0.0.251 oralig'ida
}

// Wireguard client konfiguratsiyasini yaratish
func createClientConfig(clientPrivateKey, presharedKey, clientIP, serverPublicKey string) (string, WireguardConfig) {
	// Client konfiguratsiyasi
	config := WireguardConfig{
		Endpoint:   fmt.Sprintf("%s:51820", ServerIP),
		Address:    clientIP,
		PrivateKey: clientPrivateKey,
		PublicKey:  serverPublicKey,
		DNS:        DNS,
		AllowedIPs: AllowedIPs,
	}

	// Wireguard konfiguratsiya fayli formati
	configText := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = %s

[Peer]
PublicKey = %s
PresharedKey = %s
AllowedIPs = %s
Endpoint = %s
PersistentKeepalive = 25
`, clientPrivateKey, clientIP, config.DNS, serverPublicKey, presharedKey, config.AllowedIPs, config.Endpoint)

	return configText, config
}

// Server konfiguratsiyasiga yangi peer qo'shish
func addPeerToServer(clientPublicKey, clientIP, presharedKey string) error {
	// IP manzilni formatlash (CIDR notatsiyasini olib tashlash)
	clientIPWithoutCIDR := strings.Split(clientIP, "/")[0]

	// Preshared key faylini yaratish
	tempPresharedKeyFile, err := os.CreateTemp("", "psk")
	if err != nil {
		return fmt.Errorf("vaqtinchalik preshared key fayli yaratishda xatolik: %v", err)
	}
	defer os.Remove(tempPresharedKeyFile.Name())

	_, err = tempPresharedKeyFile.WriteString(presharedKey)
	if err != nil {
		return fmt.Errorf("preshared key fayliga yozishda xatolik: %v", err)
	}
	tempPresharedKeyFile.Close()

	// wg-quick orqali yangi peer qo'shish
	cmd := exec.Command("wg", "set", InterfaceName, "peer", clientPublicKey, "preshared-key", tempPresharedKeyFile.Name(), "allowed-ips", clientIPWithoutCIDR+"/32")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("peer qo'shishda xatolik: %v, output: %s", err, string(output))
	}

	// O'zgarishlarni saqlash
	cmd = exec.Command("bash", "-c", fmt.Sprintf("wg-quick save %s", InterfaceName))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("konfiguratsiyani saqlashda xatolik: %v, output: %s", err, string(output))
	}

	return nil
}

func generatePresharedKey() (string, error) {
	// Preshared key yaratish
	cmd := exec.Command("wg", "genpsk")
	presharedKeyBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("preshared key yaratishda xatolik: %v", err)
	}
	presharedKey := strings.TrimSpace(string(presharedKeyBytes))
	return presharedKey, nil
}

func main() {
	r := gin.Default()

	// Client yaratish uchun endpoint
	r.POST("/api/client", func(c *gin.Context) {
		// Server public key ni o'qish
		serverPublicKey, err := getServerPublicKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Client uchun key pair yaratish
		clientPrivateKey, clientPublicKey, err := generateKeyPair()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Preshared key yaratish
		presharedKey, err := generatePresharedKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Client IP manzilini yaratish
		clientIP := generateClientIP()

		// Client konfiguratsiyasini yaratish
		configText, configData := createClientConfig(clientPrivateKey, presharedKey, clientIP, serverPublicKey)

		// Serverda client konfiguratsiyasini saqlash
		err = addPeerToServer(clientPublicKey, clientIP, presharedKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Natijani qaytarish
		response := ClientResponse{
			Config: configText,
			Data:   configData,
		}

		c.JSON(http.StatusOK, response)
	})

	// Serverni ishga tushirish
	log.Println("Server started on :8080")
	r.Run(":8080")
}
