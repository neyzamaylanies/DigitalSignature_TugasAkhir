package utils

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MaxPDFSize int64 = 20 * 1024 * 1024 // 20 MB

func ValidatePDF(fileHeader *multipart.FileHeader) error {
	if fileHeader == nil {
		return errors.New("file is required")
	}

	if fileHeader.Size <= 0 {
		return errors.New("file is empty")
	}

	if fileHeader.Size > MaxPDFSize {
		return errors.New("file size exceeds 20MB limit")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".pdf" {
		return errors.New("only PDF files are allowed")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return errors.New("failed to open uploaded file")
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return errors.New("failed to read uploaded file")
	}

	contentType := http.DetectContentType(buffer[:n])
	if contentType != "application/pdf" {
		return errors.New("invalid PDF file content")
	}

	return nil
}

func GenerateFileName(prefix string, originalName string) string {
	ext := strings.ToLower(filepath.Ext(originalName))
	timestamp := time.Now().UnixNano()

	cleanPrefix := strings.ReplaceAll(prefix, " ", "_")
	cleanPrefix = strings.ToLower(cleanPrefix)

	return fmt.Sprintf("%s_%d%s", cleanPrefix, timestamp, ext)
}

func SaveUploadedFile(fileHeader *multipart.FileHeader, destinationPath string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	destinationDir := filepath.Dir(destinationPath)
	if err := os.MkdirAll(destinationDir, os.ModePerm); err != nil {
		return err
	}

	dst, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
