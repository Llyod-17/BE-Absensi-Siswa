package utils

import (
	"errors"
	"time"
)

var jakarta, _ = time.LoadLocation("Asia/Jakarta")

func Now() time.Time {
	return time.Now().In(jakarta)
}

func DayRange(dateStr string) (time.Time, time.Time, error) {
	loc := jakarta

	var t time.Time
	var err error

	if dateStr == "" {
		t = time.Now().In(loc)
	} else {
		t, err = time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	return start, end, nil
}

// DateRange menerima startDate dan endDate (format YYYY-MM-DD).
// Jika kosong, default ke hari ini.
func DateRange(startStr, endStr string) (time.Time, time.Time, error) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)

	parseDate := func(s string, fallback time.Time) (time.Time, error) {
		if s == "" {
			return fallback, nil
		}
		return time.ParseInLocation("2006-01-02", s, loc)
	}

	start, err := parseDate(startStr, now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	end, err := parseDate(endStr, now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if end.Before(start) {
		return time.Time{}, time.Time{}, errors.New("end_date tidak boleh sebelum start_date")
	}

	startOfDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	endOfDay := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, loc)

	return startOfDay, endOfDay, nil
}
