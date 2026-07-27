package handlers

import (
	"github.com/KicauOrgspark/BE-Absensi-Siswa/repo"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/services"
	"github.com/gofiber/fiber/v2"
)

// Dashboard godoc
// @Summary Ambil data dashboard
// @Description Mengambil data ringkasan dashboard (total user, absensi, dll)
// @Tags dashboard
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /dashboard [get]
func Dashboard(c *fiber.Ctx) error {

	data, err := repo.GetDashboardData()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get dashboard data"})
	}
	return c.Status(200).JSON(fiber.Map{"Message": "success get dashboard data", "data": data})
}

func GetTrendAttendance(c *fiber.Ctx) error {

	data, err := services.GetAttendanceTrend7Days()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    data,
	})
}

