package responses

type UserRes struct {
	ID               int64  `json:"id"`
	Nisn             string `json:"nisn"`
	FullName         string `json:"full_name"`
	Username         string `json:"username"`
	Role             string `json:"role"`
	Status           string `json:"status"`
	ClassGroup       string `json:"class_group"`
	ParentPhone      string `json:"parent_phone"`
	AttendanceStatus string `json:"attendance_status"`
}

type UserMini struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}
