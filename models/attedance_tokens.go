package models

import "time"

type AttedanceTokens struct {
	ID                    int64     `gorm:"primaryKey;autoIncrement"`
	TokenCode             string    `gorm:"type:varchar(10);uniqueIndex;not null"`
	CreatedBy             int64     `gorm:"not null"`
	Category              string     `gorm:"type:enum('hadir','telat');default:'hadir';not null"`
	IsActive              bool       `gorm:"type:boolean"`
	LateAfter             *time.Time `gorm:"type:datetime"`
	ValidUntil            time.Time  `gorm:"type:datetime"`
	NotificationProcessed bool      `gorm:"type:boolean;default:false;not null"`
	CreatedAt             time.Time `gorm:"autoCreateTime"`

	User Users `gorm:"foreignKey:CreatedBy;references:ID"`
}
