package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"digital-signature-api/db"
	"digital-signature-api/models"
	"digital-signature-api/utils"
)

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func getUserIDFromContext(r *http.Request) (uint, bool) {
	userID, ok := r.Context().Value("user_id").(uint)
	return userID, ok
}

func getRoleFromContext(r *http.Request) string {
	role, _ := r.Context().Value("role").(string)
	return role
}

func createActivityLog(userID uint, dokumenID uint, aksi string, keterangan string) {
	log := models.ActivityLog{
		UserID:     userID,
		DokumenID:  dokumenID,
		Aksi:       aksi,
		Keterangan: keterangan,
	}

	db.DB.Create(&log)
}

func parseDocumentIDFromPath(path string, isDownload bool) (uint, error) {
	idString := strings.TrimPrefix(path, "/api/dokumen/")

	if isDownload {
		idString = strings.TrimSuffix(idString, "/download")
	}

	idString = strings.Trim(idString, "/")

	idUint64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(idUint64), nil
}

func getDocumentWithAccess(documentID uint, userID uint, role string) (models.Document, bool, error) {
	var document models.Document

	result := db.DB.First(&document, documentID)
	if result.Error != nil {
		return document, false, result.Error
	}

	if role == "admin" || document.UserID == userID {
		return document, true, nil
	}

	var count int64
	db.DB.Model(&models.SignatureRequest{}).
		Where("dokumen_id = ? AND user_id = ?", documentID, userID).
		Count(&count)

	if count > 0 {
		return document, true, nil
	}

	return document, false, nil
}

func UploadDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	err := r.ParseMultipartForm(25 << 20)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid multipart form data",
		})
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "PDF file is required",
		})
		return
	}
	file.Close()

	if err := utils.ValidatePDF(fileHeader); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	judul := strings.TrimSpace(r.FormValue("judul"))
	deskripsi := strings.TrimSpace(r.FormValue("deskripsi"))
	pesan := strings.TrimSpace(r.FormValue("pesan"))
	tipe := strings.TrimSpace(r.FormValue("tipe"))
	jenis := strings.TrimSpace(r.FormValue("jenis"))

	if judul == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Judul is required",
		})
		return
	}

	if jenis == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Jenis is required",
		})
		return
	}

	if jenis != "self" && jenis != "request" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Jenis must be self or request",
		})
		return
	}

	fileName := utils.GenerateFileName("dokumen", fileHeader.Filename)
	filePath := filepath.Join("uploads", "documents", fileName)

	if err := utils.SaveUploadedFile(fileHeader, filePath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to save uploaded file",
		})
		return
	}

	document := models.Document{
		UserID:        userID,
		Judul:         judul,
		Deskripsi:     deskripsi,
		Pesan:         pesan,
		Tipe:          tipe,
		Jenis:         jenis,
		FilePath:      filepath.ToSlash(filePath),
		FinalFilePath: "",
		Status:        "draft",
	}

	result := db.DB.Create(&document)
	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to save document metadata",
		})
		return
	}

	createActivityLog(userID, document.ID, "upload_dokumen", "User uploaded a PDF document")

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Document uploaded successfully",
		"data":    document,
	})
}

func GetMyDocuments(w http.ResponseWriter, r *http.Request) {
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

	var documents []models.Document

	result := db.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&documents)

	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to retrieve documents",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Documents retrieved successfully",
		"data":    documents,
	})
}

func DocumentRouter(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/download") {
		DownloadDocument(w, r)
		return
	}

	GetDocumentDetail(w, r)
}

func GetDocumentDetail(w http.ResponseWriter, r *http.Request) {
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

	role := getRoleFromContext(r)

	documentID, err := parseDocumentIDFromPath(r.URL.Path, false)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid document ID",
		})
		return
	}

	document, allowed, err := getDocumentWithAccess(documentID, userID, role)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Document not found",
		})
		return
	}

	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"message": "You do not have access to this document",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Document detail retrieved successfully",
		"data":    document,
	})
}

func DownloadDocument(w http.ResponseWriter, r *http.Request) {
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

	role := getRoleFromContext(r)

	documentID, err := parseDocumentIDFromPath(r.URL.Path, true)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid document ID",
		})
		return
	}

	document, allowed, err := getDocumentWithAccess(documentID, userID, role)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Document not found",
		})
		return
	}

	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"message": "You do not have access to this document",
		})
		return
	}

	filePath := document.FilePath
	if document.FinalFilePath != "" {
		filePath = document.FinalFilePath
	}

	if _, err := os.Stat(filePath); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Document file not found",
		})
		return
	}

	createActivityLog(userID, document.ID, "download_dokumen", "User downloaded a document")

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	http.ServeFile(w, r, filePath)
}
