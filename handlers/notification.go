package handlers

import (
	"strings"
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/requests"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/services"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/utils"
	"github.com/gofiber/fiber/v2"
)

// GetNotificationSettings godoc
// @Summary Ambil semua notification settings
// @Description Mengambil semua konfigurasi notifikasi WA
// @Tags notification
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /notification/settings [get]
func GetNotificationSettings(c *fiber.Ctx) error {
	var settings []models.NotificationSettings
	if err := database.DB.Find(&settings).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal mengambil settings"})
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    settings,
	})
}

// UpdateNotificationSettings godoc
// @Summary Update notification settings
// @Description Update satu atau lebih konfigurasi notifikasi WA
// @Tags notification
// @Accept json
// @Produce json
// @Param request body requests.UpdateSettingsBulkReq true "Settings to update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /notification/settings [put]
func UpdateNotificationSettings(c *fiber.Ctx) error {
	var req requests.UpdateSettingsBulkReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "payload tidak valid"})
	}

	if len(req.Settings) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "settings tidak boleh kosong"})
	}

	for _, s := range req.Settings {
		if s.SettingKey == "" || s.SettingValue == "" {
			return c.Status(400).JSON(fiber.Map{"error": "setting_key dan setting_value wajib diisi"})
		}

		result := database.DB.
			Model(&models.NotificationSettings{}).
			Where("setting_key = ?", s.SettingKey).
			Update("setting_value", s.SettingValue)

		if result.RowsAffected == 0 {
			// Buat baru jika belum ada
			newSetting := models.NotificationSettings{
				SettingKey:   s.SettingKey,
				SettingValue: s.SettingValue,
			}
			database.DB.Create(&newSetting)
		}
	}

	return c.JSON(fiber.Map{
		"message": "settings berhasil diupdate",
	})
}

// TestSendWA godoc
// @Summary Test kirim WA
// @Description Test kirim pesan WhatsApp ke nomor tertentu
// @Tags notification
// @Accept json
// @Produce json
// @Param request body requests.TestWAReq true "Test WA request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /notification/test [post]
func TestSendWA(c *fiber.Ctx) error {
	var req requests.TestWAReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "payload tidak valid"})
	}

	if req.Phone == "" || req.Message == "" {
		return c.Status(400).JSON(fiber.Map{"error": "phone dan message wajib diisi"})
	}

	response, err := services.TestSendWhatsApp(req.Phone, req.Message)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":    err.Error(),
			"response": response,
		})
	}

	return c.JSON(fiber.Map{
		"message":  "pesan WA berhasil dikirim",
		"response": response,
	})
}

// GetNotificationLogs godoc
// @Summary Ambil log notifikasi
// @Description Mengambil riwayat pengiriman notifikasi WA
// @Tags notification
// @Produce json
// @Param date query string false "Filter tanggal (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /notification/logs [get]
func GetNotificationLogs(c *fiber.Ctx) error {
	dateFilter := c.Query("date")

	var logs []models.NotificationLogs
	query := database.DB.Preload("User").Order("sent_at DESC")

	if dateFilter != "" {
		query = query.Where("sent_date = ?", dateFilter)
	}

	if err := query.Limit(100).Find(&logs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal mengambil logs"})
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    logs,
	})
}

