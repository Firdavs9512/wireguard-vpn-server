package database

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"wireguard-vpn-client-creater/pkg/models"
	"wireguard-vpn-client-creater/pkg/wireguard"
)

var DB *gorm.DB

// InitDB - Databaseni ishga tushirish
func InitDB(dbPath string) (*gorm.DB, error) {
	// Database papkasini yaratish
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// Database bilan bog'lanish
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Modellarni migrate qilish
	err = db.AutoMigrate(&models.WireguardClient{})
	if err != nil {
		return nil, err
	}

	DB = db
	log.Println("Database initialized at", dbPath)
	return db, nil
}

// SaveClient - Yangi clientni databasega saqlash
func SaveClient(client *models.WireguardClient) error {
	return DB.Create(client).Error
}

// GetAllClients - Barcha clientlarni olish
func GetAllClients() ([]models.WireguardClient, error) {
	var clients []models.WireguardClient
	err := DB.Find(&clients).Error
	return clients, err
}

// GetClientByPublicKey - Public key bo'yicha clientni topish
func GetClientByPublicKey(publicKey string) (models.WireguardClient, error) {
	var client models.WireguardClient
	err := DB.Where("public_key = ?", publicKey).First(&client).Error
	return client, err
}

// GetClientByAddress - IP address bo'yicha clientni topish
func GetClientByAddress(address string) (models.WireguardClient, error) {
	var client models.WireguardClient
	err := DB.Where("address = ?", address).First(&client).Error
	return client, err
}

// UpdateClient - Clientni yangilash
func UpdateClient(client *models.WireguardClient) error {
	return DB.Save(client).Error
}

// DeleteClient - Clientni o'chirish
func DeleteClient(id uint) error {
	return DB.Delete(&models.WireguardClient{}, id).Error
}

// DeactivateClient - Clientni deaktivatsiya qilish
func DeactivateClient(id uint) error {
	return DB.Model(&models.WireguardClient{}).Where("id = ?", id).Update("active", false).Error
}

// GetClientByID - ID bo'yicha clientni topish
func GetClientByID(id uint) (models.WireguardClient, error) {
	var client models.WireguardClient
	err := DB.First(&client, id).Error
	return client, err
}

// CheckExpiredClients - Muddati o'tgan clientlarni tekshirish
func CheckExpiredClients() ([]models.WireguardClient, error) {
	var expiredClients []models.WireguardClient
	// Faqat muddati o'tgan va muddati chekli bo'lgan clientlarni topish
	// ExpiresAt NULL bo'lmagan va hozirgi vaqtdan kichik bo'lgan
	err := DB.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Find(&expiredClients).Error
	return expiredClients, err
}

// DeleteExpiredClients - Muddati o'tgan clientlarni o'chirish
func DeleteExpiredClients() error {
	// Muddati o'tgan clientlarni topish
	expiredClients, err := CheckExpiredClients()
	if err != nil {
		return err
	}

	// Har bir muddati o'tgan clientni o'chirish
	for _, client := range expiredClients {
		log.Printf("O'chirish: muddati o'tgan client: %d - %s (muddati tugagan: %s)",
			client.ID, client.Description, client.ExpiresAt.Format(time.RFC3339))

		// Wireguard konfiguratsiyasidan peer ni o'chirish
		if err := wireguard.RemovePeerFromServer(client.PublicKey); err != nil {
			log.Printf("Xatolik: Wireguard konfiguratsiyasidan client %d ni o'chirishda: %v", client.ID, err)
			continue
		}

		// Databasedan o'chirish
		if err := DB.Delete(&client).Error; err != nil {
			log.Printf("Xatolik: Databasedan client %d ni o'chirishda: %v", client.ID, err)
			continue
		}

		log.Printf("Client %d muvaffaqiyatli o'chirildi", client.ID)
	}

	return nil
}

// GetUsedIPAddresses - Ishlatilayotgan IP manzillarni olish
func GetUsedIPAddresses(subnetPrefix string) ([]string, error) {
	var clients []models.WireguardClient
	var addresses []string

	// Berilgan subnet prefixga mos IP manzillarni olish
	err := DB.Where("address LIKE ?", subnetPrefix+"%").Find(&clients).Error
	if err != nil {
		return nil, err
	}

	// IP manzillarni saqlash
	for _, client := range clients {
		// CIDR notatsiyasini olib tashlash (/32)
		ipWithoutCIDR := strings.Split(client.Address, "/")[0]
		addresses = append(addresses, ipWithoutCIDR)
	}

	return addresses, nil
}
