package handlers

import (
	"github.com/KicauOrgspark/BE-Absensi-Siswa/database"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/requests"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/dto/responses"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/mappers"
	"github.com/KicauOrgspark/BE-Absensi-Siswa/models"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// GetUsers godoc
// @Summary Ambil daftar pengguna
// @Description Mengambil daftar semua pengguna dengan paginasi, filter role, filter kelas, dan search kata kunci (nama/NISN)
// @Tags users
// @Produce json
// @Param page query int false "Halaman (default: 1)"
// @Param limit query int false "Batas item per halaman (default: 20)"
// @Param role query string false "Filter berdasarkan Role (siswa, guru, admin, superadmin)"
// @Param class_group query string false "Filter berdasarkan Kelas"
// @Param search query string false "Pencarian berdasarkan Nama Lengkap atau NISN"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /users [get]
func GetUsers(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	role := c.Query("role")
	classGroup := c.Query("class_group")
	search := c.Query("search")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	query := database.DB.Model(&models.Users{})

	if role != "" {
		query = query.Where("role = ?", role)
	}
	if classGroup != "" {
		query = query.Where("class_group = ?", classGroup)
	}
	if search != "" {
		query = query.Where("full_name LIKE ? OR nisn LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal menghitung total pengguna"})
	}

	var users []models.Users
	if err := query.
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal mengambil data pengguna"})
	}

	var result []responses.UserRes
	for _, u := range users {
		result = append(result, mappers.ToUserResponse(u))
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data": fiber.Map{
			"users":      result,
			"totalPages": totalPages,
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

// GetUserByID godoc
// @Summary Ambil detail pengguna
// @Description Mengambil detail lengkap pengguna berdasarkan ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /users/{id} [get]
func GetUserByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID tidak valid"})
	}

	var user models.Users
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "pengguna tidak ditemukan"})
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    mappers.ToUserResponse(user),
	})
}

// CreateUser godoc
// @Summary Tambah pengguna baru
// @Description Menambahkan pengguna baru secara manual dengan role dan password
// @Tags users
// @Accept json
// @Produce json
// @Param request body requests.CreateUserReq true "Create User Request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /users [post]
func CreateUser(c *fiber.Ctx) error {
	var req requests.CreateUserReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "payload tidak valid"})
	}

	if req.Username == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username wajib diisi"})
	}
	if req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "password wajib diisi"})
	}

	if req.Nisn == "" {
		return c.Status(400).JSON(fiber.Map{"error": "NISN wajib diisi untuk siswa"})
	}

	if req.FullName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "fullname wajib iisi"})
	}

	// Validasi unique username
	var existUsername models.Users
	if err := database.DB.Where("username = ?", req.Username).First(&existUsername).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{"error": "username sudah digunakan"})
	}

	// Validasi unique NISN jika diisi
	if req.Nisn != "" {
		var existNisn models.Users
		if err := database.DB.Where("nisn = ?", req.Nisn).First(&existNisn).Error; err == nil {
			return c.Status(409).JSON(fiber.Map{"error": "NISN sudah digunakan"})
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal memproses password"})
	}

	user := models.Users{
		Nisn:        req.Nisn,
		FullName:    req.FullName,
		Username:    req.Username,
		Password:    string(hashedPassword),
		Role:        req.Role,
		Status:      req.Status,
		ClassGroup:  req.ClassGroup,
		ParentPhone: req.ParentPhone,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal menyimpan pengguna"})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "success",
		"data":    mappers.ToUserResponse(user),
	})
}

// UpdateUser godoc
// @Summary Edit pengguna
// @Description Mengubah data profil pengguna (tidak termasuk password)
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body requests.UpdateUserReq true "Update User Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /users/{id} [put]
func UpdateUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID tidak valid"})
	}

	var req requests.UpdateUserReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "payload tidak valid"})
	}

	var user models.Users
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "pengguna tidak ditemukan"})
	}

	if req.Username == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username wajib diisi"})
	}
	if req.Role == "" {
		return c.Status(400).JSON(fiber.Map{"error": "role wajib diisi"})
	}
	if req.Role == "siswa" && req.Nisn == "" {
		return c.Status(400).JSON(fiber.Map{"error": "NISN wajib diisi untuk siswa"})
	}

	// Cek keunikan username jika diubah
	if req.Username != user.Username {
		var existUsername models.Users
		if err := database.DB.Where("username = ? AND id != ?", req.Username, id).First(&existUsername).Error; err == nil {
			return c.Status(409).JSON(fiber.Map{"error": "username sudah digunakan"})
		}
	}

	// Cek keunikan NISN jika diubah
	if req.Nisn != "" && req.Nisn != user.Nisn {
		var existNisn models.Users
		if err := database.DB.Where("nisn = ? AND id != ?", req.Nisn, id).First(&existNisn).Error; err == nil {
			return c.Status(409).JSON(fiber.Map{"error": "NISN sudah digunakan"})
		}
	}

	user.Nisn = req.Nisn
	user.FullName = req.FullName
	user.Username = req.Username
	user.Role = req.Role
	user.Status = req.Status
	user.ClassGroup = req.ClassGroup
	user.ParentPhone = req.ParentPhone

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal memperbarui pengguna"})
	}

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    mappers.ToUserResponse(user),
	})
}

// DeleteUser godoc
// @Summary Hapus pengguna
// @Description Menghapus pengguna dari sistem. Admin tidak bisa menghapus diri sendiri.
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /users/{id} [delete]
func DeleteUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID tidak valid"})
	}

	currentUserID := c.Locals("user_id").(int64)
	if int64(id) == currentUserID {
		return c.Status(400).JSON(fiber.Map{"error": "tidak dapat menghapus diri sendiri"})
	}

	var user models.Users
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "pengguna tidak ditemukan"})
	}

	if err := database.DB.Delete(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal menghapus pengguna"})
	}

	return c.JSON(fiber.Map{
		"message": "success",
	})
}

// ResetPassword godoc
// @Summary Reset password pengguna
// @Description Mengubah password pengguna secara paksa oleh Admin
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body requests.ResetPasswordReq true "Reset Password Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /users/{id}/reset-password [post]
func ResetPassword(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID tidak valid"})
	}

	var req requests.ResetPasswordReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "payload tidak valid"})
	}

	if req.NewPassword == "" {
		return c.Status(400).JSON(fiber.Map{"error": "password baru wajib diisi"})
	}

	var user models.Users
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "pengguna tidak ditemukan"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal memproses password"})
	}

	user.Password = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal menyimpan password baru"})
	}

	return c.JSON(fiber.Map{
		"message": "success",
	})
}
