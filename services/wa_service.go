package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/config"
)

// InitWA untuk WAHA hanya perlu validasi konfigurasi
func InitWA() error {
	cfg := config.AppConfig
	if cfg.WAHAURL == "" || cfg.WAHAAPIKey == "" || cfg.WAHASession == "" {
		return fmt.Errorf("konfigurasi WAHA tidak lengkap di .env")
	}
	return nil
}

// ConnectWA untuk WAHA, kita cek koneksi ke API
func ConnectWA() error {
	cfg := config.AppConfig

	url := fmt.Sprintf("%s/api/sessions/%s", cfg.WAHAURL, cfg.WAHASession)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("gagal membuat request: %w", err)
	}

	req.Header.Set("X-Api-Key", cfg.WAHAAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal konek ke WAHA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WAHA API status: %d", resp.StatusCode)
	}

	return nil
}

// SendWAHA - Fungsi helper untuk kirim request ke WAHA API
func SendWAHA(phone, message string) error {
	cfg := config.AppConfig

	url := fmt.Sprintf("%s/api/sendText", cfg.WAHAURL)

	payload := map[string]string{
		"session": cfg.WAHASession,
		"chatId":  phone + "@c.us",
		"text":    message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", cfg.WAHAAPIKey)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WAHA sendText gagal: %d", resp.StatusCode)
	}

	return nil
}

// FUNGSI DIBAWAH INI PERLU DIHAPUS JIKA TIDAK DIGUNAKAN LAGI
// LogoutWA, StartQRPairing, GetWAStatus, dsb (sesuaikan dengan kebutuhan dashboard admin Anda)
