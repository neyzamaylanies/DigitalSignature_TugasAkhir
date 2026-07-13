package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"digital-signature-api/db"
	"digital-signature-api/routes"
	"digital-signature-api/utils"
)

func main() {
	db.ConnectDatabase()
	utils.SeedUsers()

	routes.SetupRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "8080"
	}

	fmt.Println("Server running on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
