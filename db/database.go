package db

import (
	"fmt"
	"log"
	"os"

	"digital-signature-api/models"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		host,
		user,
		password,
		dbname,
		port,
		sslMode,
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect database: ", err)
	}

	DB = database

	err = DB.AutoMigrate(
		&models.User{},
		&models.Document{},
		&models.SignatureRequest{},
		&models.Certificate{},
		&models.SignatureImage{},
		&models.CertificateTransaction{},
		&models.ActivityLog{},
	)

	if err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	applyIntegrityConstraints()

	fmt.Println("Database connected successfully")
}

type foreignKeyConstraint struct {
	Name      string
	Table     string
	Column    string
	RefTable  string
	RefColumn string
}

func applyIntegrityConstraints() {
	ensureDokumenTipeNotNull()
	ensureUsersRoleDefault()
	ensureSignatureRequestUniqueIndexes()

	foreignKeys := []foreignKeyConstraint{
		{"fk_dokumen_user", "dokumen", "user_id", "users", "id"},
		{"fk_permintaan_ttd_dokumen", "permintaan_ttd", "dokumen_id", "dokumen", "id"},
		{"fk_permintaan_ttd_user", "permintaan_ttd", "user_id", "users", "id"},
		{"fk_sertifikat_user", "sertifikat", "user_id", "users", "id"},
		{"fk_tanda_tangan_user", "tanda_tangan", "user_id", "users", "id"},
		{"fk_transaksi_sertifikat_permintaan_ttd", "transaksi_sertifikat", "permintaan_ttd_id", "permintaan_ttd", "id"},
		{"fk_transaksi_sertifikat_dokumen", "transaksi_sertifikat", "dokumen_id", "dokumen", "id"},
		{"fk_transaksi_sertifikat_user", "transaksi_sertifikat", "user_id", "users", "id"},
		{"fk_transaksi_sertifikat_sertifikat", "transaksi_sertifikat", "sertifikat_id", "sertifikat", "id"},
		{"fk_log_aktivitas_dokumen", "log_aktivitas", "dokumen_id", "dokumen", "id"},
		{"fk_log_aktivitas_user", "log_aktivitas", "user_id", "users", "id"},
	}

	for _, fk := range foreignKeys {
		if constraintExists(fk.Name) {
			continue
		}

		stmt := fmt.Sprintf(
			"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
			fk.Table, fk.Name, fk.Column, fk.RefTable, fk.RefColumn,
		)

		if err := DB.Exec(stmt).Error; err != nil {
			log.Printf("failed to add constraint %s: %v", fk.Name, err)
			continue
		}

		log.Printf("constraint %s created (dokumen/permintaan_ttd integrity)", fk.Name)
	}
}

func constraintExists(name string) bool {
	var count int64
	DB.Raw(
		`SELECT COUNT(*) FROM pg_constraint WHERE conname = ?`,
		name,
	).Scan(&count)

	return count > 0
}

func ensureDokumenTipeNotNull() {
	var isNullable string
	err := DB.Raw(
		`SELECT is_nullable FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'dokumen' AND column_name = 'tipe'`,
	).Scan(&isNullable).Error

	if err != nil || isNullable != "YES" {
		return
	}

	if err := DB.Exec(`ALTER TABLE dokumen ALTER COLUMN tipe SET NOT NULL`).Error; err != nil {
		log.Printf("failed to set dokumen.tipe NOT NULL: %v", err)
		return
	}

	log.Println("dokumen.tipe set to NOT NULL")
}

// ensureSignatureRequestUniqueIndexes memastikan constraint UNIQUE pada kombinasi
// (dokumen_id, urutan) dan (dokumen_id, user_id) benar-benar ada di tabel
// permintaan_ttd. GORM AutoMigrate seharusnya sudah membuat kedua index ini dari
// tag `uniqueIndex:idx_dokumen_urutan` / `uniqueIndex:idx_dokumen_user` pada
// models.SignatureRequest, tapi statement ini dijalankan secara eksplisit sebagai
// jaring pengaman (idempotent, aman dijalankan berulang) supaya constraint ini
// terjamin ada di database walau AutoMigrate dilewati/gagal sebagian, sesuai
// yang dijelaskan pada BAB IV mengenai pencegahan duplikasi urutan/penanda tangan.
func ensureSignatureRequestUniqueIndexes() {
	if err := DB.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_dokumen_urutan ON permintaan_ttd (dokumen_id, urutan)`,
	).Error; err != nil {
		log.Printf("failed to ensure idx_dokumen_urutan unique index: %v", err)
	}

	if err := DB.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_dokumen_user ON permintaan_ttd (dokumen_id, user_id)`,
	).Error; err != nil {
		log.Printf("failed to ensure idx_dokumen_user unique index: %v", err)
	}
}

func ensureUsersRoleDefault() {
	var columnDefault *string
	err := DB.Raw(
		`SELECT column_default FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'role'`,
	).Scan(&columnDefault).Error

	if err != nil || (columnDefault != nil && *columnDefault != "") {
		return
	}

	if err := DB.Exec(`ALTER TABLE users ALTER COLUMN role SET DEFAULT 'user'`).Error; err != nil {
		log.Printf("failed to set users.role default: %v", err)
		return
	}

	log.Println("users.role default set to 'user'")
}
