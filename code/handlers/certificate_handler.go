package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"digital-signature-api/db"
	"digital-signature-api/models"
)

func CertificateRootRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		CreateCertificate(w, r)
	case http.MethodGet:
		GetMyCertificates(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
	}
}

func CertificateDetailRouter(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/revoke") {
		RevokeCertificate(w, r)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{
		"message": "Route not found",
	})
}

func parseCertificateIDFromPath(path string) (uint, error) {
	idString := strings.TrimPrefix(path, "/api/sertifikat/")
	idString = strings.TrimSuffix(idString, "/revoke")
	idString = strings.Trim(idString, "/")

	idUint64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(idUint64), nil
}

func CreateCertificate(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	var activeCertificateCount int64

	db.DB.Model(&models.Certificate{}).
		Where("user_id = ? AND status = ?", userID, "active").
		Count(&activeCertificateCount)

	if activeCertificateCount > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "User already has an active certificate",
		})
		return
	}

	now := time.Now()

	certificate := models.Certificate{
		UserID:       userID,
		SerialNumber: fmt.Sprintf("CERT-%d-%d", userID, now.UnixNano()),
		PublicKey:    fmt.Sprintf("SIMULATED-PUBLIC-KEY-USER-%d-%d", userID, now.UnixNano()),
		Status:       "active",
		ValidFrom:    now,
		ValidUntil:   now.AddDate(1, 0, 0),
	}

	result := db.DB.Create(&certificate)
	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to create certificate",
		})
		return
	}

	createActivityLog(userID, 0, "buat_sertifikat", "User created a digital certificate simulation")

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Certificate created successfully",
		"data":    certificate,
	})
}

func GetMyCertificates(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Unauthorized user",
		})
		return
	}

	var certificates []models.Certificate

	result := db.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&certificates)

	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to retrieve certificates",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Certificates retrieved successfully",
		"data":    certificates,
	})
}

func RevokeCertificate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
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

	certificateID, err := parseCertificateIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid certificate ID",
		})
		return
	}

	var certificate models.Certificate

	result := db.DB.First(&certificate, certificateID)
	if result.Error != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"message": "Certificate not found",
		})
		return
	}

	if certificate.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"message": "You do not have access to this certificate",
		})
		return
	}

	if certificate.Status == "revoked" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Certificate is already revoked",
		})
		return
	}

	certificate.Status = "revoked"

	if err := db.DB.Save(&certificate).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to revoke certificate",
		})
		return
	}

	createActivityLog(userID, 0, "revoke_sertifikat", "User revoked a digital certificate simulation")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Certificate revoked successfully",
		"data":    certificate,
	})
}
