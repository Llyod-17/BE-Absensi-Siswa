package services

import (
	"fmt"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/responses"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

		// Skip header
		if i == 0 {
			continue
		}

		// Skip baris kosong
		if len(row) == 0 {
			continue
		}

		if len(row) < 3 {
			result.Failed++
			result.FailedUsers = append(result.FailedUsers, responses.FailedUser{
				Row:    i + 1,
				Reason: "Jumlah kolom kurang dari 3",
			})
			continue
		}

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
			result.FailedUsers = append(result.FailedUsers, responses.FailedUser{
				Row:    i + 1,
				NISN:   nisn,
				Name:   fullName,
				Reason: "Password kosong",
			})
			continue
		}

		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(password),
			bcrypt.DefaultCost,
		)

		if err != nil {
			result.Failed++
			result.FailedUsers = append(result.FailedUsers, responses.FailedUser{
				Row:    i + 1,
				NISN:   nisn,
				Name:   fullName,
				Reason: err.Error(),
			})
			continue
		}

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

		err = database.DB.
			Where("username = ? OR nisn = ?", user.Username, user.Nisn).
			First(&existing).Error

		if err == nil {
			result.Duplicates++
			result.SkippedUsers = append(result.SkippedUsers, user.Username)
			continue
		}

		if err != gorm.ErrRecordNotFound {
			result.Failed++
			result.FailedUsers = append(result.FailedUsers, responses.FailedUser{
				Row:    i + 1,
				NISN:   nisn,
				Name:   fullName,
				Reason: err.Error(),
			})
			continue
		}

		if err := database.DB.Create(&user).Error; err != nil {

			result.Failed++

			result.FailedUsers = append(result.FailedUsers, responses.FailedUser{
				Row:    i + 1,
				NISN:   nisn,
				Name:   fullName,
				Reason: err.Error(),
			})

			fmt.Printf(
				"[FAILED] Row=%d NISN=%s Name=%s Error=%v\n",
				i+1,
				nisn,
				fullName,
				err,
			)

			continue
		}

		result.Inserted++

		fmt.Printf(
			"[INSERTED] Row=%d NISN=%s Name=%s\n",
			i+1,
			nisn,
			fullName,
		)
	}

	return result, nil
}
