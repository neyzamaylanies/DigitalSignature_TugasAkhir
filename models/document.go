package models

import "time"

type Document struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"not null" json:"user_id"`
	Judul         string    `gorm:"type:varchar(255);not null" json:"judul"`
	Deskripsi     string    `gorm:"type:text" json:"deskripsi"`
	Pesan         string    `gorm:"type:text" json:"pesan"`
	Tipe          string    `gorm:"type:varchar(100);not null" json:"tipe"`
	Jenis         string    `gorm:"type:varchar(50);not null" json:"jenis"`
	FilePath      string    `gorm:"type:varchar(255);not null" json:"file_path"`
	FinalFilePath string    `gorm:"type:varchar(255)" json:"final_file_path"`
	Status        string    `gorm:"type:varchar(50);default:'draft'" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func (Document) TableName() string {
	return "dokumen"
}
