package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/requests"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/responses"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/mappers"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/services"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/utils"
	"github.com/gofiber/fiber/v2"
)

// CreateToken godoc
// @Summary Buat token absensi (custom)
// @Description Guru/Admin membuat token dengan durasi dan toleransi keterlambatan
// @Tags token
// @Accept json
// @Produce json
// @Param request body requests.TokenReq true "Request token"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /token/create [post]
func CreateToken(c *fiber.Ctx) error {
	adminID, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req requests.TokenReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Muatan tidak valid"})
	}

	token, err := utils.CreateToken(adminID, req.Duration, req.Category)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"errors": err.Error()})
	}

	// Notifikasi WA dijadwalkan secara otomatis oleh StartTokenCleaner
	// setelah token expired (tidak perlu time.AfterFunc di sini).

	return c.Status(201).JSON(fiber.Map{
		"message": "Token berhasil dibuat",
		"data":    mappers.ToTokenResponse(token),
	})
}

// CreateTokenHadir godoc
// @Summary Buat token absensi HADIR (Fase 1)
// @Description Guru/Admin membuat token khusus kehadiran tepat waktu (default 30 menit)
// @Tags token
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /token/create/hadir [post]
func CreateTokenHadir(c *fiber.Ctx) error {
	adminID, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	token, err := utils.CreateToken(adminID, 30, "hadir")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"errors": err.Error()})
	}

	// Notifikasi WA dijadwalkan secara otomatis oleh StartTokenCleaner

	return c.Status(201).JSON(fiber.Map{
		"message": "Token Hadir berhasil dibuat",
		"data":    mappers.ToTokenResponse(token),
	})
}

// CreateTokenTelat godoc
// @Summary Buat token absensi TELAT (Fase 2)
// @Description Guru/Admin membuat token khusus keterlambatan (default 60 menit)
// @Tags token
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /token/create/telat [post]
func CreateTokenTelat(c *fiber.Ctx) error {
	adminID, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	token, err := utils.CreateToken(adminID, 60, "telat")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"errors": err.Error()})
	}

	// Notifikasi WA dijadwalkan secara otomatis oleh StartTokenCleaner

	return c.Status(201).JSON(fiber.Map{
		"message": "Token Telat berhasil dibuat",
		"data":    mappers.ToTokenResponse(token),
	})
}

// SubmitToken godoc
// @Summary Submit token absensi
// @Description Siswa memasukkan token untuk melakukan absensi
// @Tags token
// @Accept json
// @Produce json
// @Param request body requests.SubmitToken true "Submit token"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /token/absen [post]
func SubmitToken(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req requests.SubmitToken
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Muatan tidak valid"})
	}

	// Validasi geofence (jarak dari sekolah)
	if req.Latitude == 0 || req.Longitude == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Lokasi GPS diperlukan untuk melakukan absensi"})
	}

	if !utils.IsInsideSchool(req.Latitude, req.Longitude) {
		return c.Status(400).JSON(fiber.Map{"error": "Kamu berada di luar area sekolah"})
	}

	// Verifikasi token
	token, isExpired, err := utils.VerifyTokenCode(req.TokenCode)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// Tolak jika token sudah kedaluwarsa (harus menunggu QR fase berikutnya jika masih ada)
	if isExpired {
		return c.Status(400).JSON(fiber.Map{"error": "Token QR sudah kedaluwarsa, silakan minta QR yang baru"})
	}

	// ── Gunakan Transaction untuk menghindari Race Condition ──
	tx := database.DB.Begin()
	if tx.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal memulai transaksi"})
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// SELECT FOR UPDATE: kunci baris agar request lain menunggu
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	var existingLog models.AttedanceLogs
	err = tx.Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ? AND clock_in_time >= ? AND clock_in_time < ?", userID, start, end).
		First(&existingLog).Error

	if err == nil {
		// Log absensi sudah ada hari ini
		tx.Rollback()
		if existingLog.Status == "hadir" || existingLog.Status == "sakit" || existingLog.Status == "alfa" {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Kamu tidak bisa absen karena status hari ini adalah %s", existingLog.Status),
			})
		}

		if existingLog.Status == "telat" {
			if existingLog.TokenID != nil {
				return c.Status(400).JSON(fiber.Map{
					"error": "Kamu sudah melakukan absensi (telat)",
				})
			}

			if token.Category != "telat" {
				return c.Status(400).JSON(fiber.Map{
					"error": "Token tidak valid untuk status keterlambatan Anda",
				})
			}

			ip := c.IP()
			tokenID := token.ID
			existingLog.TokenID = &tokenID
			existingLog.CapturedIp = &ip
			existingLog.ClockInTime = utils.Now()

			// Gunakan tx baru karena yang lama sudah di-Rollback
			if err := database.DB.Save(&existingLog).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Gagal memperbarui status absensi"})
			}

			return c.Status(200).JSON(fiber.Map{
				"message": "Success To Absen",
				"status":  "telat",
			})
		}
	}

	// Jika belum ada log absensi sama sekali hari ini
	status := services.DetermineAttendanceStatus(token)
	tokenID := token.ID
	ip := c.IP()

	log := models.AttedanceLogs{
		UserID:      userID,
		TokenID:     &tokenID,
		Status:      status,
		CapturedIp:  &ip,
		ClockInTime: utils.Now(),
	}

	if err := tx.Create(&log).Error; err != nil {
		tx.Rollback()
		// Jika error adalah duplicate entry (dari UNIQUE INDEX), anggap sukses
		// karena artinya request lain sudah berhasil duluan
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062") {
			return c.Status(200).JSON(fiber.Map{
				"message": "Success To Absen",
				"status":  status,
			})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan absensi"})
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "Success To Absen",
		"status":  status,
	})
}

