package utils

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func getImageDimensions(imagePath string) (widthPx, heightPx float64, err error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return float64(config.Width), float64(config.Height), nil
}

func StampSignatureOnPDF(inputPath, outputPath, imagePath string, pageNumber int, koordinatXPercent, koordinatYPercent, widthPercent, heightPercent float64) error {
	dims, err := api.PageDimsFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read PDF page dimensions: %w", err)
	}

	if pageNumber < 1 || pageNumber > len(dims) {
		return fmt.Errorf("page %d does not exist in this document", pageNumber)
	}

	pageWidthPt := dims[pageNumber-1].Width
	pageHeightPt := dims[pageNumber-1].Height

	imgWidthPx, imgHeightPx, err := getImageDimensions(imagePath)
	if err != nil {
		return fmt.Errorf("failed to read signature image dimensions: %w", err)
	}
	aspectRatio := imgWidthPx / imgHeightPx

	boxWidthPt := (widthPercent / 100) * pageWidthPt
	boxHeightPt := (heightPercent / 100) * pageHeightPt

	var actualWidthPt, actualHeightPt float64

	if boxWidthPt/aspectRatio <= boxHeightPt {
		actualWidthPt = boxWidthPt
		actualHeightPt = actualWidthPt / aspectRatio
	} else {
		actualHeightPt = boxHeightPt
		actualWidthPt = actualHeightPt * aspectRatio
	}

	offsetXPt := (koordinatXPercent / 100) * pageWidthPt
	offsetYPt := pageHeightPt - (koordinatYPercent/100)*pageHeightPt - actualHeightPt

	scaleFactor := actualWidthPt / imgWidthPx

	desc := fmt.Sprintf(
		"pos:bl, offset:%.2f %.2f, scale:%.4f abs, rotation:0",
		offsetXPt, offsetYPt, scaleFactor,
	)

	conf := model.NewDefaultConfiguration()

	return api.AddImageWatermarksFile(
		inputPath,
		outputPath,
		[]string{strconv.Itoa(pageNumber)},
		true,
		imagePath,
		desc,
		conf,
	)
}
