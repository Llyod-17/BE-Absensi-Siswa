package services

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
	"github.com/robfig/cron/v3"
)

// InitDataCleanup — jadwalkan pembersihan data lama setiap hari jam 03:00 WIB.
// Menghapus:
//   - attedance_logs lebih lama dari CLEANUP_ATTENDANCE_DAYS (default 240 hari / 8 bulan)
//   - notification_logs lebih lama dari CLEANUP_NOTIFICATION_DAYS (default 90 hari / 3 bulan)
//   - admin_notifications lebih lama dari CLEANUP_ADMIN_NOTIF_DAYS (default 90 hari / 3 bulan)
//
// Semua nilai bisa diatur lewat .env. Fitur ini tidak menyentuh logika WA sama sekali.
func InitDataCleanup() {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Printf("[CLEANUP] Gagal memuat zona waktu Asia/Jakarta: %v", err)
		return
	}

	c := cron.New(cron.WithLocation(loc))

	// Pukul 03:00 WIB — setelah rekap harian (15:00) dan sebelum aktivitas sekolah
	_, err = c.AddFunc("0 3 * * *", runDataCleanup)
	if err != nil {
		log.Printf("[CLEANUP] Gagal menjadwalkan cleanup: %v", err)
		return
	}

	// OPTIMIZE TABLE tiap tanggal 1 bulan, jam 03:30 WIB — defragmentasi & reclaim ruang disk
	_, err = c.AddFunc("30 3 1 * *", optimizeLogTables)
	if err != nil {
		log.Printf("[CLEANUP] Gagal menjadwalkan optimize: %v", err)
		return
	}

	c.Start()
	log.Println("[CLEANUP] Scheduler data cleanup aktif (03:00 WIB).")
}

func optimizeLogTables() {
	tables := []string{"attedance_logs", "notification_logs", "admin_notifications"}
	for _, t := range tables {
		log.Printf("[CLEANUP] Mengoptimalkan tabel %s...", t)
		if err := database.DB.Exec("OPTIMIZE TABLE " + t).Error; err != nil {
			log.Printf("[CLEANUP] Gagal optimize %s: %v", t, err)
		}
	}
}

func runDataCleanup() {
	log.Println("[CLEANUP] Memulai pembersihan data lama...")

	attendanceDays := envInt("CLEANUP_ATTENDANCE_DAYS", 240)
	notifDays := envInt("CLEANUP_NOTIFICATION_DAYS", 90)
	adminNotifDays := envInt("CLEANUP_ADMIN_NOTIF_DAYS", 90)

	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)
	cutoffAttendance := now.AddDate(0, 0, -attendanceDays)
	cutoffNotif := now.AddDate(0, 0, -notifDays).Format("2006-01-02")
	cutoffAdmin := now.AddDate(0, 0, -adminNotifDays)

	// 1. attedance_logs — hapus log absensi yang lebih lama dari cutoff
	res := database.DB.
		Where("clock_in_time < ?", cutoffAttendance).
		Delete(&models.AttedanceLogs{})
	if res.Error != nil {
		log.Printf("[CLEANUP] Gagal hapus attedance_logs: %v", res.Error)
	} else {
		log.Printf("[CLEANUP] attedance_logs dihapus: %d row (> %d hari).", res.RowsAffected, attendanceDays)
	}

	// 2. notification_logs — hapus log notifikasi yang lebih lama dari cutoff
	res = database.DB.
		Where("sent_date < ?", cutoffNotif).
		Delete(&models.NotificationLogs{})
	if res.Error != nil {
		log.Printf("[CLEANUP] Gagal hapus notification_logs: %v", res.Error)
	} else {
		log.Printf("[CLEANUP] notification_logs dihapus: %d row (> %d hari).", res.RowsAffected, notifDays)
	}

	// 3. admin_notifications — hard delete (model punya soft delete, wajib Unscoped)
	res = database.DB.Unscoped().
		Where("created_at < ?", cutoffAdmin).
		Delete(&models.AdminNotifications{})
	if res.Error != nil {
		log.Printf("[CLEANUP] Gagal hapus admin_notifications: %v", res.Error)
	} else {
		log.Printf("[CLEANUP] admin_notifications dihapus: %d row (> %d hari).", res.RowsAffected, adminNotifDays)
	}

	log.Println("[CLEANUP] Pembersihan selesai.")
}

// envInt — baca integer dari environment variable dengan default jika kosong/rusak
func envInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
