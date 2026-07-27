package database

import (
	"fmt"
	"log"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// bikin type untuk DB
var DB *gorm.DB

func ConnectDB() error {
	//ambil config dari AppConfig
	cfg := config.AppConfig

	//lakukan koneksi ke database dengan config tersebut
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FJakarta",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	//sambungkan ke mysql
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("⚠️ ERROR CONNECT TO DATABASE !", err)
	}
	//database di simpan ke DB
	DB = db

	fmt.Println("👌 CONNECT TO DATABASE COMPLETED !")
	return nil
}

