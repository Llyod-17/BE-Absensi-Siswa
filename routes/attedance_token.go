package routes

import (
	"fmt"
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/handlers"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func SetupRouteAttedanceToken(api fiber.Router) {
	token := api.Group("/token")

	token.Get("/", middleware.AdminOnly, handlers.GetTokensPaginated)

	token.Post("/create", middleware.AdminOnly, handlers.CreateToken)
	token.Post("/create/hadir", middleware.AdminOnly, handlers.CreateTokenHadir)
	token.Post("/create/telat", middleware.AdminOnly, handlers.CreateTokenTelat)

	// Rate limiting: maks 5 request per 1 menit per User ID untuk mencegah brute-force QR code
	absenLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			userID := c.Locals("user_id")
			if userID != nil {
				return fmt.Sprintf("absen_limit_%v", userID)
			}
			return c.IP() // fallback
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Terlalu banyak percobaan absen. Silakan tunggu 1 menit.",
			})
		},
	})

	token.Post("/absen", middleware.SiswaOnly, absenLimiter, handlers.SubmitToken)

	token.Get("/qr_code/active", middleware.AdminGuru, handlers.GetActiveTokens)
	token.Get("/:id/image", middleware.AdminGuru, handlers.GetTokenQRImage)
	token.Post("/:id/update", middleware.AdminOnly, handlers.UpdateToken)
}
