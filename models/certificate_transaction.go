package models

import "time"

type CertificateTransaction struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PermintaanTTDID *uint     `json:"permintaan_ttd_id"`
	DokumenID       uint      `gorm:"not null" json:"dokumen_id"`
	UserID          uint      `gorm:"not null" json:"user_id"`
	SertifikatID    *uint     `json:"sertifikat_id"`
	Aksi            string    `gorm:"type:varchar(50);not null" json:"aksi"`
	AlasanTolak     string    `gorm:"type:text" json:"alasan_tolak"`
	FileResultPath  string    `gorm:"type:varchar(255)" json:"file_result_path"`
	CreatedAt       time.Time `json:"created_at"`
}

func (CertificateTransaction) TableName() string {
	return "transaksi_sertifikat"
}
