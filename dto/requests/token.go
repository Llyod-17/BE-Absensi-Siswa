package requests

type TokenReq struct {
	Duration int    `json:"duration"`
	Category string `json:"category"` // "hadir" atau "telat"
}

type SubmitToken struct {
	TokenCode string  `json:"token_code"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type UpdatedToken struct {
	ValidUntil string `json:"valid_until"`
	Category   string `json:"category"`
	LateAfter  string `json:"late_after"`
}

type QuickUpdateToken struct {
	Date string `json:"date"` // Format "YYYY-MM-DD", optional
}
