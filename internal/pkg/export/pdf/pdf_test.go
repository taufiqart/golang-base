package pdf_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"golang-base/internal/pkg/export"
	"golang-base/internal/pkg/export/exporttest"
	"golang-base/internal/pkg/export/pdf"
)

func TestPDFGenerateProducesPDF(t *testing.T) {
	data, err := pdf.Generate(exporttest.SampleMeta(), exporttest.SampleTable(), pdf.DefaultOptions())
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// PDF files start with the %PDF- magic header.
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")), "expected PDF magic header")
	require.True(t, bytes.Contains(data, []byte("%%EOF")), "expected PDF EOF marker")
	require.Greater(t, len(data), 1000, "expected non-trivial PDF output")
}

func TestPDFLandscapeOrientation(t *testing.T) {
	meta := exporttest.SampleMeta()
	meta.Landscape = true
	data, err := pdf.Generate(meta, exporttest.SampleTable(), pdf.DefaultOptions())
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}

func TestPDFNoColumns(t *testing.T) {
	_, err := pdf.Generate(export.Meta{}, export.Table{}, pdf.DefaultOptions())
	require.ErrorIs(t, err, pdf.ErrNoColumns)
}

func TestPDFManyRowsSpanPages(t *testing.T) {
	table := export.Table{
		Columns: []export.Column{
			{Header: "No", Key: "no", Width: 20},
			{Header: "Value", Key: "value"},
		},
		Rows: make([]export.Row, 0, 200),
	}
	for i := 0; i < 200; i++ {
		table.Rows = append(table.Rows, export.Row{"no": i + 1, "value": strings.Repeat("x", 20)})
	}

	data, err := pdf.Generate(exporttest.SampleMeta(), table, pdf.DefaultOptions())
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}
