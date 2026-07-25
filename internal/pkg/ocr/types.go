package ocr

import "errors"

var (
	ErrUnavailable     = errors.New("ocr service unavailable")
	ErrFailed          = errors.New("ocr failed")
	ErrUnsupportedFile = errors.New("file type is not supported for OCR")
	ErrFileTooLarge    = errors.New("file size exceeds maximum allowed size")
)

// Input represents the file or image payload submitted for OCR extraction.
type Input struct {
	FileName string
	MimeType string
	Data     []byte
	Pages    [][]byte
}

// ExtractionRequest configures the extraction task for the AI client.
type ExtractionRequest struct {
	Input      Input
	SchemaName string
	Schema     map[string]any
	Prompt     string
}
