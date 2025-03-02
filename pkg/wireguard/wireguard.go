package wireguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"wireguard-vpn-client-creater/pkg/config"
	"wireguard-vpn-client-creater/pkg/models"
)

// GetServerPublicKey - Server public key ni o'qish
func GetServerPublicKey() (string, error) {
	// Konfiguratsiyadan server public key faylini olish
	serverPublicKeyPath := config.Config.Wireguard.ServerPublicKeyPath

	// Agar fayl ko'rsatilmagan bo'lsa, default qiymatni ishlatish
	if serverPublicKeyPath == "" {
		serverPublicKeyPath = "/etc/wireguard/server_public.key"
	}

	// Faylni o'qish
	publicKey, err := os.ReadFile(serverPublicKeyPath)
	if err != nil {
		return "", fmt.Errorf("server public key faylini o'qishda xatolik: %v", err)
	}

	// Qator so'ngidagi bo'sh joylarni olib tashlash
	return strings.TrimSpace(string(publicKey)), nil
}

// GenerateKeyPair - Yangi client uchun private va public key yaratish
func GenerateKeyPair() (string, string, error) {
	// Private key yaratish
	privateKeyCmd := exec.Command("wg", "genkey")
	privateKeyOutput, err := privateKeyCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("private key yaratishda xatolik: %v", err)
	}
	privateKey := strings.TrimSpace(string(privateKeyOutput))

	// Private keydan public key yaratish
	publicKeyCmd := exec.Command("wg", "pubkey")
	publicKeyCmd.Stdin = bytes.NewBufferString(privateKey)
	publicKeyOutput, err := publicKeyCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("public key yaratishda xatolik: %v", err)
	}
	publicKey := strings.TrimSpace(string(publicKeyOutput))

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
	// Endpoint yaratish
	endpoint := fmt.Sprintf("%s:%d", config.Config.Server.IP, config.Config.Server.Port)

	// Client konfiguratsiyasi
	clientConfig := models.WireguardConfig{
		Endpoint:   endpoint,
		Address:    clientIP,
		PrivateKey: clientPrivateKey,
		PublicKey:  serverPublicKey,
		DNS:        config.Config.Wireguard.DNS,
		AllowedIPs: config.Config.Wireguard.AllowedIPs,
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
PersistentKeepalive = %d
`, clientPrivateKey, clientIP, config.Config.Wireguard.DNS, serverPublicKey, presharedKey, config.Config.Wireguard.AllowedIPs, endpoint, config.Config.Wireguard.PersistentKeepalive)

	return configText, clientConfig
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
	cmd := exec.Command("wg", "set", config.Config.Server.Interface, "peer", clientPublicKey, "preshared-key", tempPresharedKeyFile.Name(), "allowed-ips", clientIPWithoutCIDR+"/32")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("peer qo'shishda xatolik: %v, output: %s", err, string(output))
	}

	// O'zgarishlarni saqlash
	cmd = exec.Command("bash", "-c", fmt.Sprintf("wg-quick save %s", config.Config.Server.Interface))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("konfiguratsiyani saqlashda xatolik: %v, output: %s", err, string(output))
	}

	return nil
}

// GeneratePresharedKey - Preshared key yaratish
func GeneratePresharedKey() (string, error) {
	// Preshared key yaratish
	presharedKeyCmd := exec.Command("wg", "genpsk")
	presharedKeyOutput, err := presharedKeyCmd.Output()
	if err != nil {
		return "", fmt.Errorf("preshared key yaratishda xatolik: %v", err)
	}
	presharedKey := strings.TrimSpace(string(presharedKeyOutput))
	return presharedKey, nil
}

// RemovePeerFromServer - Server konfiguratsiyasidan peerni o'chirish
func RemovePeerFromServer(publicKey string) error {
	// wg-quick orqali peerni o'chirish
	cmd := exec.Command("wg", "set", config.Config.Server.Interface, "peer", publicKey, "remove")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("peerni o'chirishda xatolik: %v, output: %s", err, string(output))
	}

	// O'zgarishlarni saqlash
	cmd = exec.Command("bash", "-c", fmt.Sprintf("wg-quick save %s", config.Config.Server.Interface))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("konfiguratsiyani saqlashda xatolik: %v, output: %s", err, string(output))
	}

	return nil
}

// ClientTraffic - Client traffic ma'lumotlari
type ClientTraffic struct {
	PublicKey              string    `json:"public_key"`
	LatestHandshake        time.Time `json:"latest_handshake"`
	BytesReceived          int64     `json:"bytes_received"`
	BytesSent              int64     `json:"bytes_sent"`
	AllowedIPs             string    `json:"allowed_ips"`
	BytesReceivedFormatted string    `json:"bytes_received_formatted"`
	BytesSentFormatted     string    `json:"bytes_sent_formatted"`
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
	// Konfiguratsiyadan interface nomini olish
	interfaceName := config.Config.Server.Interface

	// Default qiymatni o'rnatish
	if interfaceName == "" {
		interfaceName = "wg0"
	}

	// wg komandasi orqali traffic ma'lumotlarini olish
	cmd := exec.Command("wg", "show", interfaceName, "dump")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("traffic ma'lumotlarini olishda xatolik: %v", err)
	}

	// Natijani qayta ishlash
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[0] == publicKey {
			// Handshake vaqtini o'zgartirish
			handshakeUnix, _ := strconv.ParseInt(fields[4], 10, 64)
			var handshakeTime time.Time
			if handshakeUnix > 0 {
				handshakeTime = time.Unix(handshakeUnix, 0)
			}

			// Traffic ma'lumotlarini o'zgartirish
			bytesReceived, _ := strconv.ParseInt(fields[5], 10, 64)
			bytesSent, _ := strconv.ParseInt(fields[6], 10, 64)

			// AllowedIPs ni olish
			allowedIPs := ""
			if len(fields) >= 8 {
				allowedIPs = fields[7]
			}

			// Traffic obyektini yaratish
			traffic := &ClientTraffic{
				PublicKey:              publicKey,
				LatestHandshake:        handshakeTime,
				BytesReceived:          bytesReceived,
				BytesSent:              bytesSent,
				AllowedIPs:             allowedIPs,
				BytesReceivedFormatted: formatBytes(bytesReceived),
				BytesSentFormatted:     formatBytes(bytesSent),
			}

			return traffic, nil
		}
	}

	return nil, fmt.Errorf("client topilmadi")
}

// GetAllClientsTraffic - Barcha clientlar traffic ma'lumotlarini olish
func GetAllClientsTraffic() ([]*ClientTraffic, error) {
	// Konfiguratsiyadan interface nomini olish
	interfaceName := config.Config.Server.Interface

	// Default qiymatni o'rnatish
	if interfaceName == "" {
		interfaceName = "wg0"
	}

	// wg komandasi orqali traffic ma'lumotlarini olish
	cmd := exec.Command("wg", "show", interfaceName, "dump")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("traffic ma'lumotlarini olishda xatolik: %v", err)
	}

	// Natijani qayta ishlash
	var trafficList []*ClientTraffic
	lines := strings.Split(string(output), "\n")

	// Birinchi qatorni o'tkazib yuborish (interface ma'lumotlari)
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 7 {
			// Handshake vaqtini o'zgartirish
			handshakeUnix, _ := strconv.ParseInt(fields[4], 10, 64)
			var handshakeTime time.Time
			if handshakeUnix > 0 {
				handshakeTime = time.Unix(handshakeUnix, 0)
			}

			// Traffic ma'lumotlarini o'zgartirish
			bytesReceived, _ := strconv.ParseInt(fields[5], 10, 64)
			bytesSent, _ := strconv.ParseInt(fields[6], 10, 64)

			// AllowedIPs ni olish
			allowedIPs := ""
			if len(fields) >= 8 {
				allowedIPs = fields[7]
			}

			// Traffic obyektini yaratish
			traffic := &ClientTraffic{
				PublicKey:              fields[0],
				LatestHandshake:        handshakeTime,
				BytesReceived:          bytesReceived,
				BytesSent:              bytesSent,
				AllowedIPs:             allowedIPs,
				BytesReceivedFormatted: formatBytes(bytesReceived),
				BytesSentFormatted:     formatBytes(bytesSent),
			}

			trafficList = append(trafficList, traffic)
		}
	}

	return trafficList, nil
}

// formatBytes - Baytlarni o'qish uchun qulay formatga o'zgartirish
func formatBytes(bytes int64) string {
	const (
		_          = iota
		KB float64 = 1 << (10 * iota)
		MB
		GB
		TB
	)

	var size float64
	var unit string

	switch {
	case bytes >= int64(TB):
		size = float64(bytes) / TB
		unit = "TB"
	case bytes >= int64(GB):
		size = float64(bytes) / GB
		unit = "GB"
	case bytes >= int64(MB):
		size = float64(bytes) / MB
		unit = "MB"
	case bytes >= int64(KB):
		size = float64(bytes) / KB
		unit = "KB"
	default:
		size = float64(bytes)
		unit = "B"
	}

	return fmt.Sprintf("%.2f %s", size, unit)
}

// ServerStatus - Server holati ma'lumotlari
type ServerStatus struct {
	InterfaceName               string    `json:"interface_name"`
	ListenPort                  int       `json:"listen_port"`
	PublicKey                   string    `json:"public_key"`
	PrivateKey                  string    `json:"private_key,omitempty"`
	ActiveClients               int       `json:"active_clients"`
	TotalClients                int       `json:"total_clients"`
	TotalBytesReceived          int64     `json:"total_bytes_received"`
	TotalBytesSent              int64     `json:"total_bytes_sent"`
	TotalTraffic                int64     `json:"total_traffic"`
	LastHandshake               time.Time `json:"last_handshake"`
	Uptime                      string    `json:"uptime"`
	TotalBytesReceivedFormatted string    `json:"total_bytes_received_formatted"`
	TotalBytesSentFormatted     string    `json:"total_bytes_sent_formatted"`
	TotalTrafficFormatted       string    `json:"total_traffic_formatted"`
}

// GetServerStatus - Server holatini olish
func GetServerStatus() (*ServerStatus, error) {
	// Konfiguratsiyadan interface nomini olish
	interfaceName := config.Config.Server.Interface

	// Default qiymatni o'rnatish
	if interfaceName == "" {
		interfaceName = "wg0"
	}

	// wg komandasi orqali server ma'lumotlarini olish
	cmd := exec.Command("wg", "show", interfaceName, "dump")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("server ma'lumotlarini olishda xatolik: %v", err)
	}

	// Natijani qayta ishlash
	lines := strings.Split(string(output), "\n")
	if len(lines) < 1 {
		return nil, fmt.Errorf("server ma'lumotlari topilmadi")
	}

	// Server ma'lumotlarini olish (birinchi qator)
	serverFields := strings.Fields(lines[0])
	if len(serverFields) < 3 {
		return nil, fmt.Errorf("server ma'lumotlari formati noto'g'ri")
	}

	// Server ma'lumotlarini o'zgartirish
	publicKey := serverFields[1]
	listenPort, _ := strconv.Atoi(serverFields[2])

	// Barcha clientlar ma'lumotlarini olish
	var totalBytesReceived, totalBytesSent int64
	var lastHandshake time.Time
	activeClients := 0

	// Birinchi qatorni o'tkazib yuborish (interface ma'lumotlari)
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 7 {
			// Handshake vaqtini o'zgartirish
			handshakeUnix, _ := strconv.ParseInt(fields[4], 10, 64)
			if handshakeUnix > 0 {
				clientHandshake := time.Unix(handshakeUnix, 0)
				// Oxirgi handshake vaqtini yangilash
				if clientHandshake.After(lastHandshake) {
					lastHandshake = clientHandshake
				}

				// Aktiv clientlarni hisoblash (oxirgi 3 minut ichida handshake bo'lgan)
				if time.Since(clientHandshake).Minutes() < 3 {
					activeClients++
				}
			}

			// Traffic ma'lumotlarini qo'shish
			bytesReceived, _ := strconv.ParseInt(fields[5], 10, 64)
			bytesSent, _ := strconv.ParseInt(fields[6], 10, 64)
			totalBytesReceived += bytesReceived
			totalBytesSent += bytesSent
		}
	}

	// Uptime ni olish
	uptime := "Ma'lumot yo'q"
	uptimeCmd := exec.Command("uptime", "-p")
	uptimeOutput, err := uptimeCmd.Output()
	if err == nil {
		uptime = strings.TrimSpace(string(uptimeOutput))
	}

	// Server holati obyektini yaratish
	status := &ServerStatus{
		InterfaceName:               interfaceName,
		ListenPort:                  listenPort,
		PublicKey:                   publicKey,
		PrivateKey:                  "", // Xavfsizlik uchun private key ni qaytarmaymiz
		ActiveClients:               activeClients,
		TotalClients:                len(lines) - 1, // Birinchi qator server ma'lumotlari
		TotalBytesReceived:          totalBytesReceived,
		TotalBytesSent:              totalBytesSent,
		TotalTraffic:                totalBytesReceived + totalBytesSent,
		LastHandshake:               lastHandshake,
		Uptime:                      uptime,
		TotalBytesReceivedFormatted: formatBytes(totalBytesReceived),
		TotalBytesSentFormatted:     formatBytes(totalBytesSent),
		TotalTrafficFormatted:       formatBytes(totalBytesReceived + totalBytesSent),
	}

	return status, nil
}
