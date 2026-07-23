package handlers

import (
	"fmt"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/responses"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/mappers"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
	"github.com/gofiber/fiber/v2"
)

// GetAttendanceLogsAdmin godoc
// @Summary Ambil riwayat absensi (Admin/Guru)
// @Description Mengambil riwayat absensi siswa secara paginated dengan filter rentang tanggal, kelas, status, dan pencarian nama/NISN
// @Tags attendance
// @Produce json
// @Param page query int false "Halaman (default: 1)"
// @Param limit query int false "Batas item per halaman (default: 20)"
// @Param start_date query string false "Filter tanggal mulai (Format: YYYY-MM-DD)"
// @Param end_date query string false "Filter tanggal selesai (Format: YYYY-MM-DD)"
// @Param class_group query string false "Filter berdasarkan Kelas"
// @Param status query string false "Filter berdasarkan Status (hadir, telat, alfa, sakit)"
// @Param search query string false "Pencarian berdasarkan Nama atau NISN Siswa"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /attendance/logs [get]
func GetAttendanceLogsAdmin(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	classGroup := c.Query("class_group")
	status := c.Query("status")
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	query := database.DB.Model(&models.AttedanceLogs{}).
		Joins("JOIN users ON users.id = attedance_logs.user_id").
		Select("attedance_logs.*")

	if startDate != "" {
		query = query.Where("attedance_logs.clock_in_time >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		query = query.Where("attedance_logs.clock_in_time <= ?", endDate+" 23:59:59")
	}
	if classGroup != "" {
		query = query.Where("users.class_group = ?", classGroup)
	}
	if status != "" {
		query = query.Where("attedance_logs.status = ?", status)
	}
	if search != "" {
		query = query.Where("users.full_name LIKE ? OR users.nisn LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal menghitung total log absensi"})
	}

	var logs []models.AttedanceLogs
	if err := query.
		Preload("User").
		Order("clock_in_time DESC, id DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal mengambil riwayat absensi"})
	}

	var result []responses.LogsRes
	for _, l := range logs {
		result = append(result, mappers.ToLogsResponse(l))
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data": fiber.Map{
			"logs":       result,
			"totalPages": totalPages,
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

type TopAlfaStudent struct {
	Name       string `json:"name"`
	Nisn       string `json:"nisn"`
	AlfaCount  int    `json:"alfaCount"`
	TelatCount int    `json:"telatCount"`
	ClassGroup string `json:"class_group"`
}

func GetTopAlfaStudents(c *fiber.Ctx) error {
	var result []TopAlfaStudent

	err := database.DB.Table("attedance_logs").
		Select("users.full_name as name, users.nisn, SUM(CASE WHEN attedance_logs.status = 'alfa' THEN 1 ELSE 0 END) as alfa_count, SUM(CASE WHEN attedance_logs.status = 'telat' THEN 1 ELSE 0 END) as telat_count, users.class_group").
		Joins("JOIN users ON users.id = attedance_logs.user_id").
		Where("users.role = ?", "siswa").
		Group("users.id, users.full_name, users.nisn, users.class_group").
		Having("alfa_count > 0 OR telat_count > 0").
		Order("alfa_count DESC, telat_count DESC").
		Limit(10).
		Scan(&result).Error

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal mengambil data top alfa"})
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    result,
	})
}

type MonthlyRecapData struct {
	Month string `json:"month"`
	Hadir int    `json:"Hadir"`
	Sakit int    `json:"Sakit"`
	Alfa  int    `json:"Alfa"`
	Rate  int    `json:"rate"`
}

func GetMonthlyRecap(c *fiber.Ctx) error {
	year := c.Query("year")

	type DBRecap struct {
		MonthKey  string `gorm:"column:month_key"`
		MonthName string `gorm:"column:month_name"`
		Hadir     int    `gorm:"column:hadir"`
		Sakit     int    `gorm:"column:sakit"`
		Alfa      int    `gorm:"column:alfa"`
	}

	var dbRecaps []DBRecap

	query := database.DB.Table("attedance_logs").
		Select(`
			DATE_FORMAT(clock_in_time, '%Y-%m') as month_key,
			DATE_FORMAT(clock_in_time, '%b %y') as month_name,
			SUM(CASE WHEN status IN ('hadir', 'telat') THEN 1 ELSE 0 END) as hadir,
			SUM(CASE WHEN status = 'sakit' THEN 1 ELSE 0 END) as sakit,
			SUM(CASE WHEN status = 'alfa' THEN 1 ELSE 0 END) as alfa
		`)

	if year != "" {
		var startYear, endYear int
		_, err := fmt.Sscanf(year, "%d/%d", &startYear, &endYear)
		if err == nil {
			startDate := fmt.Sprintf("%d-07-01 00:00:00", startYear)
			endDate := fmt.Sprintf("%d-06-30 23:59:59", endYear)
			query = query.Where("clock_in_time >= ? AND clock_in_time <= ?", startDate, endDate)
		}
	}

	err := query.Group("month_key, month_name").
		Order("month_key ASC").
		Limit(12).
		Scan(&dbRecaps).Error

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal mengambil rekap bulanan"})
	}

	var result []MonthlyRecapData = []MonthlyRecapData{}
	for _, r := range dbRecaps {
		total := r.Hadir + r.Sakit + r.Alfa
		rate := 0
		if total > 0 {
			rate = int((float64(r.Hadir) / float64(total)) * 100)
		}
		result = append(result, MonthlyRecapData{
			Month: r.MonthName,
			Hadir: r.Hadir,
			Sakit: r.Sakit,
			Alfa:  r.Alfa,
			Rate:  rate,
		})
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    result,
	})
}

