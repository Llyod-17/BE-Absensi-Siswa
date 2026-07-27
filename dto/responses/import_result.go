package responses

type FailedUser struct {
	Row    int    `json:"row"`
	NISN   string `json:"nisn"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type ImportResult struct {
	Inserted     int          `json:"inserted"`
	Duplicates   int          `json:"duplicates"`
	Failed       int          `json:"failed"`
	SkippedUsers []string     `json:"skipped_users"`
	FailedUsers  []FailedUser `json:"failed_users"`
}
