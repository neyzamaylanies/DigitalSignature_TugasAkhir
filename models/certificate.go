package models

import "time"

type Certificate struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null" json:"user_id"`
	SerialNumber string    `gorm:"type:varchar(100);unique;not null" json:"serial_number"`
	PublicKey    string    `gorm:"type:text;not null" json:"public_key"`
	Status       string    `gorm:"type:varchar(50);default:'active'" json:"status"`
	ValidFrom    time.Time `json:"valid_from"`
	ValidUntil   time.Time `json:"valid_until"`
	CreatedAt    time.Time `json:"created_at"`
}

func (Certificate) TableName() string {
	return "sertifikat"
}
