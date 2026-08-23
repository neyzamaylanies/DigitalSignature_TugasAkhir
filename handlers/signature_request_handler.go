package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"digital-signature-api/db"
	"digital-signature-api/models"
	"digital-signature-api/utils"
)

type ApproveSignatureRequestBody struct {
	TandaTanganID uint `json:"tanda_tangan_id"`
}

func SignatureRequestRootRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetMySignatureRequests(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
	}
}

func SignatureRequestDetailRouter(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/setujui"):
		ApproveSignatureRequest(w, r)
	case strings.HasSuffix(r.URL.Path, "/tolak"):
		RejectSignatureRequest(w, r)
	default:
		switch r.Method {
		case http.MethodGet:
			GetSignatureRequestDetail(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed"})
		}
	}
}

type RejectSignatureRequestBody struct {
	AlasanTolak string `json:"alasan_tolak"`
}

func ApproveSignatureRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed"})
		return
	}

	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized user"})
		return
	}

	signatureRequestID, err := parseSignatureRequestIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid signature request ID"})
		return
	}

	var signatureRequest models.SignatureRequest
	if err := db.DB.First(&signatureRequest, signatureRequestID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Signature request not found"})
		return
	}

	if signatureRequest.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "You are not the assigned signer for this request"})
		return
	}

	if signatureRequest.Status != "menunggu" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "This signature request has already been processed"})
		return
	}

	var document models.Document
	if err := db.DB.First(&document, signatureRequest.DokumenID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Related document not found"})
		return
	}

	// Dokumen self-sign berstatus "proses_ttd", dokumen cross-sign berstatus "menunggu_ttd".
	// Keduanya valid dieksekusi lewat endpoint approve yang sama, sesuai alur magang.
	if document.Status != "menunggu_ttd" && document.Status != "proses_ttd" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "This document is no longer awaiting signature actions"})
		return
	}

	// urutan validation: signer sebelumnya harus sudah diproses (selesai ATAU ditolak) dulu.
	// Signer dengan status "ditolak" dianggap sudah diproses dan tidak menghalangi giliran berikutnya.
	var pendingBefore int64
	if err := db.DB.Model(&models.SignatureRequest{}).
		Where("dokumen_id = ? AND urutan < ? AND status = ?", signatureRequest.DokumenID, signatureRequest.Urutan, "menunggu").
		Count(&pendingBefore).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to validate signing order"})
		return
	}

	if pendingBefore > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Please wait for the previous signer(s) to sign first"})
		return
	}

	var reqBody ApproveSignatureRequestBody
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil || reqBody.TandaTanganID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Tanda tangan ID is required"})
		return
	}

	var signatureImage models.SignatureImage
	if err := db.DB.First(&signatureImage, reqBody.TandaTanganID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Signature image not found"})
		return
	}

	if signatureImage.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "You do not have access to this signature image"})
		return
	}

	var certificate models.Certificate
	if err := db.DB.Where("user_id = ? AND status = ?", userID, "active").First(&certificate).Error; err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "You need an active certificate before signing a document"})
		return
	}

	// stamping dilakukan di luar transaction karena ini operasi file, bukan DB
	sourceFile := document.FilePath
	if document.FinalFilePath != "" {
		sourceFile = document.FinalFilePath
	}
	outputPath := fmt.Sprintf("uploads/documents/signed_%d_%d.pdf", document.ID, time.Now().UnixNano())

	if err := utils.StampSignatureOnPDF(
		sourceFile,
		outputPath,
		signatureImage.FilePath,
		signatureRequest.PageNumber,
		signatureRequest.KoordinatX, signatureRequest.KoordinatY, signatureRequest.Width, signatureRequest.Height,
	); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to stamp signature onto document: " + err.Error()})
		return
	}

	tx := db.DB.Begin()

	signatureRequest.Status = "selesai"
	if err := tx.Save(&signatureRequest).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to approve signature request"})
		return
	}

	document.FinalFilePath = outputPath

	// Dokumen baru difinalisasi setelah TIDAK ADA LAGI signer berstatus "menunggu".
	// Signer lain yang masih menunggu tetap boleh approve/reject di gilirannya masing-masing;
	// approve pada signer ini tidak langsung mengunci dokumen untuk signer berikutnya.
	var stillWaiting int64
	if err := tx.Model(&models.SignatureRequest{}).
		Where("dokumen_id = ? AND status = ?", signatureRequest.DokumenID, "menunggu").
		Count(&stillWaiting).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to check remaining signers"})
		return
	}

	if stillWaiting == 0 {
		var rejectedCount int64
		if err := tx.Model(&models.SignatureRequest{}).
			Where("dokumen_id = ? AND status = ?", signatureRequest.DokumenID, "ditolak").
			Count(&rejectedCount).Error; err != nil {
			tx.Rollback()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to check rejected signers"})
			return
		}

		if rejectedCount > 0 {
			document.Status = "selesai_dengan_penolakan"
		} else {
			document.Status = "selesai"
		}
	}

	if err := tx.Save(&document).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to update document"})
		return
	}

	transaction := models.CertificateTransaction{
		PermintaanTTDID: signatureRequest.ID,
		DokumenID:       document.ID,
		UserID:          userID,
		SertifikatID:    certificate.ID,
		Aksi:            "approve",
		FileResultPath:  outputPath,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to record certificate transaction"})
		return
	}

	activityLog := models.ActivityLog{
		UserID:     userID,
		DokumenID:  document.ID,
		Aksi:       "sign",
		Keterangan: "User approved and signed the document",
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

	dokumenSelesai := stillWaiting == 0
	dokumenStatus := ""
	if dokumenSelesai {
		dokumenStatus = document.Status
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "Signature request approved successfully",
		"permintaan_id":   signatureRequest.ID,
		"dokumen_id":      document.ID,
		"dokumen_selesai": dokumenSelesai,
		"dokumen_status":  dokumenStatus,
		"data":            document,
	})
}

func RejectSignatureRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed"})
		return
	}

	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized user"})
		return
	}

	signatureRequestID, err := parseSignatureRequestIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid signature request ID"})
		return
	}

	var requestBody RejectSignatureRequestBody
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
		return
	}

	requestBody.AlasanTolak = strings.TrimSpace(requestBody.AlasanTolak)
	if requestBody.AlasanTolak == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Alasan tolak is required"})
		return
	}

	var signatureRequest models.SignatureRequest
	if err := db.DB.First(&signatureRequest, signatureRequestID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Signature request not found"})
		return
	}

	if signatureRequest.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "You are not the assigned signer for this request"})
		return
	}

	if signatureRequest.Status != "menunggu" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "This signature request has already been processed"})
		return
	}

	var document models.Document
	if err := db.DB.First(&document, signatureRequest.DokumenID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Related document not found"})
		return
	}

	if document.Status != "menunggu_ttd" && document.Status != "proses_ttd" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "This document is no longer awaiting signature actions"})
		return
	}

	// urutan validation: signer sebelumnya harus sudah diproses (selesai ATAU ditolak) dulu.
	// Signer dengan status "ditolak" dianggap sudah diproses dan tidak menghalangi giliran berikutnya.
	var pendingBefore int64
	if err := db.DB.Model(&models.SignatureRequest{}).
		Where(
			"dokumen_id = ? AND urutan < ? AND status = ?",
			signatureRequest.DokumenID,
			signatureRequest.Urutan,
			"menunggu",
		).
		Count(&pendingBefore).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to validate signing order",
		})
		return
	}

	if pendingBefore > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Please wait for the previous signer(s) to sign first",
		})
		return
	}

	tx := db.DB.Begin()

	signatureRequest.Status = "ditolak"
	signatureRequest.AlasanTolak = requestBody.AlasanTolak
	if err := tx.Save(&signatureRequest).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to reject signature request"})
		return
	}

	// Penolakan SATU signer tidak otomatis membatalkan/mengunci seluruh dokumen.
	// Signer lain yang masih berstatus "menunggu" tetap dapat approve/reject sesuai gilirannya.
	// Status akhir dokumen baru ditentukan setelah tidak ada lagi signer berstatus "menunggu".
	var stillWaiting int64
	if err := tx.Model(&models.SignatureRequest{}).
		Where("dokumen_id = ? AND status = ?", signatureRequest.DokumenID, "menunggu").
		Count(&stillWaiting).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to check remaining signers"})
		return
	}

	if stillWaiting == 0 {
		var approvedCount int64
		if err := tx.Model(&models.SignatureRequest{}).
			Where("dokumen_id = ? AND status = ?", signatureRequest.DokumenID, "selesai").
			Count(&approvedCount).Error; err != nil {
			tx.Rollback()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to check approved signers"})
			return
		}

		if approvedCount > 0 {
			document.Status = "selesai_dengan_penolakan"
		} else {
			document.Status = "ditolak"
		}

		if err := tx.Save(&document).Error; err != nil {
			tx.Rollback()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to update document status"})
			return
		}
	}

	transaction := models.CertificateTransaction{
		PermintaanTTDID: signatureRequest.ID,
		DokumenID:       document.ID,
		UserID:          userID,
		Aksi:            "reject",
		AlasanTolak:     requestBody.AlasanTolak,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to record certificate transaction"})
		return
	}

	activityLog := models.ActivityLog{
		UserID:     userID,
		DokumenID:  document.ID,
		Aksi:       "reject",
		Keterangan: "User rejected the signature request: " + requestBody.AlasanTolak,
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

	dokumenSelesai := stillWaiting == 0
	dokumenStatus := ""
	if dokumenSelesai {
		dokumenStatus = document.Status
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "Signature request rejected successfully",
		"permintaan_id":   signatureRequest.ID,
		"dokumen_id":      document.ID,
		"dokumen_selesai": dokumenSelesai,
		"dokumen_status":  dokumenStatus,
		"data":            signatureRequest,
	})
}

