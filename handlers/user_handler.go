package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"digital-signature-api/db"
	"digital-signature-api/models"
	"digital-signature-api/utils"
)

func UserRootRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetUsers(w, r)
	case http.MethodPost:
		CreateUser(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
	}
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Method not allowed",
		})
		return
	}

	var users []models.User

	result := db.DB.Order("id ASC").Find(&users)
	if result.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Failed to retrieve users",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Users retrieved successfully",
		"data":    users,
	})
}

// CreateUserBody adalah payload POST /api/users.
type CreateUserBody struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// CreateUser menangani POST /api/users (admin only): membuat akun user baru
// tanpa lewat proses registrasi mandiri. Berguna untuk admin mendaftarkan
// pegawai/pihak lain yang perlu jadi penanda tangan.
func CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
		return
	}

	var body CreateUserBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	body.Role = strings.TrimSpace(body.Role)

	if body.Name == "" || body.Email == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Name, email, and password are required",
		})
		return
	}

	if len(body.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Password must be at least 6 characters",
		})
		return
	}

	if body.Role == "" {
		body.Role = "user"
	}

	if body.Role != "admin" && body.Role != "user" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Role must be admin or user",
		})
		return
	}

	var existing int64
	db.DB.Model(&models.User{}).Where("email = ?", body.Email).Count(&existing)
	if existing > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Email is already registered",
		})
		return
	}

	hashedPassword, err := utils.HashPassword(body.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to hash password",
		})
		return
	}

	user := models.User{
		Name:     body.Name,
		Email:    body.Email,
		Password: hashedPassword,
		Role:     body.Role,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to create user",
		})
		return
	}

	if adminID, ok := getUserIDFromContext(r); ok {
		createActivityLog(adminID, 0, "create_user", "Admin created a new user account: "+user.Email)
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User created successfully",
		"data":    user,
	})
}

// UserDetailRouter menangani semua path di bawah /api/users/{id} (admin only):
// GET detail, PATCH edit, PATCH .../nonaktifkan, PATCH .../reset-password.
func UserDetailRouter(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/nonaktifkan"):
		DeactivateUser(w, r)
	case strings.HasSuffix(r.URL.Path, "/reset-password"):
		ResetUserPassword(w, r)
	default:
		switch r.Method {
		case http.MethodGet:
			GetUserDetail(w, r)
		case http.MethodPatch:
			UpdateUser(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed"})
		}
	}
}

// parseUserIDFromPath mem-parsing user ID dari path /api/users/{id}[/suffix].
func parseUserIDFromPath(path string) (uint, error) {
	idString := strings.TrimPrefix(path, "/api/users/")
	idString = strings.TrimSuffix(idString, "/nonaktifkan")
	idString = strings.TrimSuffix(idString, "/reset-password")
	idString = strings.Trim(idString, "/")

	idUint64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(idUint64), nil
}

// GetUserDetail menangani GET /api/users/{id} (admin only).
func GetUserDetail(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUserIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid user ID"})
		return
	}

	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "User not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User detail retrieved successfully",
		"data":    user,
	})
}

// UpdateUserBody adalah payload PATCH /api/users/{id}.
type UpdateUserBody struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// UpdateUser menangani PATCH /api/users/{id} (admin only): ubah nama/role
// user lain. Admin tidak boleh mengubah role dirinya sendiri lewat endpoint
// ini supaya tidak ada risiko admin tunggal kehilangan akses admin.
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	adminID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized user"})
		return
	}

	userID, err := parseUserIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid user ID"})
		return
	}

	var body UpdateUserBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	body.Role = strings.TrimSpace(body.Role)

	if body.Name == "" && body.Role == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Nothing to update"})
		return
	}

	if body.Role != "" && body.Role != "admin" && body.Role != "user" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Role must be admin or user"})
		return
	}

	if body.Role != "" && userID == adminID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "You cannot change your own role"})
		return
	}

	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "User not found"})
		return
	}

	if body.Name != "" {
		user.Name = body.Name
	}
	if body.Role != "" {
		user.Role = body.Role
	}

	if err := db.DB.Save(&user).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to update user"})
		return
	}

	createActivityLog(adminID, 0, "update_user", "Admin updated user account: "+user.Email)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User updated successfully",
		"data":    user,
	})
}

// DeactivateUser menangani PATCH /api/users/{id}/nonaktifkan (admin only).
// Akun dinonaktifkan (bukan dihapus) supaya relasi FK ke dokumen/log lama
// tetap utuh, dan bisa diaktifkan kembali kapan saja.
func DeactivateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed"})
		return
	}

	adminID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized user"})
		return
	}

	userID, err := parseUserIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid user ID"})
		return
	}

	if userID == adminID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "You cannot deactivate your own account"})
		return
	}

	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "User not found"})
		return
	}

	if !user.IsActive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "User is already deactivated"})
		return
	}

	user.IsActive = false
	if err := db.DB.Save(&user).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to deactivate user"})
		return
	}

	createActivityLog(adminID, 0, "deactivate_user", "Admin deactivated user account: "+user.Email)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User deactivated successfully",
		"data":    user,
	})
}

// ResetUserPasswordBody adalah payload PATCH /api/users/{id}/reset-password.
type ResetUserPasswordBody struct {
	NewPassword string `json:"new_password"`
}

// ResetUserPassword menangani PATCH /api/users/{id}/reset-password (admin only):
// admin set password baru untuk user yang lupa password, tanpa perlu tahu
// password lamanya.
func ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed"})
		return
	}

	adminID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized user"})
		return
	}

	userID, err := parseUserIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid user ID"})
		return
	}

	var body ResetUserPasswordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
		return
	}

	if len(body.NewPassword) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Password must be at least 6 characters"})
		return
	}

	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "User not found"})
		return
	}

	hashedPassword, err := utils.HashPassword(body.NewPassword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to hash password"})
		return
	}

	user.Password = hashedPassword
	if err := db.DB.Save(&user).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to reset password"})
		return
	}

	createActivityLog(adminID, 0, "reset_password", "Admin reset password for user: "+user.Email)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Password reset successfully",
	})
}