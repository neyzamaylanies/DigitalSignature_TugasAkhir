package utils

import (
	"fmt"

	"digital-signature-api/db"
	"digital-signature-api/models"
)

func SeedUsers() {
	var count int64
	db.DB.Model(&models.User{}).Count(&count)

	if count > 0 {
		fmt.Println("Users already seeded")
		return
	}

	adminPassword, _ := HashPassword("admin123")
	userPassword, _ := HashPassword("user123")

	users := []models.User{
		{
			Name:     "Admin Demo",
			Email:    "admin@example.com",
			Password: adminPassword,
			Role:     "admin",
		},
		{
			Name:     "User Demo",
			Email:    "user@example.com",
			Password: userPassword,
			Role:     "user",
		},
	}

	db.DB.Create(&users)
	fmt.Println("Demo users seeded successfully")
}
