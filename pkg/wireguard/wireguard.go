package wireguard

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"wireguard-vpn-client-creater/pkg/config"
	"wireguard-vpn-client-creater/pkg/models"
)

// GetServerPublicKey - Server public key ni o'qish
func GetServerPublicKey() (string, error) {
	// Server public key ni o'qish
	publicKeyBytes, err := os.ReadFile(config.ServerPublicKeyPath)
	if err != nil {
		return "", fmt.Errorf("server public key o'qishda xatolik: %v", err)
	}
	return strings.TrimSpace(string(publicKeyBytes)), nil
}

// GenerateKeyPair - Yangi client uchun private va public key yaratish
func GenerateKeyPair() (string, string, error) {
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

// FindAvailableIP - Bo'sh IP manzilni topish
func FindAvailableIP(clientType models.ClientType, usedIPs []string) (string, error) {
	var subnetPrefix string

	// Client turiga qarab subnet tanlash
	if clientType == models.ClientTypeVIP {
		// VIP clientlar uchun 10.77.x.x subnet
		subnetPrefix = "10.77."
	} else {
		// Normal clientlar uchun 10.7.x.x subnet
		subnetPrefix = "10.7."
	}

	// IP manzillarni map ga o'tkazish (tezroq qidirish uchun)
	usedIPMap := make(map[string]bool)
	for _, ip := range usedIPs {
		usedIPMap[ip] = true
	}

	// Bo'sh IP manzilni topish
	// Avval 10.x.0.2 dan 10.x.0.251 gacha tekshirish
	for thirdOctet := 0; thirdOctet <= 255; thirdOctet++ {
		for fourthOctet := 2; fourthOctet <= 251; fourthOctet++ {
			candidateIP := fmt.Sprintf("%s%d.%d", subnetPrefix, thirdOctet, fourthOctet)

			// Agar bu IP manzil ishlatilmayotgan bo'lsa, uni qaytarish
			if !usedIPMap[candidateIP] {
				log.Printf("Yangi IP manzil yaratildi: %s", candidateIP)
				return candidateIP + "/32", nil
			}
		}
	}

	// Agar barcha IP manzillar band bo'lsa, xatolik qaytarish
	return "", fmt.Errorf("bo'sh IP manzil topilmadi")
}

// CreateClientConfig - Wireguard client konfiguratsiyasini yaratish
func CreateClientConfig(clientPrivateKey, presharedKey, clientIP, serverPublicKey string) (string, models.WireguardConfig) {
	// Client konfiguratsiyasi
	config := models.WireguardConfig{
		Endpoint:   fmt.Sprintf("%s:51820", config.ServerIP),
		Address:    clientIP,
		PrivateKey: clientPrivateKey,
		PublicKey:  serverPublicKey,
		DNS:        config.DNS,
		AllowedIPs: config.AllowedIPs,
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

// AddPeerToServer - Server konfiguratsiyasiga yangi peer qo'shish
func AddPeerToServer(clientPublicKey, clientIP, presharedKey string) error {
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
	cmd := exec.Command("wg", "set", config.InterfaceName, "peer", clientPublicKey, "preshared-key", tempPresharedKeyFile.Name(), "allowed-ips", clientIPWithoutCIDR+"/32")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("peer qo'shishda xatolik: %v, output: %s", err, string(output))
	}

	// O'zgarishlarni saqlash
	cmd = exec.Command("bash", "-c", fmt.Sprintf("wg-quick save %s", config.InterfaceName))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("konfiguratsiyani saqlashda xatolik: %v, output: %s", err, string(output))
	}

	return nil
}

// GeneratePresharedKey - Preshared key yaratish
func GeneratePresharedKey() (string, error) {
	// Preshared key yaratish
	cmd := exec.Command("wg", "genpsk")
	presharedKeyBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("preshared key yaratishda xatolik: %v", err)
	}
	presharedKey := strings.TrimSpace(string(presharedKeyBytes))
	return presharedKey, nil
}

// RemovePeerFromServer - Server konfiguratsiyasidan peerni o'chirish
func RemovePeerFromServer(publicKey string) error {
	// wg-quick orqali peerni o'chirish
	cmd := exec.Command("wg", "set", config.InterfaceName, "peer", publicKey, "remove")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("peerni o'chirishda xatolik: %v, output: %s", err, string(output))
	}

	// O'zgarishlarni saqlash
	cmd = exec.Command("bash", "-c", fmt.Sprintf("wg-quick save %s", config.InterfaceName))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("konfiguratsiyani saqlashda xatolik: %v, output: %s", err, string(output))
	}

	return nil
}

// ClientTraffic - Client traffic ma'lumotlari
type ClientTraffic struct {
	PublicKey       string `json:"public_key"`
	LatestHandshake string `json:"latest_handshake"`
	BytesReceived   int64  `json:"bytes_received"`
	BytesSent       int64  `json:"bytes_sent"`
	AllowedIPs      string `json:"allowed_ips"`
	Endpoint        string `json:"endpoint"`
}

