package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

//struct untuk config database
type Config struct {
	DBName     string
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	Port       string
	WAHAURL    string
	WAHAAPIKey string
	WAHASession string
	WAHASwaggerUsername string
	WAHASwaggerPassword string
	WASendCron string
}

//variable dari struct
var AppConfig *Config


func LoadEnv() {
	// cek env kalo ga ada kasih error
	if err := godotenv.Load(); err != nil {
		log.Println("Error Not Found file .env !⚠️")
	}

	//instalasi untuk config
	AppConfig = &Config{
		Port:       os.Getenv("PORT"),
		DBName:     os.Getenv("DB_NAME"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBUser:     os.Getenv("DB_USER"),
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		WAHAURL:    os.Getenv("WAHA_URL"),
		WAHAAPIKey: os.Getenv("WAHA_API_KEY"),
		WAHASession: os.Getenv("WAHA_SESSION"),
		WAHASwaggerUsername: os.Getenv("WHATSAPP_SWAGGER_USERNAME"),
		WAHASwaggerPassword: os.Getenv("WHATSAPP_SWAGGER_PASSWORD"),
		WASendCron: os.Getenv("WA_SEND_CRON"),
	}
}