// ini untuk membuat api get qr code by id token
func GetTokenQRImage(c *fiber.Ctx) error {

	id := c.Params("id")

	var token models.AttedanceTokens

	if err := database.DB.
		First(&token, id).Error; err != nil {

		return c.Status(404).JSON(fiber.Map{
			"error": "token not found",
		})
	}

	png, err := utils.GenerateQRCode(
		token.TokenCode,
	)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed generate qr",
		})
	}

	c.Set("Content-Type", "image/png")

	return c.Send(png)
}

// ini untuk membuat api get qr code all token aktif
func GetActiveTokens(c *fiber.Ctx) error {

	var tokens []models.AttedanceTokens

	now := time.Now()

	err := database.DB.
		Where("is_active = ?", true).
		Where("valid_until > ?", now).
		Order("created_at desc").
		Find(&tokens).Error

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "gagal ambil token aktif",
		})
	}

	var result []fiber.Map

	for _, token := range tokens {

		result = append(result, fiber.Map{
			"id":         token.ID,
			"token_code": token.TokenCode,

			"qr_url": fmt.Sprintf(
				"/api/v1/token/%d/image",
				token.ID,
			),

			"expired_at": token.ValidUntil,
			"category":   token.Category,
			"is_active":  token.IsActive,
		})
	}

	return c.JSON(fiber.Map{
		"message": "success get qr code active!",
		"data":    result,
	})
}

// GetTokensPaginated godoc
// @Summary Ambil daftar semua token (paginasi)
// @Description Mengambil riwayat semua token absensi dengan paginasi
// @Tags token
// @Produce json
// @Param page query int false "Nomor halaman (default: 1)"
// @Param limit query int false "Jumlah per halaman (default: 20)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /token [get]
func GetTokensPaginated(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var total int64
	if err := database.DB.Model(&models.AttedanceTokens{}).Count(&total).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal menghitung total token"})
	}

	var tokens []models.AttedanceTokens
	if err := database.DB.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&tokens).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal mengambil data token"})
	}

	var result []responses.TokenRes
	for _, t := range tokens {
		result = append(result, mappers.ToTokenResponse(&t))
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data": fiber.Map{
			"tokens":     result,
			"totalPages": totalPages,
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

// UpdateToken godoc
// @Summary Update token absensi (quick update)
// @Description Memperbarui masa berlaku token (ValidUntil: 08:00, LateAfter: 07:30) untuk tanggal tertentu (default: besok)
// @Tags token
// @Accept json
// @Produce json
// @Param id path int true "Token ID"
// @Param request body requests.QuickUpdateToken false "Request update token"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /token/{id}/update [post]
func UpdateToken(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID token tidak valid"})
	}

	var req requests.QuickUpdateToken
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Muatan tidak valid"})
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	var targetDate time.Time

	if req.Date != "" {
		targetDate, err = time.ParseInLocation("2006-01-02", req.Date, loc)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Format tanggal tidak valid, gunakan YYYY-MM-DD"})
		}
	} else {
		// Default to tomorrow
		now := time.Now().In(loc)
		targetDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	}

	validUntil := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 21, 25, 0, 0, loc)
	lateAfter := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 21, 23, 0, 0, loc)

	var token models.AttedanceTokens
	if err := database.DB.First(&token, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "token tidak ditemukan"})
	}

	token.ValidUntil = validUntil
	token.LateAfter = &lateAfter
	token.IsActive = true               // Aktifkan kembali jika sempat dinonaktifkan
	token.NotificationProcessed = false // Reset agar diproses kembali

	if err := database.DB.Preload("User").Save(&token).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal memperbarui token"})
	}

	return c.JSON(fiber.Map{
		"message": "Token berhasil diperbarui",
		"data":    mappers.ToTokenResponse(&token),
	})
}
