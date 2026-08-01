package services

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/repo"
	"gorm.io/gorm"
)

const (
	StatusAlfa  = "alfa"
	StatusSakit = "sakit"
)

var (
	autoAlfaMutex          sync.Mutex
	isSendingNotifications int32
	waDisabledLogged       bool
)

// struct ringan buat nampung data target notif
type notifTarget struct {
	UserID      int64
	FullName    string
	Nisn        string
	ClassGroup  string
	ParentPhone string
	Status      string
}

// NormalizePhone — konversi nomor HP ke format internasional (+628xxx).
// Menghapus semua karakter non-angka (spasi, strip, kurung, dll) lalu
// mengkonversi semua variasi awalan ke +62:
//   0895345570902 -> +62895345570902
//   895345570902  -> +62895345570902
//   +62 895...    -> +62895345570902
//   62895...      -> +62895345570902
func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	// Hapus semua karakter non-digit
	re := regexp.MustCompile(`[^0-9]`)
	phone = re.ReplaceAllString(phone, "")

	switch {
	case strings.HasPrefix(phone, "0"):
		// 08xxx / 089xxx -> 628xxx
		return "+62" + phone[1:]
	case strings.HasPrefix(phone, "62"):
		// 628xxx (dari +62 yang sudah di-strip) -> +628xxx
		return "+" + phone
	case strings.HasPrefix(phone, "8"):
		// 812xxx tanpa leading 0 -> +62812xxx
		return "+62" + phone
	}
	return phone
}

// SendWhatsAppMessage — kirim pesan WA lewat WAHA API
func SendWhatsAppMessage(phone, message string) (string, error) {
	err := SendWAHA(NormalizePhone(phone), message)
	if err != nil {
		return "", fmt.Errorf("gagal kirim pesan via WAHA: %w", err)
	}
	return "sent_via_waha", nil
}

// BuildNotificationMessage — bikin teks pesan dinamis sesuai status (pake switch-case)
func BuildNotificationMessage(settings map[string]string, nama, nisn, kelas, status string) string {
	template := settings["wa_message_template"]
	schoolName := settings["school_name"]
	if schoolName == "" {
		schoolName = "SMK PLUS PELITA NUSANTARA" // fallback
	}

	if template != "" {
		// Format display status agar ramah dibaca
		displayStatus := strings.ToUpper(status)
		if strings.ToLower(status) == "belum_absen" || strings.ToLower(status) == "belum" {
			displayStatus = "BELUM ABSEN"
		}

		// Replace placeholders: {nama}, {nisn}, {kelas}, {status}, {nama_sekolah}
		r := strings.NewReplacer(
			"{nama}", nama,
			"{nisn}", nisn,
			"{kelas}", kelas,
			"{status}", displayStatus,
			"{nama_sekolah}", schoolName,
		)
		return r.Replace(template)
	}

	header := fmt.Sprintf(
		"Assalamualaikum Wr. Wb.\n\nYth. Bapak/Ibu Orang Tua/Wali dari:\n"+
			"  Nama  : *%s*\n"+
			"  NISN  : %s\n"+
			"  Kelas : %s\n\n",
		nama, nisn, kelas,
	)

	var body string
	switch strings.ToLower(status) {
	case "hadir":
		body = fmt.Sprintf(
			"Kami informasikan bahwa ananda *%s* hari ini telah hadir di sekolah dan melakukan absensi dengan status *HADIR*. "+
				"Terima kasih atas perhatian Bapak/Ibu.",
			nama,
		)
	case "telat":
		body = fmt.Sprintf(
			"Kami informasikan bahwa ananda *%s* hari ini hadir di sekolah namun tercatat *TERLAMBAT*. "+
				"Mohon Bapak/Ibu dapat mengingatkan putra/putri Anda untuk datang tepat waktu.",
			nama,
		)
	case StatusAlfa:
		body = fmt.Sprintf(
			"Kami informasikan bahwa ananda *%s* hari ini terpantau *ALFA*. "+
				"Mohon Bapak/Ibu dapat mengonfirmasi kehadiran putra/putri Anda.",
			nama,
		)
	case "belum_absen", "belum":
		body = fmt.Sprintf(
			"Kami informasikan bahwa ananda *%s* hari ini *BELUM MELAKUKAN ABSENSI*. "+
				"Mohon Bapak/Ibu dapat mengonfirmasi kehadiran putra/putri Anda.",
			nama,
		)
	case StatusSakit:
		body = fmt.Sprintf(
			"Kami informasikan bahwa hari ini ananda *%s* tidak dapat mengikuti kegiatan belajar mengajar karena *SAKIT*. "+
				"Kami pihak sekolah mendoakan agar ananda lekas sembuh dan dapat kembali beraktivitas seperti biasa.",
			nama,
		)
	case "izin":
		body = fmt.Sprintf(
			"Kami informasikan bahwa hari ini ananda *%s* tidak dapat mengikuti kegiatan belajar mengajar karena *IZIN*. "+
				"Terima kasih atas informasi dan perhatian Bapak/Ibu.",
			nama,
		)
	default:
		body = fmt.Sprintf(
			"Kami informasikan bahwa ananda *%s* hari ini tercatat dengan status *%s*.",
			nama, strings.ToUpper(status),
		)
	}

	footer := fmt.Sprintf("\n\nTerima kasih atas perhatian Bapak/Ibu.\nWassalamualaikum Wr. Wb.\n\n_Pesan ini dikirim secara otomatis oleh Sistem Absensi Sekolah %s._", schoolName)

	return header + body + footer
}

