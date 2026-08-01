package repo

import (
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
	"gorm.io/gorm"
)

type UnattendedStudent struct {
	ID          int64
	Nisn        string
	FullName    string
	ClassGroup  string
	ParentPhone string
}

type AbsentStudentWithStatus struct {
	ID          int64
	Nisn        string
	FullName    string
	ClassGroup  string
	ParentPhone string
	Status      string
}

// GetUnattendedStudents — ambil siswa yg belum absen sama sekali hari ini
func GetUnattendedStudents(db *gorm.DB) ([]UnattendedStudent, error) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	var students []UnattendedStudent

	err := db.
		Table("users u").
		Select("u.id, u.nisn, u.full_name, u.class_group, u.parent_phone").
		Joins(`LEFT JOIN (
			SELECT user_id, id
			FROM attedance_logs
			WHERE clock_in_time >= ? AND clock_in_time < ?
		) l ON l.user_id = u.id`, start, end).
		Where("u.role = ?", "siswa").
		Where("u.status = ?", "AKTIF").
		Where("l.id IS NULL").
		Order("u.class_group ASC, u.full_name ASC").
		Scan(&students).Error

	return students, err
}

// GetStudentsByStatusToday — ambil siswa yg punya status tertentu hari ini
func GetStudentsByStatusToday(db *gorm.DB, statuses []string) ([]AbsentStudentWithStatus, error) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	var students []AbsentStudentWithStatus

	err := db.
		Table("users u").
		Select("u.id, u.nisn, u.full_name, u.class_group, u.parent_phone, l.status").
		Joins(`INNER JOIN (
			SELECT user_id, status
			FROM attedance_logs
			WHERE clock_in_time >= ? AND clock_in_time < ?
			AND status IN ?
		) l ON l.user_id = u.id`, start, end, statuses).
		Where("u.role = ?", "siswa").
		Where("u.status = ?", "AKTIF").
		Order("u.class_group ASC, u.full_name ASC").
		Scan(&students).Error

	return students, err
}

// GetNotificationSettingsMap — ambil semua settings jadi map[key]value
func GetNotificationSettingsMap(db *gorm.DB) (map[string]string, error) {
	var settings []models.NotificationSettings
	if err := db.Find(&settings).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.SettingKey] = s.SettingValue
	}
	return result, nil
}

// IsNotificationSentOrPendingToday — cek udah pernah kirim (success) atau sedang antre (pending) notif buat user+status ini hari ini
func IsNotificationSentOrPendingToday(db *gorm.DB, userID int64, status string, today string) bool {
	var count int64
	db.Model(&models.NotificationLogs{}).
		Where("user_id = ? AND status = ? AND sent_date = ? AND (response_status LIKE ? OR response_status = ?)",
			userID, status, today, "success%", "pending").
		Count(&count)
	return count > 0
}

// TodayDateString — return tanggal hari ini format YYYY-MM-DD (WIB)
func TodayDateString() string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return time.Now().In(loc).Format("2006-01-02")
}
