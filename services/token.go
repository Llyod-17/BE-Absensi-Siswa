package services

import (
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
)

// StartTokenCleaner — background loop yang berjalan tiap 1 menit.
// Nonaktifkan token yang sudah melewati valid_until (is_active = false).
func StartTokenCleaner() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()

			database.DB.
				Model(&models.AttedanceTokens{}).
				Where("is_active = ? AND valid_until < ?", true, now).
				Update("is_active", false)
		}
	}()
}