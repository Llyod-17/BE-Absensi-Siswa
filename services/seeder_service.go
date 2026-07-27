package services

import (
	"fmt"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/responses"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

func ImportUsersFromExcel(path string) (*responses.ImportResult, error) {

	file, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}

	rows, err := file.GetRows("Sheet1")
	if err != nil {
		return nil, err
	}

	result := &responses.ImportResult{}

	for i, row := range rows {

		if i == 0 {
			continue
		}

		if len(row) < 3 {
			result.Failed++
			continue
		}

		// Akses kolom secara aman dengan nilai default
		nisn := row[0]
		fullName := row[1]
		username := row[2]
		password := ""
		if len(row) > 3 {
			password = row[3]
		}
		classGroup := ""
		if len(row) > 4 {
			classGroup = row[4]
		}
		role := "siswa"
		if len(row) > 5 && row[5] != "" {
			role = row[5]
		}
		parentPhone := ""
		if len(row) > 6 {
			parentPhone = row[6]
		}

		if password == "" {
			result.Failed++
			continue
		}

		passwordHash, _ := bcrypt.GenerateFromPassword(
			[]byte(password),
			bcrypt.DefaultCost,
		)

		user := models.Users{
			Nisn:        nisn,
			FullName:    fullName,
			Username:    username,
			Password:    string(passwordHash),
			Role:        role,
			ClassGroup:  classGroup,
			ParentPhone: parentPhone,
		}

		var existing models.Users

		err := database.DB.Select("username").
			Where("username = ?", user.Username).
			First(&existing).Error

		if err == nil {
			result.Duplicates++
			result.SkippedUsers = append(
				result.SkippedUsers,
				user.Username,
			)
			continue
		}

		if err := database.DB.Create(&user).Error; err != nil {
			result.Failed++
			continue
		}

		result.Inserted++
		fmt.Println("Inserted:", user.Username)
	}

	return result, nil
}

