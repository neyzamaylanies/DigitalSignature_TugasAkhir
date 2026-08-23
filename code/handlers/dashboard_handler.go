package handlers

import (
	"encoding/json"
	"net/http"

	"digital-signature-api/db"
	"digital-signature-api/models"
)

func GetDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Method not allowed",
		})
		return
	}

	userID := r.Context().Value("user_id")

	var totalDokumenSaya int64
	var totalPermintaanMenunggu int64
	var totalDokumenDitandatangani int64

	db.DB.Model(&models.Document{}).
		Where("user_id = ?", userID).
		Count(&totalDokumenSaya)

	db.DB.Model(&models.SignatureRequest{}).
		Where("user_id = ? AND status = ?", userID, "menunggu").
		Count(&totalPermintaanMenunggu)

	db.DB.Model(&models.SignatureRequest{}).
		Where("user_id = ? AND status = ?", userID, "selesai").
		Count(&totalDokumenDitandatangani)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Dashboard summary retrieved successfully",
		"data": map[string]int64{
			"dokumen_saya":           totalDokumenSaya,
			"permintaan_menunggu":    totalPermintaanMenunggu,
			"dokumen_ditandatangani": totalDokumenDitandatangani,
		},
	})
}
