package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"digital-signature-api/db"
	"digital-signature-api/models"
)

type SignatureRequestInput struct {
	UserID     uint    `json:"user_id"`
	Urutan     int     `json:"urutan"`
	PageNumber int     `json:"page_number"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
}

type CreateSignatureRequestBody struct {
	DokumenID uint                    `json:"dokumen_id"`
	Signers   []SignatureRequestInput `json:"signers"`
}

func SignatureRequestRootRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		CreateSignatureRequest(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
	}
}

func SignatureRequestMeRouter(w http.ResponseWriter, r *http.Request) {
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
	switch r.Method {
	case http.MethodGet:
		GetSignatureRequestDetail(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
	}
}

func parseSignatureRequestIDFromPath(path string) (uint, error) {
	idString := strings.TrimPrefix(path, "/api/permintaan-ttd/")
	idString = strings.Trim(idString, "/")

	idUint64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(idUint64), nil
}

func CreateSignatureRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	var requestBody CreateSignatureRequestBody

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	if requestBody.DokumenID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Dokumen ID is required",
		})
		return
	}

	if len(requestBody.Signers) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "At least one signer is required",
		})
		return
	}

	var document models.Document

	if err := db.DB.First(&document, requestBody.DokumenID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Document not found",
		})
		return
	}

	if document.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"message": "You do not have access to create request for this document",
		})
		return
	}

	if document.Status != "draft" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Only draft document can be submitted for signature request",
		})
		return
	}

	seenSigner := map[uint]bool{}
	seenOrder := map[int]bool{}

	var signatureRequests []models.SignatureRequest

	for _, signer := range requestBody.Signers {
		if signer.UserID == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"message": "Signer user ID is required",
			})
			return
		}

		if signer.Urutan <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"message": "Urutan must be greater than 0",
			})
			return
		}

		if signer.PageNumber <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"message": "Page number must be greater than 0",
			})
			return
		}

		if signer.Width <= 0 || signer.Height <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"message": "Width and height must be greater than 0",
			})
			return
		}

		if seenSigner[signer.UserID] {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"message": "Duplicate signer is not allowed",
			})
			return
		}

		if seenOrder[signer.Urutan] {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"message": "Duplicate signing order is not allowed",
			})
			return
		}

		var signerUser models.User

		if err := db.DB.First(&signerUser, signer.UserID).Error; err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"message": "Signer user not found",
			})
			return
		}

		seenSigner[signer.UserID] = true
		seenOrder[signer.Urutan] = true

		signatureRequest := models.SignatureRequest{
			DokumenID:  requestBody.DokumenID,
			UserID:     signer.UserID,
			Urutan:     signer.Urutan,
			PageNumber: signer.PageNumber,
			Width:      signer.Width,
			Height:     signer.Height,
			Status:     "menunggu",
		}

		signatureRequests = append(signatureRequests, signatureRequest)
	}

	tx := db.DB.Begin()

	if err := tx.Create(&signatureRequests).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to create signature requests",
		})
		return
	}

	document.Jenis = "request"
	document.Status = "menunggu_ttd"

	if err := tx.Save(&document).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to update document status",
		})
		return
	}

	activityLog := models.ActivityLog{
		UserID:     userID,
		DokumenID:  document.ID,
		Aksi:       "upload",
		Keterangan: "User submitted document for signature request",
	}

	if err := tx.Create(&activityLog).Error; err != nil {
		tx.Rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to create activity log",
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to commit transaction",
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Signature request created successfully",
		"data":    signatureRequests,
	})
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Signature requests retrieved successfully",
		"data":    signatureRequests,
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Signature request detail retrieved successfully",
		"data": map[string]interface{}{
			"signature_request": signatureRequest,
			"document":          document,
		},
	})
}
