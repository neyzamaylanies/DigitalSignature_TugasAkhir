package models

import "time"

type ActivityLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null" json:"user_id"`
	DokumenID  *uint     `json:"dokumen_id"`
	Aksi       string    `gorm:"type:varchar(100);not null" json:"aksi"`
	Keterangan string    `gorm:"type:text" json:"keterangan"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ActivityLog) TableName() string {
	return "log_aktivitas"
}
