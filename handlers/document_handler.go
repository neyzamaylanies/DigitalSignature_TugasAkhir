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

// signerInput merepresentasikan satu entri pada field "penanda_tangan" saat
// mengajukan dokumen ke pihak lain (POST /api/dokumen/ajukan), sesuai
// API_Documentation_Backend_Final dan alur referensi magang.
type signerInput struct {
	UserID     uint    `json:"user_id"`
	Urutan     int     `json:"urutan"`
	PageNumber int     `json:"page_number"`
	KoordinatX float64 `json:"koordinat_x"`
	KoordinatY float64 `json:"koordinat_y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
}

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
		DokumenID:  &dokumenID,
		Aksi:       aksi,
		Keterangan: keterangan,
	}

	db.DB.Create(&log)
}

func parseDocumentIDFromPath(path string, isDownload bool) (uint, error) {
	idString := strings.TrimPrefix(path, "/api/dokumen/")

	if isDownload {
		idString = strings.TrimPrefix(idString, "unduh/")
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

	if judul == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Judul is required",
		})
		return
	}

	if tipe == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Tipe is required",
		})
		return
	}

	// Endpoint ini khusus untuk self-signing (jenis selalu "self"), sesuai alur
	// referensi magang. Untuk mengajukan dokumen ke pihak lain, gunakan
	// POST /api/dokumen/ajukan (SubmitDocumentForSigning).
	jenis := "self"

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

	createActivityLog(userID, document.ID, "upload", "User uploaded a PDF document for self-signing")

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Document uploaded successfully",
		"data":    document,
	})
}

// SubmitDocumentForSigning menangani POST /api/dokumen/ajukan: mengunggah PDF
// sekaligus mengajukannya ke satu atau beberapa penanda tangan dalam SATU request,
// meniru persis alur AjukanTandaTangan pada proyek magang (referensi) dan
// endpoint POST /api/dokumen/ajukan pada API_Documentation_Backend_Final.
// Dokumen dan seluruh permintaan_ttd dibuat dalam satu transaction supaya atomik.
func SubmitDocumentForSigning(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(25 << 20); err != nil {
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

	if judul == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Judul is required",
		})
		return
	}

	if tipe == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Tipe is required",
		})
		return
	}

	var signers []signerInput
	rawSigners := r.FormValue("penanda_tangan")
	if strings.TrimSpace(rawSigners) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "penanda_tangan is required",
		})
		return
	}

	if err := json.Unmarshal([]byte(rawSigners), &signers); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid penanda_tangan format",
		})
		return
	}

	if len(signers) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "At least one signer is required",
		})
		return
	}

	seenSigner := map[uint]bool{}
	seenOrder := map[int]bool{}

	for _, signer := range signers {
		if signer.UserID == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Signer user ID is required"})
			return
		}

		if signer.UserID == userID {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Document owner cannot be a signer"})
			return
		}

		if signer.Urutan <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Urutan must be greater than 0"})
			return
		}

		if signer.PageNumber <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Page number must be greater than 0"})
			return
		}

		if signer.KoordinatX < 0 || signer.KoordinatX > 100 || signer.KoordinatY < 0 || signer.KoordinatY > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Koordinat X dan Y must be between 0 and 100"})
			return
		}

		if signer.Width <= 0 || signer.Width > 100 || signer.Height <= 0 || signer.Height > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Width and height must be greater than 0 and not exceed 100"})
			return
		}

		if signer.KoordinatX+signer.Width > 100 || signer.KoordinatY+signer.Height > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Signature area exceeds page boundaries"})
			return
		}

		if seenSigner[signer.UserID] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Duplicate signer is not allowed"})
			return
		}

		if seenOrder[signer.Urutan] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Duplicate signing order is not allowed"})
			return
		}

		var signerUser models.User
		if err := db.DB.First(&signerUser, signer.UserID).Error; err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "Signer user not found"})
			return
		}

		seenSigner[signer.UserID] = true
		seenOrder[signer.Urutan] = true
	}

	for urutan := 1; urutan <= len(signers); urutan++ {
		if !seenOrder[urutan] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Signing order must start from 1 and be sequential"})
			return
		}
	}

	fileName := utils.GenerateFileName("dokumen", fileHeader.Filename)
	filePath := filepath.Join("uploads", "documents", fileName)

	if err := utils.SaveUploadedFile(fileHeader, filePath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to save uploaded file",
		})
		return
	}

	tx := db.DB.Begin()
	if tx.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to start transaction"})
		return
	}

	document := models.Document{
		UserID:        userID,
		Judul:         judul,
		Deskripsi:     deskripsi,
		Pesan:         pesan,
		Tipe:          tipe,
		Jenis:         "request",
		FilePath:      filepath.ToSlash(filePath),
		FinalFilePath: "",
		Status:        "menunggu_ttd",
	}

	if err := tx.Create(&document).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to save document metadata"})
		return
	}

	signatureRequests := make([]models.SignatureRequest, 0, len(signers))
	for _, signer := range signers {
		signatureRequests = append(signatureRequests, models.SignatureRequest{
			DokumenID:  document.ID,
			UserID:     signer.UserID,
			Urutan:     signer.Urutan,
			PageNumber: signer.PageNumber,
			KoordinatX: signer.KoordinatX,
			KoordinatY: signer.KoordinatY,
			Width:      signer.Width,
			Height:     signer.Height,
			Status:     "menunggu",
		})
	}

	if err := tx.Create(&signatureRequests).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to create signature requests"})
		return
	}

	activityLog := models.ActivityLog{
		UserID:     userID,
		DokumenID:  &document.ID,
		Aksi:       "upload",
		Keterangan: "User submitted a document for signature request",
	}

	if err := tx.Create(&activityLog).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to create activity log"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to commit transaction"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":        "Document request created successfully",
		"dokumen_id":     document.ID,
		"jenis":          document.Jenis,
		"status":         document.Status,
		"permintaan_ttd": signatureRequests,
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
	path := r.URL.Path

	if path == "/api/dokumen/diunggah" {
		GetUploadedDocuments(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/dokumen/diunggah/") {
		GetUploadedDocumentDetail(w, r)
		return
	}

	if path == "/api/dokumen/ditandatangani" {
		GetSignedDocuments(w, r)
		return
	}

	if strings.HasPrefix(path, "/api/dokumen/unduh/") {
		DownloadDocument(w, r)
		return
	}

	if strings.HasSuffix(path, "/tanda-tangani") {
		SelfSignDocument(w, r)
		return
	}

	GetDocumentDetail(w, r)
}

func parseUploadedDocumentIDFromPath(path string) (uint, error) {
	idString := strings.TrimPrefix(path, "/api/dokumen/diunggah/")
	idString = strings.Trim(idString, "/")

	idUint64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(idUint64), nil
}

// DocumentProgress menggabungkan data dokumen dengan ringkasan progress penanda tangan,
// sesuai kebutuhan "Modul Dokumen Diunggah" (Minggu 8).
type DocumentProgress struct {
	models.Document
	TotalPenandaTangan int64 `json:"total_penanda_tangan"`
	SelesaiCount       int64 `json:"selesai_count"`
	DitolakCount       int64 `json:"ditolak_count"`
	MenungguCount      int64 `json:"menunggu_count"`
}

// GetUploadedDocuments menampilkan daftar dokumen yang pernah diunggah oleh user login
// (jenis self maupun request), lengkap dengan progress jumlah penanda tangan.
func GetUploadedDocuments(w http.ResponseWriter, r *http.Request) {
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
	if err := db.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&documents).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to retrieve documents",
		})
		return
	}

	result := make([]DocumentProgress, 0, len(documents))
	for _, document := range documents {
		progress := DocumentProgress{Document: document}

		db.DB.Model(&models.SignatureRequest{}).
			Where("dokumen_id = ?", document.ID).
			Count(&progress.TotalPenandaTangan)

		db.DB.Model(&models.SignatureRequest{}).
			Where("dokumen_id = ? AND status = ?", document.ID, "selesai").
			Count(&progress.SelesaiCount)

		db.DB.Model(&models.SignatureRequest{}).
			Where("dokumen_id = ? AND status = ?", document.ID, "ditolak").
			Count(&progress.DitolakCount)

		db.DB.Model(&models.SignatureRequest{}).
			Where("dokumen_id = ? AND status = ?", document.ID, "menunggu").
			Count(&progress.MenungguCount)

		result = append(result, progress)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Uploaded documents retrieved successfully",
		"data":    result,
	})
}

// SignerProgress merepresentasikan status satu penanda tangan pada suatu dokumen.
type SignerProgress struct {
	UserID      uint    `json:"user_id"`
	Nama        string  `json:"nama"`
	Urutan      int     `json:"urutan"`
	PageNumber  int     `json:"page_number"`
	KoordinatX  float64 `json:"koordinat_x"`
	KoordinatY  float64 `json:"koordinat_y"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	Status      string  `json:"status"`
	AlasanTolak string  `json:"alasan_tolak,omitempty"`
}

