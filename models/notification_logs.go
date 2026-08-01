package models

import "time"

type NotificationLogs struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         int64     `gorm:"not null;index;uniqueIndex:unique_daily_notif,priority:1" json:"user_id"`
	Phone          string    `gorm:"type:varchar(20)" json:"phone"`
	Status         string    `gorm:"type:varchar(20);uniqueIndex:unique_daily_notif,priority:2" json:"status"`
	Message        string    `gorm:"type:text" json:"message"`
	SentDate       string    `gorm:"type:date;not null;uniqueIndex:unique_daily_notif,priority:3" json:"sent_date"`
	SentAt         time.Time `gorm:"autoCreateTime" json:"sent_at"`
	ResponseStatus string    `gorm:"type:varchar(255)" json:"response_status"`

	User Users `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}
