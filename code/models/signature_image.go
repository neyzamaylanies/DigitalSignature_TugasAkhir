package models

import "time"

type SignatureImage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	Tipe      string    `gorm:"type:varchar(50);not null" json:"tipe"`
	FilePath  string    `gorm:"type:varchar(255);not null" json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
}

func (SignatureImage) TableName() string {
	return "tanda_tangan"
}