// UpdateStudentStatus godoc
// @Summary Set status absensi siswa (oleh guru/admin)
// @Description Guru atau admin mengubah status absensi siswa menjadi sakit, izin, atau alfa
// @Tags notification
// @Accept json
// @Produce json
// @Param request body requests.UpdateStudentStatusReq true "Update status"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /attendance/status [put]
func UpdateStudentStatus(c *fiber.Ctx) error {
	var req requests.UpdateStudentStatusReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "payload tidak valid"})
	}

	// Validasi status
	validStatuses := map[string]bool{"hadir": true, "telat": true, "sakit": true, "alfa": true, "belum_absen": true}
	if !validStatuses[req.Status] {
		return c.Status(400).JSON(fiber.Map{"error": "status harus salah satu dari: hadir, telat, sakit, alfa, belum_absen"})
	}

	// Cek apakah user ada
	var user models.Users
	if err := database.DB.First(&user, req.UserID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "siswa tidak ditemukan"})
	}

	if user.Role != "siswa" {
		return c.Status(400).JSON(fiber.Map{"error": "hanya siswa yang bisa diubah statusnya"})
	}

	now := utils.Now()
	today := now.Format("2006-01-02")

	// Jika status diubah menjadi belum_absen, hapus log absensi hari ini jika ada
	if req.Status == "belum_absen" {
		err := database.DB.
			Where("user_id = ? AND DATE(clock_in_time) = ?", req.UserID, today).
			Delete(&models.AttedanceLogs{}).Error
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "gagal menghapus log absensi"})
		}

		return c.JSON(fiber.Map{
			"message": "status siswa berhasil di-reset (belum absen)",
			"data": fiber.Map{
				"user_id":   req.UserID,
				"full_name": user.FullName,
				"status":    "belum_absen",
				"updated":   true,
			},
		})
	}

	// Cek apakah sudah ada log hari ini
	var existingLog models.AttedanceLogs
	err := database.DB.
		Where("user_id = ? AND DATE(clock_in_time) = ?", req.UserID, today).
		First(&existingLog).Error

	if err == nil {
		// Update status yang sudah ada
		existingLog.Status = req.Status
		if err := database.DB.Save(&existingLog).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "gagal update status"})
		}

		return c.JSON(fiber.Map{
			"message": "status siswa berhasil diupdate",
			"data": fiber.Map{
				"user_id":   req.UserID,
				"full_name": user.FullName,
				"status":    req.Status,
				"updated":   true,
			},
		})
	}

	// Buat log baru (tanpa token)
	log := models.AttedanceLogs{
		UserID:      req.UserID,
		TokenID:     nil,
		Status:      req.Status,
		ClockInTime: now,
	}

	if err := database.DB.Create(&log).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal membuat log absensi"})
	}

	return c.JSON(fiber.Map{
		"message": "status siswa berhasil di-set",
		"data": fiber.Map{
			"user_id":   req.UserID,
			"full_name": user.FullName,
			"status":    req.Status,
			"created":   true,
		},
	})
}

// GetWAStatus godoc
// @Summary Cek status koneksi WhatsApp
// @Description Mengambil status koneksi WhatsApp (connected/disconnected/waiting_pair)
// @Tags whatsapp
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /notification/wa/status [get]
// func GetWAStatus(c *fiber.Ctx) error {
// 	status := services.GetWAStatus()
// 	return c.JSON(fiber.Map{
// 		"message": "success",
// 		"data":    status,
// 	})
// }

// PairWA godoc
// @Summary Request pairing code WhatsApp
// @Description Generate pairing code baru untuk menghubungkan WhatsApp. Masukkan kode di WhatsApp > Perangkat Tertaut > Tautkan dengan nomor telepon.
// @Tags whatsapp
// @Accept json
// @Produce json
// @Param request body requests.PairWAReq true "Nomor WhatsApp"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /notification/wa/pair [post]
// func PairWA(c *fiber.Ctx) error {
// 	qr, err := services.StartQRPairing()
// 	if err != nil {
// 		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
// 	}
//
// 	return c.JSON(fiber.Map{
// 		"message": "QR Code berhasil digenerate",
// 		"qr":      qr,
// 		"status":  "pairing",
// 	})
// }

// LogoutWA godoc
// @Summary Logout WhatsApp
// @Description Disconnect dan hapus sesi WhatsApp. Setelah logout, perlu pair ulang.
// @Tags whatsapp
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /notification/wa/logout [post]
// func LogoutWA(c *fiber.Ctx) error {
// 	if err := services.LogoutWA(); err != nil {
// 		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
// 	}
//
// 	return c.JSON(fiber.Map{
// 		"message": "WhatsApp berhasil logout. Silakan pair ulang.",
// 	})
// }