// GetUploadedDocumentDetail menampilkan detail dokumen yang diunggah user login,
// termasuk daftar penanda tangan dan progress tanda tangan masing-masing.
func GetUploadedDocumentDetail(w http.ResponseWriter, r *http.Request) {
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

	documentID, err := parseUploadedDocumentIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid document ID",
		})
		return
	}

	var document models.Document
	if err := db.DB.First(&document, documentID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Document not found",
		})
		return
	}

	if document.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"message": "You do not have access to this document",
		})
		return
	}

	var signatureRequests []models.SignatureRequest
	db.DB.
		Where("dokumen_id = ?", document.ID).
		Order("urutan ASC").
		Find(&signatureRequests)

	signers := make([]SignerProgress, 0, len(signatureRequests))
	for _, signatureRequest := range signatureRequests {
		var signerUser models.User
		db.DB.First(&signerUser, signatureRequest.UserID)

		signers = append(signers, SignerProgress{
			UserID:      signatureRequest.UserID,
			Nama:        signerUser.Name,
			Urutan:      signatureRequest.Urutan,
			PageNumber:  signatureRequest.PageNumber,
			KoordinatX:  signatureRequest.KoordinatX,
			KoordinatY:  signatureRequest.KoordinatY,
			Width:       signatureRequest.Width,
			Height:      signatureRequest.Height,
			Status:      signatureRequest.Status,
			AlasanTolak: signatureRequest.AlasanTolak,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Uploaded document detail retrieved successfully",
		"data": map[string]interface{}{
			"dokumen":        document,
			"penanda_tangan": signers,
		},
	})
}

