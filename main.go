// @title My API
// @version 1.0
// @description Ini adalah dokumentasi API gue
// @host www.reihan.biz.id
// @BasePath /api/v1

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/config"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/database/seeders"
	_ "github.com/KicauOrgspark/BE-Absensi-Siswa/docs" // WAJIB sesuai module
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/routes"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

func main() {

	//load env
	config.LoadEnv()

	//connect to database
	database.ConnectDB()

	// Migrasi: hapus unique index lama (user_id, sent_date) yang menghalangi
	// notifikasi status berbeda (alfa/sakit/telat) di hari yang sama.
	// AutoMigrate akan membuat index baru (user_id, status, sent_date).
	_ = database.DB.Migrator().DropIndex(&models.NotificationLogs{}, "unique_daily_notif")

	database.DB.AutoMigrate(
		&models.Users{},
		&models.AttedanceTokens{},
		&models.AttedanceLogs{},
		&models.NotificationSettings{},
		&models.NotificationLogs{},
		&models.AdminNotifications{},
	)

	// Inisialisasi WAHA (validate config)
	if err := services.InitWA(); err != nil {
		log.Printf("[WAHA] Gagal inisialisasi: %v", err)
	} else if err := services.ConnectWA(); err != nil {
		log.Printf("[WAHA] Gagal connect: %v", err)
	} else {
		log.Println("[WAHA] Berhasil terhubung ke WAHA API.")
	}

	//to running seeders
	seeders.RunSeed()

	//start token cleaner service
	services.StartTokenCleaner()

	//start background notification sender
	services.StartNotificationSender(database.DB)

	//start cron scheduler
	services.InitCronScheduler()

	//start data cleanup scheduler (03:00 WIB)
	services.InitDataCleanup()

	app := fiber.New()

	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	app.Use(cors.New(cors.Config{AllowOrigins: "https://smart-presence.smkpluspnb.sch.id,https://api.smart-presence.smkpluspnb.sch.id,http://localhost:3000,http://localhost:3001,http://localhost:3052,http://localhost:5173",
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true, //jika pake jwt
	}))

	routes.SetupRoutes(app)

	// Notifikasi WA harian dijadwalkan otomatis oleh cron scheduler
	// (default 08:30 WIB, bisa diubah via WA_SEND_CRON di .env).
	// Worker StartNotificationSender di atas mengirim antrean (pending)
	// di notification_logs setiap 15 detik.

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Jalankan server di goroutine terpisah
	go func() {
		log.Printf("Server berjalan di port %s", config.AppConfig.Port)
		if err := app.Listen(config.AppConfig.Port); err != nil {
			log.Fatalf("Gagal menjalankan server: %v", err)
		}
	}()

	// Tunggu sinyal shutdown
	sig := <-quit
	log.Printf("Sinyal [%s] diterima — memulai graceful shutdown...", sig)

	// Beri waktu 10 detik untuk menyelesaikan request yang masih berjalan
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Fatalf("Gagal melakukan graceful shutdown: %v", err)
	}

	// Cron scheduler sudah tidak digunakan

	log.Println("Server berhasil dimatikan dengan aman.")
	
}
