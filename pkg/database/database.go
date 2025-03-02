package database

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"wireguard-vpn-client-creater/pkg/models"
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

// DeactivateExpiredClients - Muddati o'tgan clientlarni deaktivatsiya qilish
func DeactivateExpiredClients() error {
	// Muddati o'tgan clientlarni topish
	expiredClients, err := CheckExpiredClients()
	if err != nil {
		return err
	}

	// Har bir muddati o'tgan clientni deaktivatsiya qilish
	for _, client := range expiredClients {
		log.Printf("Deactivating expired client: %d - %s (expired at: %s)",
			client.ID, client.Description, client.ExpiresAt.Format(time.RFC3339))

		// Clientni deaktivatsiya qilish
		if err := DeactivateClient(client.ID); err != nil {
			log.Printf("Error deactivating client %d: %v", client.ID, err)
			continue
		}
	}

	return nil
}
