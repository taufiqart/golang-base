package pdf_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"

	"golang-base/internal/pkg/export"
	"golang-base/internal/pkg/export/exporttest"
	"golang-base/internal/pkg/export/pdf"
)

func testLogoPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 60, 20))
	for x := 0; x < 60; x++ {
		for y := 0; y < 20; y++ {
			img.Set(x, y, color.RGBA{R: 31, G: 41, B: 55, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestPDFBrandHeaderWithLogoBytes(t *testing.T) {
	opts := pdf.DefaultOptions()
	opts.BrandHeader = pdf.Header{
		LogoBytes:  testLogoPNG(t),
		LogoFormat: "PNG",
		LogoWidth:  30,
		LeftLines:  []string{"ACME Corp.", "123 Main St"},
		RightLines: []string{"Applicant Report", "Printed: 2024-05-06"},
		Divider:    true,
	}

	data, err := pdf.Generate(exporttest.SampleMeta(), exporttest.SampleTable(), opts)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
	require.Contains(t, string(data), "/Image")
}

func TestPDFBrandHeaderRepeatsOnNewPage(t *testing.T) {
	table := exporttest.SampleTable()
	for i := 0; i < 200; i++ {
		table.Rows = append(table.Rows, export.Row{"id": 100 + i, "name": "X"})
	}

	opts := pdf.DefaultOptions()
	opts.BrandHeader = pdf.Header{
		LogoBytes:  testLogoPNG(t),
		LogoFormat: "PNG",
		LeftLines:  []string{"ACME Corp."},
		Repeat:     true,
	}

	data, err := pdf.Generate(exporttest.SampleMeta(), table, opts)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}

func TestPDFBrandHeaderTextOnly(t *testing.T) {
	opts := pdf.DefaultOptions()
	opts.BrandHeader = pdf.Header{
		LeftLines:  []string{"Company Name"},
		RightLines: []string{"Report"},
	}

	data, err := pdf.Generate(exporttest.SampleMeta(), exporttest.SampleTable(), opts)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}

func TestPDFBrandHeaderMissingLogoPath(t *testing.T) {
	opts := pdf.DefaultOptions()
	opts.BrandHeader = pdf.Header{
		LogoPath:  "/nonexistent/logo.png",
		LeftLines: []string{"Company"},
	}

	_, err := pdf.Generate(exporttest.SampleMeta(), exporttest.SampleTable(), opts)
	require.Error(t, err)
}
