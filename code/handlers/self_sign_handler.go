package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"digital-signature-api/db"
	"digital-signature-api/models"
)

// SelfSignPositionRequest adalah body POST /api/dokumen/{id}/tanda-tangani,
// berisi posisi stempel tanda tangan pada dokumen.
type SelfSignPositionRequest struct {
	PageNumber int     `json:"page_number"`
	KoordinatX float64 `json:"koordinat_x"`
	KoordinatY float64 `json:"koordinat_y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
}

// parseSelfSignDocumentIDFromPath mem-parsing document ID dari path
// POST /api/dokumen/{id}/tanda-tangani
func parseSelfSignDocumentIDFromPath(path string) (uint, error) {
	idString := strings.TrimPrefix(path, "/api/dokumen/")
	idString = strings.TrimSuffix(idString, "/tanda-tangani")
	idString = strings.Trim(idString, "/")

	idUint64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(idUint64), nil
}

func parseSelfSignPositionBody(w http.ResponseWriter, r *http.Request) (SelfSignPositionRequest, bool) {
	var req SelfSignPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
		return req, false
	}

	if req.PageNumber <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Page number must be greater than 0"})
		return req, false
	}

	if req.KoordinatX < 0 || req.KoordinatX > 100 || req.KoordinatY < 0 || req.KoordinatY > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Koordinat X dan Y must be between 0 and 100"})
		return req, false
	}

	if req.Width <= 0 || req.Width > 100 || req.Height <= 0 || req.Height > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Width and height must be greater than 0 and not exceed 100"})
		return req, false
	}

	if req.KoordinatX+req.Width > 100 || req.KoordinatY+req.Height > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Signature area exceeds page boundaries"})
		return req, false
	}

	return req, true
}

func SelfSignDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed"})
		return
	}

	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized user"})
		return
	}

	documentID, err := parseSelfSignDocumentIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid document ID"})
		return
	}

	var document models.Document
	if err := db.DB.First(&document, documentID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Document not found"})
		return
	}

	if document.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "You do not have access to this document"})
		return
	}

	if document.Jenis != "self" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "This document is not set up for self-signing"})
		return
	}

	if document.Status != "draft" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Only draft document can start the self-signing process"})
		return
	}

	req, ok := parseSelfSignPositionBody(w, r)
	if !ok {
		return
	}

	tx := db.DB.Begin()
	if tx.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to start transaction"})
		return
	}

	signatureRequest := models.SignatureRequest{
		DokumenID:  document.ID,
		UserID:     userID,
		Urutan:     1,
		PageNumber: req.PageNumber,
		KoordinatX: req.KoordinatX,
		KoordinatY: req.KoordinatY,
		Width:      req.Width,
		Height:     req.Height,
		Status:     "menunggu",
	}

	if err := tx.Create(&signatureRequest).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to create signing request"})
		return
	}

	document.Status = "proses_ttd"
	if err := tx.Save(&document).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to update document status"})
		return
	}

	activityLog := models.ActivityLog{
		UserID:     userID,
		DokumenID:  document.ID,
		Aksi:       "sign",
		Keterangan: "User initiated self-signing process",
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
		"message":       "Signing request initiated. Call POST /api/permintaan-ttd/{id}/setujui to finish signing.",
		"dokumen_id":    document.ID,
		"permintaan_id": signatureRequest.ID,
		"status":        document.Status,
	})
}
