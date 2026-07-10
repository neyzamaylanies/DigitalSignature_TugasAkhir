package utils

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

const MaxImageSize int64 = 2 * 1024 * 1024 // 2 MB

func ValidateSignatureImage(fileHeader *multipart.FileHeader) error {
	if fileHeader == nil {
		return errors.New("image file is required")
	}

	if fileHeader.Size <= 0 {
		return errors.New("image file is empty")
	}

	if fileHeader.Size > MaxImageSize {
		return errors.New("image size exceeds 2MB limit")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return errors.New("only PNG, JPG, or JPEG files are allowed")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return errors.New("failed to open uploaded image")
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return errors.New("failed to read uploaded image")
	}

	contentType := http.DetectContentType(buffer[:n])
	if contentType != "image/png" && contentType != "image/jpeg" {
		return errors.New("invalid image file content")
	}

	return nil
}
