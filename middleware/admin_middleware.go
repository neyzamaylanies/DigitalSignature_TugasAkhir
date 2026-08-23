package middleware

import (
	"encoding/json"
	"net/http"
)

func AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value("role")

		if role != "admin" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Access denied. Admin only.",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
