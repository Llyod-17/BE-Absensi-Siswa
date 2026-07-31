package models

//model users
type Users struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	Nisn        string `gorm:"type:varchar(10);unique"`
	FullName    string `gorm:"type:varchar(100)"`
	Username    string `gorm:"type:varchar(50);uniqueIndex;not null"`
	Password    string `gorm:"type:varchar(225);not null"`
	Role        string `gorm:"type:enum('siswa','guru','admin','superadmin');default:'siswa'"`
	ClassGroup  string `gorm:"type:varchar(20)"`
	ParentPhone string `gorm:"column:parent_phone;type:varchar(20)" json:"parent_phone"`
	Status 			string `gorm:"type:enum('PKL', 'AKTIF', 'NONAKTIF')";default:'AKTIF'`

	AttedanceTokens []AttedanceTokens `gorm:"foreignKey:CreatedBy"`
	AttedanceLogs   []AttedanceLogs   `gorm:"foreignKey:UserID"`
}
