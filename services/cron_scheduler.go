package services

import (
	"fmt"
	"log"
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/config"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/repo"
	"github.com/robfig/cron/v3"
)

func InitCronScheduler() {
	// Buat cron scheduler dengan zona waktu Asia/Jakarta
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Printf("[CRON] Gagal memuat zona waktu Asia/Jakarta: %v", err)
		return
	}

	c := cron.New(cron.WithLocation(loc))

	// Jadwalkan notifikasi WA harian (default 08:30 WIB, bisa diubah via WA_SEND_CRON)
	waCron := "30 8 * * *"
	if config.AppConfig.WASendCron != "" {
		waCron = config.AppConfig.WASendCron
	}
	_, err = c.AddFunc(waCron, runDailyWANotification)
	if err != nil {
		log.Printf("[CRON] Gagal menjadwalkan Notifikasi WA Harian: %v", err)
		return
	}
	log.Printf("[CRON] Notifikasi WA Harian dijadwalkan pada: %s (WIB)", waCron)

	// Jadwalkan Rekap Harian jam 15:00 setiap hari
	_, err = c.AddFunc("0 15 * * *", runDailyRecap)
	if err != nil {
		log.Printf("[CRON] Gagal menjadwalkan Rekap Harian: %v", err)
		return
	}

	c.Start()
	log.Println("[CRON] Scheduler berhasil dijalankan.")
}

func runDailyWANotification() {
	waCron := "30 8 * * *"
	if config.AppConfig.WASendCron != "" {
		waCron = config.AppConfig.WASendCron
	}
	log.Printf("[CRON] Memulai pengiriman notifikasi WA harian (jadwal: %s WIB)...", waCron)
	SendDailyWANotifications(database.DB)
}

func runDailyRecap() {
	log.Println("[CRON] Memulai Rekap Harian...")

	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	var hadir, alpa, telat int64

	// Hitung Hadir
	database.DB.Model(&models.AttedanceLogs{}).
		Where("clock_in_time >= ? AND clock_in_time < ? AND status = ?", start, end, "hadir").
		Count(&hadir)

	// Hitung Telat
	database.DB.Model(&models.AttedanceLogs{}).
		Where("clock_in_time >= ? AND clock_in_time < ? AND status = ?", start, end, "telat").
		Count(&telat)

	// Hitung Alpa
	database.DB.Model(&models.AttedanceLogs{}).
		Where("clock_in_time >= ? AND clock_in_time < ? AND status = ?", start, end, "alfa").
		Count(&alpa)

	pesan := fmt.Sprintf("Rekap absensi siswa hari ini (%s):\n- Hadir: %d\n- Telat: %d\n- Alpa: %d",
		now.Format("02-01-2006"), hadir, telat, alpa)

	err := repo.InsertNotification("Rekap Harian Absensi", pesan, "Rekap")
	if err != nil {
		log.Printf("[CRON] Gagal menyimpan notifikasi rekap: %v", err)
	} else {
		log.Println("[CRON] Rekap Harian berhasil disimpan.")
	}
}