func parseSignatureRequestIDFromPath(path string) (uint, error) {
	idString := strings.TrimPrefix(path, "/api/permintaan-ttd/")
	idString = strings.TrimSuffix(idString, "/setujui")
	idString = strings.TrimSuffix(idString, "/tolak")
	idString = strings.Trim(idString, "/")

	idUint64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(idUint64), nil
}

// SignatureRequestSummary menggabungkan data permintaan_ttd dengan informasi dokumen
// dan pengaju terkait, sesuai response yang didokumentasikan di API doc.
type SignatureRequestSummary struct {
	models.SignatureRequest
	DokumenJudul     string `json:"dokumen_judul,omitempty"`
	DokumenTipe      string `json:"dokumen_tipe,omitempty"`
	DokumenJenis     string `json:"dokumen_jenis,omitempty"`
	DokumenDeskripsi string `json:"dokumen_deskripsi,omitempty"`
	PengajuID        uint   `json:"pengaju_id,omitempty"`
	PengajuNama      string `json:"pengaju_nama,omitempty"`
}

func GetMySignatureRequests(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	var signatureRequests []models.SignatureRequest

	if err := db.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&signatureRequests).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to retrieve signature requests",
		})
		return
	}

	result := make([]SignatureRequestSummary, 0, len(signatureRequests))
	for _, signatureRequest := range signatureRequests {
		summary := SignatureRequestSummary{SignatureRequest: signatureRequest}

		var document models.Document
		if err := db.DB.First(&document, signatureRequest.DokumenID).Error; err == nil {
			summary.DokumenJudul = document.Judul
			summary.DokumenTipe = document.Tipe
			summary.DokumenJenis = document.Jenis
			summary.DokumenDeskripsi = document.Deskripsi
			summary.PengajuID = document.UserID

			var pengaju models.User
			if err := db.DB.First(&pengaju, document.UserID).Error; err == nil {
				summary.PengajuNama = pengaju.Name
			}
		}

		result = append(result, summary)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Signature requests retrieved successfully",
		"data":    result,
	})
}

func GetSignatureRequestDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	role := getRoleFromContext(r)

	signatureRequestID, err := parseSignatureRequestIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid signature request ID",
		})
		return
	}

	var signatureRequest models.SignatureRequest

	if err := db.DB.First(&signatureRequest, signatureRequestID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Signature request not found",
		})
		return
	}

	var document models.Document

	if err := db.DB.First(&document, signatureRequest.DokumenID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Related document not found",
		})
		return
	}

	isOwner := document.UserID == userID
	isSigner := signatureRequest.UserID == userID
	isAdmin := role == "admin"

	if !isOwner && !isSigner && !isAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"message": "You do not have access to this signature request",
		})
		return
	}

	var allSignatureRequests []models.SignatureRequest
	db.DB.
		Where("dokumen_id = ?", document.ID).
		Order("urutan ASC").
		Find(&allSignatureRequests)

	signers := make([]SignerProgress, 0, len(allSignatureRequests))
	for _, sr := range allSignatureRequests {
		var signerUser models.User
		db.DB.First(&signerUser, sr.UserID)

		signers = append(signers, SignerProgress{
			UserID:      sr.UserID,
			Nama:        signerUser.Name,
			Urutan:      sr.Urutan,
			PageNumber:  sr.PageNumber,
			KoordinatX:  sr.KoordinatX,
			KoordinatY:  sr.KoordinatY,
			Width:       sr.Width,
			Height:      sr.Height,
			Status:      sr.Status,
			AlasanTolak: sr.AlasanTolak,
		})
	}

	// giliran_saya bernilai true apabila permintaan milik user masih menunggu dan
	// tidak ada penanda tangan dengan urutan lebih kecil yang masih berstatus menunggu.
	giliranSaya := false
	if signatureRequest.Status == "menunggu" {
		giliranSaya = true
		for _, sr := range allSignatureRequests {
			if sr.Urutan < signatureRequest.Urutan && sr.Status == "menunggu" {
				giliranSaya = false
				break
			}
		}
	}

	var pengaju models.User
	db.DB.First(&pengaju, document.UserID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Signature request detail retrieved successfully",
		"data": map[string]interface{}{
			"signature_request": signatureRequest,
			"document":          document,
			"pengaju_id":        document.UserID,
			"pengaju_nama":      pengaju.Name,
			"penanda_tangan":    signers,
			"giliran_saya":      giliranSaya,
		},
	})
}
