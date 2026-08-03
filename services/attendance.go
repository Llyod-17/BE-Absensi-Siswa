package services

import (
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/requests"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
)

// DetermineAttendanceStatus menentukan status absensi berdasarkan kategori token
// dan batas keterlambatan (LateAfter).
// - Jika waktu sekarang melewati late_after -> "telat"
// - Token berada di kategori hadir & belum melewati late_after -> "hadir"
// - Token kategori telat -> "telat"
func DetermineAttendanceStatus(token *models.AttedanceTokens) string {
	if token.LateAfter != nil {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		if time.Now().In(loc).After(*token.LateAfter) {
			return "telat"
		}
	}
	return token.Category
}




func GetAttendanceTrend7Days() ([]requests.TrendResponse, error) {

	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)

	var result []requests.TrendResponse

	for i := 6; i >= 0; i-- {

		day := now.AddDate(0, 0, -i)

		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
		end := start.Add(24 * time.Hour)

		var total int64

		// 🔥 PAKAI POLA YANG SUDAH TERBUKTI (seperti dashboard kamu)
		err := database.DB.Model(&models.AttedanceLogs{}).
			Where("clock_in_time >= ? AND clock_in_time < ?", start, end).
			Count(&total).Error

		if err != nil {
			return nil, err
		}

		result = append(result, requests.TrendResponse{
			Date:  start.Format("2006-01-02"),
			Total: int(total),
		})
	}

	return result, nil
}