// StartNotificationSender — background worker untuk memproses antrean pesan WA (pending)
func StartNotificationSender(db *gorm.DB) {
	go func() {
		for {
			time.Sleep(15 * time.Second) // Cek setiap 15 detik

			// Gunakan atomic flag untuk mencegah worker ganda berjalan bersamaan
			if !atomic.CompareAndSwapInt32(&isSendingNotifications, 0, 1) {
				continue
			}

			// Jika notifikasi WA dimatikan (wa_enabled=false), tahan semua antrean
			// (termasuk sisa pending) sampai diaktifkan kembali.
			var waEnabled int64
			db.Model(&models.NotificationSettings{}).
				Where("setting_key = ? AND setting_value = ?", "wa_enabled", "true").
				Count(&waEnabled)
			if waEnabled == 0 {
				if !waDisabledLogged {
					log.Println("[WA-SENDER] wa_enabled=false — antrean pesan WA ditahan, tidak ada pengiriman.")
					waDisabledLogged = true
				}
				atomic.StoreInt32(&isSendingNotifications, 0)
				continue
			}
			waDisabledLogged = false

			var pendingLogs []models.NotificationLogs
			err := db.Where("response_status = ?", "pending").
				Order("id ASC").
				Find(&pendingLogs).Error

			if err != nil {
				log.Printf("[WA-SENDER] Gagal mengambil antrean: %v", err)
				atomic.StoreInt32(&isSendingNotifications, 0)
				continue
			}

			if len(pendingLogs) == 0 {
				atomic.StoreInt32(&isSendingNotifications, 0)
				continue
			}

			// Pastikan session WAHA WORKING sebelum mulai kirim (auto-restart jika mati)
			if err := EnsureWASessionWorking(); err != nil {
				log.Printf("[WA-SENDER] Session WAHA bermasalah: %v — antrean ditunda.", err)
				atomic.StoreInt32(&isSendingNotifications, 0)
				continue
			}

			log.Printf("[WA-SENDER] Memproses %d pesan pending...", len(pendingLogs))

			for _, l := range pendingLogs {
				// Abaikan jika ada log dengan status "hadir" (tidak dikirim via WA)
				if strings.ToLower(l.Status) == "hadir" {
					log.Printf("[WA-SENDER] Skip log ID %d — status 'hadir' diabaikan.", l.ID)
					db.Model(&models.NotificationLogs{}).
						Where("id = ?", l.ID).
						Update("response_status", "skipped: status_hadir_ignored")
					continue
				}

				log.Printf("[WA-SENDER] Mengirim pesan ke %s (log ID: %d)...", l.Phone, l.ID)
				responseStatus, err := SendWhatsAppMessage(l.Phone, l.Message)

				deliveryStatus := "success: " + responseStatus
				if err != nil {
					log.Printf("[WA-SENDER] Gagal mengirim ke %s: %v", l.Phone, err)
					deliveryStatus = "failed: " + err.Error()
					repo.InsertNotification("Gagal Kirim Pesan WA", fmt.Sprintf("Gagal mengirim pesan ke %s: %v", l.Phone, err), "WA error")
				} else {
					log.Printf("[WA-SENDER] Sukses mengirim ke %s", l.Phone)
				}

				// Potong string jika lebih dari 250 karakter (hitung per rune,
				// bukan byte, agar karakter UTF-8 tidak terpotong di tengah)
				// agar tidak terkena error Data too long di MySQL.
				if len([]rune(deliveryStatus)) > 250 {
					deliveryStatus = string([]rune(deliveryStatus)[:250])
				}

				// Update status log di database
				db.Model(&models.NotificationLogs{}).
					Where("id = ?", l.ID).
					Update("response_status", deliveryStatus)

				// Delay rate limit 30 detik setelah setiap kali mengirim pesan
				time.Sleep(30 * time.Second)
			}

			atomic.StoreInt32(&isSendingNotifications, 0)
		}
	}()
}

// NotifyPresentStudents — notifikasi WA untuk siswa HADIR di-nonaktifkan
func NotifyPresentStudents(db *gorm.DB) {
	log.Println("[WA] Notifikasi WA untuk status HADIR di-nonaktifkan (hanya mengirim notif selain hadir).")
}

