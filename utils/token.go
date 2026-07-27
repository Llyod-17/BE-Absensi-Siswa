package utils

import (
	"errors"
	"math/rand"
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func RandomString(lenght int) string {
	b := make([]byte, lenght)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func CreateToken(adminID int64, durationMinutes int, category string) (*models.AttedanceTokens, error) {
	var tokenCode string

	for {
		code := RandomString(6)

		var count int64
		database.DB.Model(&models.AttedanceTokens{}).
			Where("token_code = ?", code).
			Count(&count)

		if count == 0 {
			tokenCode = code
			break
		}
	}

	now := Now()
	expired := now.Add(time.Minute * time.Duration(durationMinutes))

	token := models.AttedanceTokens{
		TokenCode:  tokenCode,
		CreatedBy:  adminID,
		Category:   category,
		IsActive:   true,
		ValidUntil: expired,
	}

	if err := database.DB.Create(&token).Error; err != nil {
		return nil, err
	}

	if err := database.DB.Preload("User").First(&token, token.ID).Error; err != nil {
		return nil, err
	}

	return &token, nil
}

func VerifyTokenCode(input string) (token *models.AttedanceTokens, isExpired bool, err error) {
	var t models.AttedanceTokens

	e := database.DB.
		Where("token_code = ? AND is_active = ?", input, true).
		First(&t).Error
	if e != nil {
		return nil, false, errors.New("Token expired !")
	}

	// cek expired — tetap return token, tapi tandai isExpired = true
	if Now().After(t.ValidUntil) {
		database.DB.
			Model(&models.AttedanceTokens{}).
			Where("id = ?", t.ID).
			Update("is_active", false)

		return &t, true, nil
	}

	return &t, false, nil
}
