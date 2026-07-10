package models

import "time"

type SignatureRequest struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DokumenID   uint      `gorm:"not null" json:"dokumen_id"`
	UserID      uint      `gorm:"not null" json:"user_id"`
	Urutan      int       `gorm:"not null" json:"urutan"`
	PageNumber  int       `gorm:"not null" json:"page_number"`
	Width       float64   `json:"width"`
	Height      float64   `json:"height"`
	Status      string    `gorm:"type:varchar(50);default:'menunggu'" json:"status"`
	AlasanTolak string    `gorm:"type:text" json:"alasan_tolak"`
	CreatedAt   time.Time `json:"created_at"`
}

func (SignatureRequest) TableName() string {
	return "permintaan_ttd"
}
