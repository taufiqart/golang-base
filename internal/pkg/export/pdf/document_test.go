package pdf_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"golang-base/internal/pkg/export/exporttest"
	"golang-base/internal/pkg/export/pdf"
)

func TestDocumentCustomComposition(t *testing.T) {
	doc := pdf.NewDocument(
		pdf.WithPageSize("A4"),
		pdf.WithOrientation(false),
		pdf.WithMargins(15, 15, 15, 15),
	)

	require.NoError(t, doc.Header(pdf.Header{
		LogoBytes:  testLogoPNG(t),
		LogoFormat: "PNG",
		LogoWidth:  30,
		LeftLines:  []string{"Custom Co", "Custom Address"},
		RightLines: []string{"Invoice", "2024-01-01"},
		Divider:    true,
	}))

	doc.Title("Custom Report", "Built via Document")
	doc.Paragraph("This section is drawn by the service using the exposed Fpdf handle.")
	doc.Divider()
	doc.TableAuto(exporttest.SampleTable())
	doc.SetFooter("Custom footer", true)

	data, err := doc.Bytes()
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}

func TestDocumentDirectFpdfAccess(t *testing.T) {
	doc := pdf.NewDocument()

	// Draw raw content directly through the embedded *gofpdf.Fpdf.
	doc.SetFont("Helvetica", "B", 20)
	doc.SetTextColor(200, 0, 0)
	doc.CellFormat(doc.ContentWidth(), 12, "RAW ACCESS", "", 1, "C", false, 0, "")
	doc.Spacer(4)

	data, err := doc.Bytes()
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
	require.Greater(t, len(data), 500)
	require.Contains(t, string(data), "Helvetica-Bold")
}

func TestDocumentSave(t *testing.T) {
	doc := pdf.NewDocument()
	doc.Title("Saved", "")

	path := t.TempDir() + "/out/custom.pdf"
	require.NoError(t, doc.Save(path))
}
