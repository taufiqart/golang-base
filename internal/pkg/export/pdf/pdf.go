package pdf

import (
	"bytes"
	"strings"

	"github.com/jung-kurt/gofpdf"

	"golang-base/internal/pkg/export"
)

// Orientation constants.
const (
	Portrait  = "P"
	Landscape = "L"
)

// Options tunes PDF generation behaviour.
type Options struct {
	// PageSize is a gofpdf page size (e.g. "A4", "LETTER"). Defaults to "A4".
	PageSize string
	// Zebra applies alternating row background colours.
	Zebra bool
	// RepeatHeader re-draws the table header row on every page.
	RepeatHeader bool
	// ShowPageNumbers prints "Page X of Y" in the footer.
	ShowPageNumbers bool
	// MinRowHeight is the minimum row height in millimeters.
	MinRowHeight float64
	// BrandHeader draws a custom branded header (logo + text) above the table.
	// When zero-valued the plain meta.Title/Subtitle block is used instead.
	BrandHeader Header
}

// DefaultOptions returns a sensible set of formatting defaults.
func DefaultOptions() Options {
	return Options{
		PageSize:        "A4",
		Zebra:           true,
		RepeatHeader:    true,
		ShowPageNumbers: true,
		MinRowHeight:    8,
	}
}

// Generate builds a PDF document from the table and returns it as bytes.
func Generate(meta export.Meta, table export.Table, opts Options) ([]byte, error) {
	if len(table.Columns) == 0 {
		return nil, ErrNoColumns
	}

	size := opts.PageSize
	if size == "" {
		size = "A4"
	}
	orientation := Portrait
	if meta.Landscape {
		orientation = Landscape
	}

	p := gofpdf.New(orientation, "mm", size, "")
	p.SetMargins(marginLeft, marginTop, marginRight)
	p.SetAutoPageBreak(true, marginBottom+6)
	p.AddPage()
	setupFonts(p)

	widths := resolveWidths(table.Columns, pageWidth(p)-marginLeft-marginRight)

	if hasBrandHeader(opts.BrandHeader) {
		if err := drawBrandHeader(p, opts.BrandHeader, widths); err != nil {
			return nil, err
		}
	} else {
		drawDocumentTitle(p, meta, table.Columns, widths)
	}

	drawHeader(p, table.Columns, widths)
	drawRows(p, table, widths, opts, opts.BrandHeader)

	// Register footer drawing for every page (goes through the whole doc).
	total := p.PageNo()
	p.SetFooterFunc(func() {
		drawFooter(p, meta, opts, total)
	})

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func resolveWidths(cols []export.Column, available float64) []float64 {
	widths := make([]float64, len(cols))
	total := 0.0
	weightTotal := 0.0
	for i, col := range cols {
		if col.Width > 0 {
			widths[i] = col.Width
			total += col.Width
			continue
		}
		w := col.ContentLen
		if w <= 0 {
			w = 1
		}
		weightTotal += w
	}
	remaining := available - total
	if remaining < 0 {
		remaining = available
	}
	for i, col := range cols {
		if col.Width > 0 {
			continue
		}
		w := col.ContentLen
		if w <= 0 {
			w = 1
		}
		widths[i] = remaining * (w / weightTotal)
	}
	return widths
}

func drawDocumentTitle(p *gofpdf.Fpdf, meta export.Meta, cols []export.Column, widths []float64) {
	drawTitleBlock(p, meta.Title, meta.Subtitle, tableWidth(widths))
}

func drawTitleBlock(p *gofpdf.Fpdf, title, subtitle string, width float64) {
	if title == "" {
		return
	}
	p.SetFont("Helvetica", "B", 16)
	p.SetTextColor(17, 24, 39)
	p.CellFormat(width, 10, title, "", 1, "L", false, 0, "")
	if subtitle != "" {
		p.SetFont("Helvetica", "", 10)
		p.SetTextColor(107, 114, 128)
		p.CellFormat(width, 6, subtitle, "", 1, "L", false, 0, "")
	}
	p.Ln(2)
}

func drawHeader(p *gofpdf.Fpdf, cols []export.Column, widths []float64) {
	p.SetFont("Helvetica", "B", 9)
	p.SetFillColor(31, 41, 55)
	p.SetTextColor(255, 255, 255)
	for i, col := range cols {
		p.CellFormat(widths[i], 9, col.Header, "1", 0, alignFor(col), true, 0, "")
	}
	p.Ln(-1)
}

func drawRows(p *gofpdf.Fpdf, table export.Table, widths []float64, opts Options, brand Header) {
	p.SetFont("Helvetica", "", 8)
	for r, row := range table.Rows {
		if p.GetY()+opts.MinRowHeight > pageHeight(p)-marginBottom-8 {
			p.AddPage()
			if hasBrandHeader(brand) && brand.Repeat {
				_ = drawBrandHeader(p, brand, widths)
			}
			if opts.RepeatHeader {
				drawHeader(p, table.Columns, widths)
			}
		}
		fill := opts.Zebra && r%2 == 1
		if fill {
			p.SetFillColor(243, 244, 246)
		}
		p.SetTextColor(17, 24, 39)
		height := opts.MinRowHeight
		for i, col := range table.Columns {
			text := export.FormatValue(row[col.Key])
			p.CellFormat(widths[i], height, text, "1", 0, alignFor(col), fill, 0, "")
		}
		p.Ln(-1)
	}
}

func drawFooter(p *gofpdf.Fpdf, meta export.Meta, opts Options, total int) {
	drawFooterText(p, meta.Footer, opts.ShowPageNumbers, total)
}

func drawFooterText(p *gofpdf.Fpdf, left string, pageNumbers bool, total int) {
	p.SetFont("Helvetica", "", 8)
	p.SetTextColor(156, 163, 175)
	width := pageWidth(p) - marginLeft - marginRight
	if left != "" {
		p.CellFormat(width/2, 6, left, "", 0, "L", false, 0, "")
	}
	if pageNumbers && total > 0 {
		text := "Page " + itoa(p.PageNo()) + " of " + itoa(total)
		p.CellFormat(width/2, 6, text, "", 1, "R", false, 0, "")
	}
}

func alignFor(col export.Column) string {
	switch strings.ToLower(col.Align) {
	case "center":
		return "C"
	case "right":
		return "R"
	default:
		return "L"
	}
}

func tableWidth(widths []float64) float64 {
	var total float64
	for _, w := range widths {
		total += w
	}
	return total
}

// Layout margins in millimeters.
const (
	marginLeft   = 12.0
	marginTop    = 12.0
	marginRight  = 12.0
	marginBottom = 12.0
)

func setupFonts(p *gofpdf.Fpdf) {
	p.SetFont("Helvetica", "", 9)
}

func pageWidth(p *gofpdf.Fpdf) float64 {
	w, _ := p.GetPageSize()
	return w
}

func pageHeight(p *gofpdf.Fpdf) float64 {
	_, h := p.GetPageSize()
	return h
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	if neg {
		digits = append(digits, '-')
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
