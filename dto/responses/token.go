package responses

import "time"

type TokenRes struct {
	ID         int64      `json:"id"`
	TokenCode  string     `json:"token_code"`
	CreatedBy  UserMini   `json:"created_by"`
	Category   string     `json:"category"`
	IsActive   bool       `json:"is_active"`
	LateAfter  *time.Time `json:"late_after"`
	ValidUntil time.Time  `json:"valid_until"`
	CreatedAt  time.Time  `json:"created_at"`
}
