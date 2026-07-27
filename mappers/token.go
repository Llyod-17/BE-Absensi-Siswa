package mappers

import (
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/responses"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
)

func ToTokenResponse(t *models.AttedanceTokens) responses.TokenRes {
	return responses.TokenRes{
		ID:         t.ID,
		TokenCode:  t.TokenCode,
		Category:   t.Category,
		IsActive:   t.IsActive,
		LateAfter:  t.LateAfter,
		ValidUntil: t.ValidUntil,
		CreatedAt:  t.CreatedAt,
	}
}

