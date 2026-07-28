package routes

import (
	"github.com/KicauOrgspark/BE-Absensi-Siswa/handlers"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/middleware"
	"github.com/gofiber/fiber/v2"
)

func SetupNotificationRoutes(api fiber.Router) {
	notif := api.Group("/notification")

	// Settings CRUD — hanya admin/superadmin
	notif.Get("/settings", middleware.AdminOnly, handlers.GetNotificationSettings)
	notif.Put("/settings", middleware.AdminOnly, handlers.UpdateNotificationSettings)

	// Test kirim WA — hanya admin/superadmin
	notif.Post("/test", middleware.AdminOnly, handlers.TestSendWA)

	// Log notifikasi — hanya admin/superadmin
	notif.Get("/logs", middleware.AdminOnly, handlers.GetNotificationLogs)

	// Trigger manual — hanya admin/superadmin
	notif.Post("/trigger", middleware.AdminOnly, handlers.TriggerNotificationNow)

	// WhatsApp management — hanya admin/superadmin
	// wa := notif.Group("/wa")
	// wa.Get("/status", middleware.AdminOnly, handlers.GetWAStatus)
	// wa.Post("/pair", middleware.AdminOnly, handlers.PairWA)
	// wa.Post("/logout", middleware.AdminOnly, handlers.LogoutWA)

	// Set status absensi siswa — guru/admin/superadmin
	attendance := api.Group("/attendance")
	attendance.Get("/students", middleware.AdminGuru, handlers.GetStudentsAttendanceToday)
	attendance.Put("/status", middleware.AdminGuru, handlers.UpdateStudentStatus)
	attendance.Get("/logs", middleware.AdminGuru, handlers.GetAttendanceLogsAdmin)
	attendance.Get("/top-alfa", middleware.AdminGuru, handlers.GetTopAlfaStudents)
	attendance.Get("/monthly-recap", middleware.AdminGuru, handlers.GetMonthlyRecap)
}
