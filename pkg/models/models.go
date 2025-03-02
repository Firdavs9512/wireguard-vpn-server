package models

import (
	"time"

	"gorm.io/gorm"
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

// ClientType - Client turi
type ClientType string

const (
	// ClientTypeNormal - Oddiy client
	ClientTypeNormal ClientType = "normal"
	// ClientTypeVIP - VIP client
	ClientTypeVIP ClientType = "vip"
)

// WireguardClient - Database uchun client modeli
type WireguardClient struct {
	gorm.Model
	PublicKey     string     `gorm:"uniqueIndex;not null" json:"public_key"`
	PrivateKey    string     `gorm:"not null" json:"private_key"`
	PresharedKey  string     `gorm:"not null" json:"preshared_key"`
	Address       string     `gorm:"uniqueIndex;not null" json:"address"`
	Endpoint      string     `json:"endpoint"`
	DNS           string     `json:"dns"`
	AllowedIPs    string     `json:"allowed_ips"`
	ConfigText    string     `json:"config_text"`
	LastConnected time.Time  `json:"last_connected"`
	Description   string     `json:"description"`
	Active        bool       `gorm:"default:true" json:"active"`
	Type          ClientType `gorm:"default:'normal'" json:"type"`
	LifeTime      int        `gorm:"default:0" json:"life_time"` // Soniyalarda, 0 = cheksiz
	ExpiresAt     *time.Time `json:"expires_at"`
}
