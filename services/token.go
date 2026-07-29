package services

import (
	"log"
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
)

// StartTokenCleaner — background loop yang berjalan tiap 1 menit.
// Melakukan dua tugas:
//  1. Nonaktifkan token yang sudah melewati valid_until (is_active = false).
//  2. Deteksi token expired yang belum diproses notifikasinya (notification_processed = false),
//     lalu trigger pengiriman notifikasi WA sesuai kategori token dan tandai sebagai processed.
//
// Pendekatan ini menggantikan time.AfterFunc di memori, sehingga tahan terhadap server restart.
func StartTokenCleaner() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			loc, _ := time.LoadLocation("Asia/Jakarta")
			now := time.Now().In(loc)
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

			// 1. Nonaktifkan token yang sudah expired
			database.DB.
				Model(&models.AttedanceTokens{}).
				Where("is_active = ? AND valid_until < ?", true, now).
				Update("is_active", false)

			// 2. Cari token expired yang belum diproses notifikasinya
			var unprocessedTokens []models.AttedanceTokens
			database.DB.
				Where("is_active = ? AND valid_until < ? AND notification_processed = ?", false, now, false).
				Find(&unprocessedTokens)

			for _, token := range unprocessedTokens {
				// Hanya trigger broadcast/auto-alfa jika token berlaku pada hari ini
				if !token.ValidUntil.Before(startOfDay) {
					log.Printf("[WA-CLEANER] Token %s (kategori: %s) expired & belum diproses — mendelegasikan broadcast notifikasi...",
						token.TokenCode, token.Category)

					// Jalankan di background goroutine agar tidak menyumbat loop utama cleaner
					go func(t models.AttedanceTokens) {
						AutoAlfaAndNotify(database.DB)
					}(token)
				} else {
					log.Printf("[WA-CLEANER] Token %s dari hari sebelumnya diabaikan (hanya ditandai processed).", token.TokenCode)
				}

				// Tandai token sebagai sudah diproses notifikasinya
				database.DB.
					Model(&models.AttedanceTokens{}).
					Where("id = ?", token.ID).
					Update("notification_processed", true)

				log.Printf("[WA-CLEANER] Token %s (kategori: %s) — ditandai processed.",
					token.TokenCode, token.Category)
			}
		}
	}()
}