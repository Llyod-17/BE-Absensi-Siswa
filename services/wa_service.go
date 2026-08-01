package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

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

// ConnectWA — cek koneksi ke WAHA, dan pastikan session dalam status WORKING
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

	return EnsureWASessionWorking()
}

// EnsureWASessionWorking — pastikan session WAHA berstatus WORKING.
// Jika session STOPPED/STARTING, otomatis di-start ulang dan ditunggu sampai WORKING.
func EnsureWASessionWorking() error {
	cfg := config.AppConfig

	statusURL := fmt.Sprintf("%s/api/sessions/%s", cfg.WAHAURL, cfg.WAHASession)
	client := &http.Client{Timeout: 15 * time.Second}

	check := func() (string, error) {
		req, err := http.NewRequest(http.MethodGet, statusURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("X-Api-Key", cfg.WAHAAPIKey)

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		return string(body), nil
	}

	body, err := check()
	if err != nil {
		return fmt.Errorf("gagal cek status session: %w", err)
	}
	if strings.Contains(body, `"status":"WORKING"`) {
		return nil
	}

	// Session belum WORKING — coba start ulang
	log.Printf("[WAHA] Session %s status tidak WORKING — mencoba start ulang...", cfg.WAHASession)
	startURL := fmt.Sprintf("%s/api/sessions/%s/start", cfg.WAHAURL, cfg.WAHASession)
	req, err := http.NewRequest(http.MethodPost, startURL, bytes.NewBufferString("{}"))
	if err != nil {
		return fmt.Errorf("gagal buat request start: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", cfg.WAHAAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal start session: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Tunggu hingga WORKING (maks 2 menit)
	for i := 0; i < 12; i++ {
		time.Sleep(10 * time.Second)
		b, err := check()
		if err != nil {
			continue
		}
		if strings.Contains(b, `"status":"WORKING"`) {
			log.Printf("[WAHA] Session %s WORKING.", cfg.WAHASession)
			return nil
		}
		if strings.Contains(b, `"status":"STOPPED"`) && !strings.Contains(b, "STARTING") {
			log.Printf("[WAHA] Session %s tidak bisa start — mungkin butuh pairing QR ulang.", cfg.WAHASession)
			return fmt.Errorf("session %s tidak WORKING (mungkin perlu QR pairing ulang)", cfg.WAHASession)
		}
	}

	return fmt.Errorf("session %s tidak mencapai status WORKING dalam 2 menit", cfg.WAHASession)
}

// SendWAHA - Fungsi helper untuk kirim request ke WAHA API
func SendWAHA(phone, message string) error {
	cfg := config.AppConfig

	url := fmt.Sprintf("%s/api/sendText", cfg.WAHAURL)

	payload := map[string]string{
		"session": cfg.WAHASession,
		"chatId":  strings.TrimPrefix(phone, "+") + "@c.us",
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("WAHA sendText gagal: %d", resp.StatusCode)
	}

	return nil
}

// FUNGSI DIBAWAH INI PERLU DIHAPUS JIKA TIDAK DIGUNAKAN LAGI
// LogoutWA, StartQRPairing, GetWAStatus, dsb (sesuaikan dengan kebutuhan dashboard admin Anda)
