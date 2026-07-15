package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"digital-signature-api/db"
	"digital-signature-api/models"
	"digital-signature-api/utils"
)

func SignatureImageRootRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		UploadSignatureImage(w, r)
	case http.MethodGet:
		GetMySignatureImages(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
	}
}

func SignatureImageDetailRouter(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/preview") {
		PreviewSignatureImage(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		GetSignatureImageDetail(w, r)
	case http.MethodDelete:
		DeleteSignatureImage(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
	}
}

func parseSignatureImageIDFromPath(path string) (uint, error) {
	idString := strings.TrimPrefix(path, "/api/tanda-tangan/")
	idString = strings.TrimSuffix(idString, "/preview")
	idString = strings.Trim(idString, "/")

	idUint64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(idUint64), nil
}

func UploadSignatureImage(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid multipart form data",
		})
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Image file is required",
		})
		return
	}
	file.Close()

	if err := utils.ValidateSignatureImage(fileHeader); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	tipe := strings.TrimSpace(r.FormValue("tipe"))
	if tipe == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Tipe is required",
		})
		return
	}

	if tipe != "signature" && tipe != "paraf" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Tipe must be signature or paraf",
		})
		return
	}

	fileName := utils.GenerateFileName(tipe, fileHeader.Filename)
	filePath := filepath.Join("uploads", "signatures", fileName)

	if err := utils.SaveUploadedFile(fileHeader, filePath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to save uploaded image",
		})
		return
	}

	signatureImage := models.SignatureImage{
		UserID:   userID,
		Tipe:     tipe,
		FilePath: filepath.ToSlash(filePath),
	}

	result := db.DB.Create(&signatureImage)
	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to save signature image data",
		})
		return
	}

	createActivityLog(userID, 0, "upload_tanda_tangan", "User uploaded signature/paraf image")

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Signature image uploaded successfully",
		"data":    signatureImage,
	})
}

func GetMySignatureImages(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	var signatureImages []models.SignatureImage

	result := db.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&signatureImages)

	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to retrieve signature images",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Signature images retrieved successfully",
		"data":    signatureImages,
	})
}

func GetSignatureImageDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	signatureImageID, err := parseSignatureImageIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid signature image ID",
		})
		return
	}

	var signatureImage models.SignatureImage

	result := db.DB.First(&signatureImage, signatureImageID)
	if result.Error != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Signature image not found",
		})
		return
	}

	if signatureImage.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"message": "You do not have access to this signature image",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Signature image detail retrieved successfully",
		"data":    signatureImage,
	})
}

func PreviewSignatureImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
		return
	}

	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	signatureImageID, err := parseSignatureImageIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid signature image ID",
		})
		return
	}

	var signatureImage models.SignatureImage

	result := db.DB.First(&signatureImage, signatureImageID)
	if result.Error != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Signature image not found",
		})
		return
	}

	if signatureImage.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"message": "You do not have access to this signature image",
		})
		return
	}

	if _, err := os.Stat(signatureImage.FilePath); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Signature image file not found",
		})
		return
	}

	http.ServeFile(w, r, signatureImage.FilePath)
}

func DeleteSignatureImage(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	signatureImageID, err := parseSignatureImageIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid signature image ID",
		})
		return
	}

	var signatureImage models.SignatureImage

	result := db.DB.First(&signatureImage, signatureImageID)
	if result.Error != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Signature image not found",
		})
		return
	}

	if signatureImage.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"message": "You do not have access to this signature image",
		})
		return
	}

	if signatureImage.FilePath != "" {
		_ = os.Remove(signatureImage.FilePath)
	}

	if err := db.DB.Delete(&signatureImage).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to delete signature image",
		})
		return
	}

	createActivityLog(userID, 0, "hapus_tanda_tangan", "User deleted signature/paraf image")

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Signature image deleted successfully",
	})
}
