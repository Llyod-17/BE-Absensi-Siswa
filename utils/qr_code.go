package utils

import "github.com/skip2/go-qrcode"

func GenerateQRCode(token string) ([]byte, error) {
	return qrcode.Encode(
		token,
		qrcode.Highest, // Level H error correction (30% capacity)
		512,            // 512x512 crisp high-res
	)
}