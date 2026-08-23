package handlers

import (
	"net/http"
	"strconv"

	"digital-signature-api/db"
	"digital-signature-api/models"
)

const (
	defaultActivityLogLimit = 50
	maxActivityLogLimit     = 200
)

func parseActivityLogLimit(r *http.Request) int {
	limitParam := r.URL.Query().Get("limit")
	if limitParam == "" {
		return defaultActivityLogLimit
	}

	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 {
		return defaultActivityLogLimit
	}

	if limit > maxActivityLogLimit {
		return maxActivityLogLimit
	}

	return limit
}

type ActivityLogSummary struct {
	models.ActivityLog
	UserName string `json:"user_name,omitempty"`
}

func GetMyActivityLogs(w http.ResponseWriter, r *http.Request) {
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

	query := db.DB.Where("user_id = ?", userID)

	if aksi := r.URL.Query().Get("aksi"); aksi != "" {
		query = query.Where("aksi = ?", aksi)
	}

	var logs []models.ActivityLog
	if err := query.
		Order("created_at DESC").
		Limit(parseActivityLogLimit(r)).
		Find(&logs).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to retrieve activity logs",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Activity logs retrieved successfully",
		"data":    logs,
	})
}

func GetAllActivityLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"message": "Method not allowed",
		})
		return
	}

	query := db.DB.Model(&models.ActivityLog{})

	if userIDParam := r.URL.Query().Get("user_id"); userIDParam != "" {
		userIDFilter, err := strconv.ParseUint(userIDParam, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"message": "Invalid user_id filter",
			})
			return
		}
		query = query.Where("user_id = ?", userIDFilter)
	}

	if aksi := r.URL.Query().Get("aksi"); aksi != "" {
		query = query.Where("aksi = ?", aksi)
	}

	var logs []models.ActivityLog
	if err := query.
		Order("created_at DESC").
		Limit(parseActivityLogLimit(r)).
		Find(&logs).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to retrieve activity log",
		})
		return
	}

	result := make([]ActivityLogSummary, 0, len(logs))
	for _, logEntry := range logs {
		summary := ActivityLogSummary{ActivityLog: logEntry}

		var user models.User
		if err := db.DB.First(&user, logEntry.UserID).Error; err == nil {
			summary.UserName = user.Name
		}

		result = append(result, summary)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "All activity logs retrieved successfully",
		"data":    result,
	})
}
