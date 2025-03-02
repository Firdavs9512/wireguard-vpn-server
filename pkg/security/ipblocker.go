package security

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// IPBlocker - IP manzillarni bloklash uchun struktura
type IPBlocker struct {
	failedAttempts map[string]int       // IP -> urinishlar soni
	blockedIPs     map[string]time.Time // IP -> bloklash vaqti
	blockDuration  time.Duration        // Bloklash muddati
	maxAttempts    int                  // Maksimal urinishlar soni
	mu             sync.RWMutex         // Thread-safe qilish uchun mutex
	logFile        *os.File             // Log fayli
}

// NewIPBlocker - Yangi IPBlocker yaratish
func NewIPBlocker(blockDuration time.Duration, maxAttempts int, logFilePath string) (*IPBlocker, error) {
	// Log faylini ochish
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("log faylini ochishda xatolik: %v", err)
	}

	// IPBlocker yaratish
	blocker := &IPBlocker{
		failedAttempts: make(map[string]int),
		blockedIPs:     make(map[string]time.Time),
		blockDuration:  blockDuration,
		maxAttempts:    maxAttempts,
		logFile:        logFile,
	}

	// Eskirgan bloklarni tozalash uchun goroutine ishga tushirish
	go blocker.cleanupExpiredBlocks()

	return blocker, nil
}

// IsBlocked - IP manzil bloklangan yoki yo'qligini tekshirish
func (b *IPBlocker) IsBlocked(ip string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// IP manzil bloklangan vaqtini olish
	blockTime, exists := b.blockedIPs[ip]
	if !exists {
		return false
	}

	// Bloklash muddati tugagan bo'lsa, blokni olib tashlash
	if time.Since(blockTime) > b.blockDuration {
		delete(b.blockedIPs, ip)
		delete(b.failedAttempts, ip)
		return false
	}

	return true
}

// RecordFailedAttempt - Muvaffaqiyatsiz urinishni qayd qilish
func (b *IPBlocker) RecordFailedAttempt(ip string, userAgent string, requestPath string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// IP manzil uchun urinishlar sonini oshirish
	b.failedAttempts[ip]++

	// Log faylga yozish
	logEntry := fmt.Sprintf("[%s] Xato token urinishi: IP=%s, User-Agent=%s, Path=%s, Urinishlar=%d/%d\n",
		time.Now().Format(time.RFC3339), ip, userAgent, requestPath, b.failedAttempts[ip], b.maxAttempts)
	if _, err := b.logFile.WriteString(logEntry); err != nil {
		log.Printf("Log faylga yozishda xatolik: %v", err)
	}

	// Agar urinishlar soni maksimal qiymatdan oshsa, IP manzilni bloklash
	if b.failedAttempts[ip] >= b.maxAttempts {
		b.blockedIPs[ip] = time.Now()

		// Bloklash haqida log yozish
		blockLogEntry := fmt.Sprintf("[%s] IP bloklandi: %s, Bloklash muddati: %s\n",
			time.Now().Format(time.RFC3339), ip, b.blockDuration)
		if _, err := b.logFile.WriteString(blockLogEntry); err != nil {
			log.Printf("Log faylga yozishda xatolik: %v", err)
		}
	}
}

// ResetFailedAttempts - Muvaffaqiyatsiz urinishlar sonini nolga tushirish
func (b *IPBlocker) ResetFailedAttempts(ip string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.failedAttempts, ip)
}

// GetRemainingBlockTime - IP manzil uchun qolgan bloklash vaqtini olish
func (b *IPBlocker) GetRemainingBlockTime(ip string) time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()

	blockTime, exists := b.blockedIPs[ip]
	if !exists {
		return 0
	}

	elapsed := time.Since(blockTime)
	if elapsed > b.blockDuration {
		return 0
	}

	return b.blockDuration - elapsed
}

// cleanupExpiredBlocks - Eskirgan bloklarni tozalash
func (b *IPBlocker) cleanupExpiredBlocks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		now := time.Now()

		// Eskirgan bloklarni tozalash
		for ip, blockTime := range b.blockedIPs {
			if now.Sub(blockTime) > b.blockDuration {
				delete(b.blockedIPs, ip)
				delete(b.failedAttempts, ip)

				// Log yozish
				logEntry := fmt.Sprintf("[%s] IP bloki olib tashlandi: %s\n",
					now.Format(time.RFC3339), ip)
				if _, err := b.logFile.WriteString(logEntry); err != nil {
					log.Printf("Log faylga yozishda xatolik: %v", err)
				}
			}
		}

		b.mu.Unlock()
	}
}

// Close - IPBlocker resurslarini yopish
func (b *IPBlocker) Close() error {
	return b.logFile.Close()
}
