package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Configuration - asosiy konfiguratsiya strukturasi
type Configuration struct {
	Server    ServerConfig    `yaml:"server"`
	API       APIConfig       `yaml:"api"`
	Wireguard WireguardConfig `yaml:"wireguard"`
	Database  DatabaseConfig  `yaml:"database"`
}

// ServerConfig - server konfiguratsiyasi
type ServerConfig struct {
	IP        string `yaml:"ip"`
	Port      int    `yaml:"port"`
	Interface string `yaml:"interface"`
	Debug     bool   `yaml:"debug"`
}

// APIConfig - API konfiguratsiyasi
type APIConfig struct {
	Port  int    `yaml:"port"`
	Token string `yaml:"token"`
}

// WireguardConfig - Wireguard konfiguratsiyasi
type WireguardConfig struct {
	DNS                 string `yaml:"dns"`
	AllowedIPs          string `yaml:"allowed_ips"`
	PersistentKeepalive int    `yaml:"persistent_keepalive"`
	ServerPublicKeyPath string `yaml:"server_public_key_path"`
}

// DatabaseConfig - Database konfiguratsiyasi
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// Config - global konfiguratsiya o'zgaruvchisi
var Config Configuration

// LoadConfig - konfiguratsiya faylini yuklash
func LoadConfig(configPath string) error {
	// Konfiguratsiya faylini o'qish
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("konfiguratsiya faylini o'qishda xatolik: %v", err)
	}

	// YAML formatidan strukturaga o'tkazish
	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		return fmt.Errorf("YAML formatini o'qishda xatolik: %v", err)
	}

	return nil
}

// CreateDefaultConfig - standart konfiguratsiya faylini yaratish
func CreateDefaultConfig(configPath string) error {
	// Standart konfiguratsiya
	defaultConfig := Configuration{
		Server: ServerConfig{
			IP:        "192.168.1.1",
			Port:      51820,
			Interface: "wg0",
			Debug:     false,
		},
		API: APIConfig{
			Port:  8080,
			Token: "secure-token-change-me",
		},
		Wireguard: WireguardConfig{
			DNS:                 "1.1.1.1, 8.8.8.8",
			AllowedIPs:          "0.0.0.0/0, ::/0",
			PersistentKeepalive: 25,
			ServerPublicKeyPath: "/etc/wireguard/server_public.key",
		},
		Database: DatabaseConfig{
			Path: "./data/wireguard.db",
		},
	}

	// Konfiguratsiya strukturasini YAML formatiga o'tkazish
	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return fmt.Errorf("YAML formatiga o'tkazishda xatolik: %v", err)
	}

	// Konfiguratsiya fayli uchun papkani yaratish
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("konfiguratsiya papkasini yaratishda xatolik: %v", err)
	}

	// Konfiguratsiya faylini yozish
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("konfiguratsiya faylini yozishda xatolik: %v", err)
	}

	return nil
}

// Wireguard konfiguratsiya konstantalari
const (
	// Eski konstantalar (endi Config strukturasidan olinadi)
	DNS                 = "1.1.1.1, 8.8.8.8"
	AllowedIPs          = "0.0.0.0/0, ::/0"
	Endpoint            = "192.168.1.151:51820"
	PersistentKeepalive = 25
	InterfaceName       = "wg0"
	ServerPublicKeyPath = "/etc/wireguard/server_public.key"
	ServerIP            = "192.168.1.151"

	// Database konfiguratsiyasi
	DatabasePath = "./data/wireguard.db"
)
