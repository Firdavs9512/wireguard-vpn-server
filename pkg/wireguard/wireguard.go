package wireguard

import (
	"fmt"
	"math/rand"
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

// GenerateClientIP - Yangi IP manzil yaratish
func GenerateClientIP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("10.7.0.%d/32", rand.Intn(250)+2) // 10.0.0.2 - 10.0.0.251 oralig'ida
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
