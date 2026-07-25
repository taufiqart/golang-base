package ocr

import "testing"

func TestSupportedMIMEAcceptsOnlyOCRFiles(t *testing.T) {
	if !SupportedMIME("image/jpeg") {
		t.Fatal("expected jpeg to be supported")
	}
	if !SupportedMIME("image/png") {
		t.Fatal("expected png to be supported")
	}
	if !SupportedMIME("application/pdf") {
		t.Fatal("expected pdf to be supported")
	}
	if SupportedMIME("text/plain") {
		t.Fatal("expected text/plain to be rejected")
	}
}