// WgJsonOutput - wg-json buyrug'i natijasi
type WgJsonOutput map[string]WgInterface

// WgInterface - Wireguard interface ma'lumotlari
type WgInterface struct {
	PrivateKey string                `json:"privateKey"`
	PublicKey  string                `json:"publicKey"`
	ListenPort int                   `json:"listenPort"`
	Peers      map[string]WgPeerInfo `json:"peers"`
}

// WgPeerInfo - Wireguard peer ma'lumotlari
type WgPeerInfo struct {
	PresharedKey    string   `json:"presharedKey"`
	Endpoint        string   `json:"endpoint,omitempty"`
	LatestHandshake int64    `json:"latestHandshake,omitempty"`
	TransferRx      int64    `json:"transferRx,omitempty"`
	TransferTx      int64    `json:"transferTx,omitempty"`
	AllowedIps      []string `json:"allowedIps,omitempty"`
}

// GetWgJsonData - wg-json buyrug'i orqali ma'lumotlarni olish
func GetWgJsonData() (WgJsonOutput, error) {
	// wg-json buyrug'ini ishga tushirish
	cmd := exec.Command("wg-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wg-json buyrug'ini ishga tushirishda xatolik: %v", err)
	}

	// JSON ma'lumotlarini parse qilish
	var wgData WgJsonOutput
	if err := json.Unmarshal(output, &wgData); err != nil {
		return nil, fmt.Errorf("JSON ma'lumotlarini parse qilishda xatolik: %v", err)
	}

	return wgData, nil
}

// GetClientTraffic - Client traffic ma'lumotlarini olish
func GetClientTraffic(publicKey string) (*ClientTraffic, error) {
	// wg-json buyrug'i orqali ma'lumotlarni olish
	wgData, err := GetWgJsonData()
	if err != nil {
		return nil, err
	}

	// Interface ma'lumotlarini olish
	interfaceData, exists := wgData[config.InterfaceName]
	if !exists {
		return nil, fmt.Errorf("interface topilmadi: %s", config.InterfaceName)
	}

	// Peer ma'lumotlarini olish
	peerInfo, exists := interfaceData.Peers[publicKey]
	if !exists {
		return nil, fmt.Errorf("client topilmadi: %s", publicKey)
	}

	// Handshake vaqtini formatlash
	var latestHandshake string
	if peerInfo.LatestHandshake == 0 {
		latestHandshake = "Hech qachon"
	} else {
		handshakeTime := time.Unix(peerInfo.LatestHandshake, 0)
		latestHandshake = handshakeTime.Format(time.RFC3339)
	}

	// AllowedIPs ni string ga o'zgartirish
	allowedIPs := strings.Join(peerInfo.AllowedIps, ", ")
	if allowedIPs == "" {
		allowedIPs = "Mavjud emas"
	}

	// Traffic ma'lumotlarini qaytarish
	return &ClientTraffic{
		PublicKey:       publicKey,
		LatestHandshake: latestHandshake,
		BytesReceived:   peerInfo.TransferRx,
		BytesSent:       peerInfo.TransferTx,
		AllowedIPs:      allowedIPs,
		Endpoint:        peerInfo.Endpoint,
	}, nil
}

// GetAllClientsTraffic - Barcha clientlar traffic ma'lumotlarini olish
func GetAllClientsTraffic() (map[string]*ClientTraffic, error) {
	// wg-json buyrug'i orqali ma'lumotlarni olish
	wgData, err := GetWgJsonData()
	if err != nil {
		return nil, err
	}

	// Interface ma'lumotlarini olish
	interfaceData, exists := wgData[config.InterfaceName]
	if !exists {
		return nil, fmt.Errorf("interface topilmadi: %s", config.InterfaceName)
	}

	// Traffic ma'lumotlarini saqlash uchun map
	clientsTraffic := make(map[string]*ClientTraffic)

	// Har bir peer uchun traffic ma'lumotlarini olish
	for publicKey, peerInfo := range interfaceData.Peers {
		// Handshake vaqtini formatlash
		var latestHandshake string
		if peerInfo.LatestHandshake == 0 {
			latestHandshake = "Hech qachon"
		} else {
			handshakeTime := time.Unix(peerInfo.LatestHandshake, 0)
			latestHandshake = handshakeTime.Format(time.RFC3339)
		}

		// AllowedIPs ni string ga o'zgartirish
		allowedIPs := strings.Join(peerInfo.AllowedIps, ", ")
		if allowedIPs == "" {
			allowedIPs = "Mavjud emas"
		}

		// Traffic ma'lumotlarini saqlash
		clientsTraffic[publicKey] = &ClientTraffic{
			PublicKey:       publicKey,
			LatestHandshake: latestHandshake,
			BytesReceived:   peerInfo.TransferRx,
			BytesSent:       peerInfo.TransferTx,
			AllowedIPs:      allowedIPs,
			Endpoint:        peerInfo.Endpoint,
		}
	}

	return clientsTraffic, nil
}
