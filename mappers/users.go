package mappers

import (
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/responses"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
)

func ToUserResponse(u models.Users) responses.UserRes {
	var status string

	if len(u.AttedanceLogs) > 0 {
		status = u.AttedanceLogs[0].Status
	}
	return responses.UserRes{
		ID:              u.ID,
		Nisn:            u.Nisn,
		FullName:        u.FullName,
		Username:        u.Username,
		Role:            u.Role,
		Status:          u.Status,
		ClassGroup:      u.ClassGroup,
		ParentPhone:     u.ParentPhone,
		StatusAttedance: status,
	}
}