// GetStudentsAttendanceToday godoc
// @Summary Ambil daftar semua siswa + status absensi hari ini
// @Description Menampilkan semua siswa beserta status absensi hari ini (hadir/telat/alfa/sakit/belum_absen).
//
//	Bisa filter per kelas dan per status.
//
// @Tags attendance
// @Produce json
// @Param class_group query string false "Filter berdasarkan kelas (contoh: XII-RPL-1)"
// @Param status query string false "Filter berdasarkan status (hadir/telat/alfa/sakit/belum_absen)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /attendance/students [get]
func GetStudentsAttendanceToday(c *fiber.Ctx) error {
	classFilter := c.Query("class_group")
	statusFilter := c.Query("status")
	angkatan := c.Query("angkatan")
	jurusan := c.Query("jurusan")

	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	// Query: ambil semua siswa + LEFT JOIN absensi hari ini
	type StudentRow struct {
		ID          int64   `json:"id"`
		Nisn        string  `json:"nisn"`
		FullName    string  `json:"full_name"`
		ClassGroup  string  `json:"class_group"`
		ParentPhone string  `json:"parent_phone"`
		Status      *string `json:"status"`        // nil = belum absen
		ClockInTime *string `json:"clock_in_time"` // nil = belum absen
	}

	var rows []StudentRow

	db := database.DB.
		Table("users u").
		Select(`
			u.id,
			u.nisn,
			u.full_name,
			u.class_group,
			u.parent_phone,
			l.status,
			DATE_FORMAT(l.clock_in_time, '%H:%i:%s') as clock_in_time
		`).
		Joins(`LEFT JOIN (
			SELECT user_id, status, clock_in_time
			FROM attedance_logs
			WHERE clock_in_time >= ? AND clock_in_time < ?
		) l ON l.user_id = u.id`, start, end).
		Where("u.role = ?", "siswa")

	// Filter per kelas
	if classFilter != "" {
		db = db.Where("u.class_group = ?", classFilter)
	}

	// Filter angkatan
	if angkatan != "" {
		if angkatan == "Kelas X" {
			db = db.Where("u.class_group LIKE ? OR u.class_group LIKE ?", "X-%", "X %")
		} else if angkatan == "Kelas XI" {
			db = db.Where("u.class_group LIKE ? OR u.class_group LIKE ?", "XI-%", "XI %")
		} else if angkatan == "Kelas XII" {
			db = db.Where("u.class_group LIKE ? OR u.class_group LIKE ?", "XII-%", "XII %")
		}
	}

	// Filter jurusan
	if jurusan != "" && jurusan != "Semua Jurusan" {
		db = db.Where("u.class_group LIKE ?", "%"+jurusan+"%")
	}

	db = db.Order("u.class_group ASC, u.full_name ASC")

	if err := db.Scan(&rows).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal mengambil data siswa"})
	}

	// Hitung summary dan format response
	var (
		totalSiswa int
		hadir      int
		telat      int
		alfa       int
		sakit      int
		belumAbsen int
	)

	type DeptStat struct {
		Total int `json:"total"`
		Hadir int `json:"hadir"`
	}
	deptStats := make(map[string]*DeptStat)

	type StudentResponse struct {
		ID          int64  `json:"id"`
		Nisn        string `json:"nisn"`
		FullName    string `json:"full_name"`
		ClassGroup  string `json:"class_group"`
		ParentPhone string `json:"parent_phone"`
		Status      string `json:"status"`
		ClockInTime string `json:"clock_in_time"`
	}

	var result []StudentResponse

	for _, r := range rows {
		status := "belum_absen"
		clockIn := "-"

		if r.Status != nil {
			status = *r.Status
		}
		if r.ClockInTime != nil {
			clockIn = *r.ClockInTime
		}

		// Filter status jika diminta
		if statusFilter != "" {
			if statusFilter == "belum" || statusFilter == "belum_absen" {
				if status != "belum_absen" {
					continue
				}
			} else if statusFilter == "sakit" {
				if status != "sakit" && status != "izin" {
					continue
				}
			} else if status != statusFilter {
				continue
			}
		}

		switch status {
		case "hadir":
			hadir++
		case "telat":
			telat++
		case "alfa":
			alfa++
		case "sakit":
			sakit++
		default:
			belumAbsen++
		}
		totalSiswa++

		// Extract jurusan dari class_group (misal: "X-RPL-1" -> "RPL", "RPL" -> "RPL")
		jurusan := r.ClassGroup
		parts := strings.Split(r.ClassGroup, "-")
		if len(parts) >= 2 {
			jurusan = parts[1]
		}
		if _, ok := deptStats[jurusan]; !ok {
			deptStats[jurusan] = &DeptStat{}
		}
		deptStats[jurusan].Total++
		if status == "hadir" || status == "telat" {
			deptStats[jurusan].Hadir++
		}

		result = append(result, StudentResponse{
			ID:          r.ID,
			Nisn:        r.Nisn,
			FullName:    r.FullName,
			ClassGroup:  r.ClassGroup,
			ParentPhone: r.ParentPhone,
			Status:      status,
			ClockInTime: clockIn,
		})
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"date":    start.Format("2006-01-02"),
		"summary": fiber.Map{
			"total":       totalSiswa,
			"hadir":       hadir,
			"telat":       telat,
			"alfa":        alfa,
			"sakit":       sakit,
			"belum_absen": belumAbsen,
			"departments": deptStats,
		},
		"data": result,
	})
}
