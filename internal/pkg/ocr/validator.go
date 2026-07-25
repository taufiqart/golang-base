package ocr

import (
	"strings"
)

var supportedMIMEs = map[string]struct{}{
	"application/pdf": {},
	"image/jpeg":      {},
	"image/png":       {},
	"image/webp":      {},
}

// SupportedMIME checks if a given MIME type is supported for OCR processing.
func SupportedMIME(mimeType string) bool {
	_, ok := supportedMIMEs[strings.ToLower(strings.TrimSpace(mimeType))]
	return ok
}