// queueNotificationBatch — masukkan data notifikasi ke tabel antrean (notification_logs) dengan status "pending"
func queueNotificationBatch(db *gorm.DB, settings map[string]string, targets []notifTarget, today string) (queued, skipped int) {
	for _, t := range targets {
		// status hadir tidak dikirim via WA (hanya selain hadir: telat, sakit, alfa, izin, belum_absen)
		if strings.ToLower(t.Status) == "hadir" {
			log.Printf("[WA] Skip %s (status: hadir) — notifikasi status hadir di-nonaktifkan.", t.FullName)
			skipped++
			continue
		}

		// spam guard: kalo status ini udah pernah dikirim hari ini atau sedang antre, skip aja
		if repo.IsNotificationSentOrPendingToday(db, t.UserID, t.Status, today) {
			log.Printf("[WA] Skip %s (status: %s) — udah dikirim atau sedang antre hari ini.", t.FullName, t.Status)
			skipped++
			continue
		}

		// retry: jika ada row failed hari ini untuk user ini, reset ke pending (unique index user_id+status+sent_date
		// mencegah insert baru, jadi row lama yang failed dihidupkan kembali)
		res := db.Model(&models.NotificationLogs{}).
			Where("user_id = ? AND status = ? AND sent_date = ? AND response_status LIKE ?",
				t.UserID, t.Status, today, "failed%").
			Update("response_status", "pending")
		if res.RowsAffected > 0 {
			log.Printf("[WA] Retry %s (status: %s) — row failed direset ke pending.", t.FullName, t.Status)
			queued++
			continue
		}

		// ga ada nomor ortu? ya skip juga dong
		if t.ParentPhone == "" {
			log.Printf("[WA] Skip %s — nomor ortu kosong.", t.FullName)
			skipped++
			continue
		}

		// bangun pesan sesuai status
		message := BuildNotificationMessage(settings, t.FullName, t.Nisn, t.ClassGroup, t.Status)

		// catat ke tabel notification_logs dengan status "pending"
		entry := models.NotificationLogs{
			UserID:         t.UserID,
			Phone:          NormalizePhone(t.ParentPhone),
			Status:         t.Status,
			Message:        message,
			SentDate:       today,
			ResponseStatus: "pending",
		}
		if err := db.Create(&entry).Error; err != nil {
			log.Printf("[WA] Skip %s — gagal masuk antrean: %v", t.FullName, err)
			skipped++
			continue
		}
		queued++
	}

	return queued, skipped
}

// SendDailyWANotifications — kirim notifikasi WA harian untuk siswa dengan status selain HADIR.
// Dijalankan otomatis oleh cron setiap jam 08:30 WIB.
func SendDailyWANotifications(db *gorm.DB) {
	autoAlfaMutex.Lock()
	defer autoAlfaMutex.Unlock()

	settings, err := repo.GetNotificationSettingsMap(db)
	if err != nil {
		log.Printf("[WA] Gagal ambil settings: %v", err)
		return
	}

	if settings["wa_enabled"] != "true" {
		log.Println("[WA] Notifikasi WA lagi off.")
		return
	}

	today := repo.TodayDateString()
	var allTargets []notifTarget

	// 1. Tarik siswa yang BELUM ABSEN (tidak memiliki log di attedance_logs hari ini)
	unattendedStudents, err := repo.GetUnattendedStudents(db)
	if err != nil {
		log.Printf("[WA] Gagal ambil data siswa belum absen: %v", err)
	} else {
		for _, s := range unattendedStudents {
			allTargets = append(allTargets, notifTarget{
				UserID:      s.ID,
				FullName:    s.FullName,
				Nisn:        s.Nisn,
				ClassGroup:  s.ClassGroup,
				ParentPhone: s.ParentPhone,
				Status:      "belum_absen",
			})
		}
		log.Printf("[WA] Target Notif (Belum Absen): %d siswa.", len(unattendedStudents))
	}

	// 2. Tarik data siswa yang SUDAH ABSEN dengan status ALFA, SAKIT, TELAT, IZIN (semua kecuali HADIR)
	targetStudents, err := repo.GetStudentsByStatusToday(db, []string{StatusAlfa, StatusSakit, "telat", "izin"})
	if err != nil {
		log.Printf("[WA] Gagal ambil data target notif: %v", err)
	} else {
		for _, s := range targetStudents {
			allTargets = append(allTargets, notifTarget{
				UserID:      s.ID,
				FullName:    s.FullName,
				Nisn:        s.Nisn,
				ClassGroup:  s.ClassGroup,
				ParentPhone: s.ParentPhone,
				Status:      s.Status,
			})
		}
		log.Printf("[WA] Target Notif (Sudah Absen Selain Hadir): %d siswa.", len(targetStudents))
	}

	if len(allTargets) == 0 {
		log.Println("[WA] Ga ada siswa yg perlu dinotif (semua hadir atau tidak ada data).")
		return
	}

	log.Printf("[WA] Total %d siswa masuk antrian notif telat/alfa/sakit/izin/belum_absen.", len(allTargets))
	queued, skipped := queueNotificationBatch(db, settings, allTargets, today)
	log.Printf("[WA] Done — Dimasukkan ke antrean: %d | Skip: %d", queued, skipped)
}

// TestSendWhatsApp — kirim pesan test ke nomor tertentu (buat debugging)
func TestSendWhatsApp(phone, message string) (string, error) {
	return SendWhatsAppMessage(phone, message)
}