// SignedDocument menggabungkan data dokumen dengan metadata sertifikat yang dipakai
// user login saat menandatangani dokumen tersebut.
type SignedDocument struct {
	models.Document
	SertifikatSerialNumber string `json:"sertifikat_serial_number,omitempty"`
	SertifikatStatus       string `json:"sertifikat_status,omitempty"`
}

// GetSignedDocuments menampilkan daftar dokumen selesai yang dapat diakses user login,
// baik dokumen yang diunggah sendiri maupun dokumen yang sudah ditandatangani oleh user tersebut.
func GetSignedDocuments(w http.ResponseWriter, r *http.Request) {
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
	if err := db.DB.
		Where(
			"status IN (?, ?) AND (user_id = ? OR id IN (SELECT dokumen_id FROM permintaan_ttd WHERE user_id = ? AND status = ?))",
			"selesai", "selesai_dengan_penolakan",
			userID,
			userID, "selesai",
		).
		Order("created_at DESC").
		Find(&documents).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to retrieve signed documents",
		})
		return
	}

	result := make([]SignedDocument, 0, len(documents))
	for _, document := range documents {
		signedDocument := SignedDocument{Document: document}

		var transaction models.CertificateTransaction
		if err := db.DB.
			Where("dokumen_id = ? AND user_id = ? AND sertifikat_id > 0", document.ID, userID).
			Order("created_at DESC").
			First(&transaction).Error; err == nil {
			var certificate models.Certificate
			if err := db.DB.First(&certificate, transaction.SertifikatID).Error; err == nil {
				signedDocument.SertifikatSerialNumber = certificate.SerialNumber
				signedDocument.SertifikatStatus = certificate.Status
			}
		}

		result = append(result, signedDocument)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Signed documents retrieved successfully",
		"data":    result,
	})
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

	createActivityLog(userID, document.ID, "download", "User downloaded a document")

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	http.ServeFile(w, r, filePath)
}
