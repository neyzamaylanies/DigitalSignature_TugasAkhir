package routes

import (
	"net/http"

	"digital-signature-api/handlers"
	"digital-signature-api/middleware"
)

func SetupRoutes() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Digital Signature API is running"))
	})

	http.HandleFunc("/api/auth/login", handlers.Login)
	http.HandleFunc("/api/auth/logout", middleware.AuthMiddleware(handlers.Logout))
	http.HandleFunc("/api/auth/profile", middleware.AuthMiddleware(handlers.Profile))
}
