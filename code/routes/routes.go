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

	// Auth routes
	http.HandleFunc("/api/auth/login", handlers.Login)
	http.HandleFunc("/api/auth/logout", middleware.AuthMiddleware(handlers.Logout))
	http.HandleFunc("/api/auth/profile", middleware.AuthMiddleware(handlers.Profile))

	// User routes
	http.HandleFunc("/api/users", middleware.AdminMiddleware(handlers.GetUsers))

	// Dashboard routes
	http.HandleFunc("/api/dashboard", middleware.AuthMiddleware(handlers.GetDashboard))

	// Document routes
	http.HandleFunc("/api/dokumen/upload", middleware.AuthMiddleware(handlers.UploadDocument))
	http.HandleFunc("/api/dokumen", middleware.AuthMiddleware(handlers.GetMyDocuments))
	http.HandleFunc("/api/dokumen/", middleware.AuthMiddleware(handlers.DocumentRouter))

	// Signature image routes
	http.HandleFunc("/api/tanda-tangan", middleware.AuthMiddleware(handlers.SignatureImageRootRouter))
	http.HandleFunc("/api/tanda-tangan/", middleware.AuthMiddleware(handlers.SignatureImageDetailRouter))

	// Certificate routes
	http.HandleFunc("/api/sertifikat", middleware.AuthMiddleware(handlers.CertificateRootRouter))
	http.HandleFunc("/api/sertifikat/", middleware.AuthMiddleware(handlers.CertificateDetailRouter))
}
