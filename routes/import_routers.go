package routes

import (
	"github.com/KicauOrgspark/BE-Absensi-Siswa/handlers"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/middleware"
	"github.com/gofiber/fiber/v2"
)

func SetupimportRoutes(api fiber.Router) {
	importGroup := api.Group("/import")
	importGroup.Post("/users", middleware.AdminOnly, handlers.ImportUsersExcel)
}
