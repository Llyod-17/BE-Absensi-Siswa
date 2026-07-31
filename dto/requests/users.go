package requests

type Login struct {
	Nisn     string `json:"nisn"`
	Password string `json:"password"`
}

type CreateUserReq struct {
	Nisn        string `json:"nisn"`
	FullName    string `json:"full_name"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	ClassGroup  string `json:"class_group"`
	ParentPhone string `json:"parent_phone"`
}

type UpdateUserReq struct {
	Nisn        string `json:"nisn"`
	FullName    string `json:"full_name"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	ClassGroup  string `json:"class_group"`
	ParentPhone string `json:"parent_phone"`
}

type ResetPasswordReq struct {
	NewPassword string `json:"new_password"`
}
