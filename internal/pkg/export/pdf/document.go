// Package pdf is a thin wrapper around gofpdf for building PDF documents.
//
// It offers two usage tiers:
//
//  1. One-shot: Generate / Download render an export.Table with sensible
//     defaults. Use this for the common case.
//
//  2. Full control: NewDocument returns a *Document exposing the underlying
//     *gofpdf.Fpdf plus table helpers. Service code can then draw anything it
//     wants (custom headers, extra sections, images, signatures) while still
//     reusing the table renderer.
//
// The underlying writer is reachable via Document.Fpdf so nothing is hidden.
package pdf

import (
	"bytes"
	"io"
	"time"

	"github.com/jung-kurt/gofpdf"

	"golang-base/internal/pkg/export"
)

// Document wraps a *gofpdf.Fpdf with table helpers. All gofpdf capabilities are
// available through the embedded Fpdf field.
type Document struct {
	*Fpdf
}

// Fpdf is an alias for gofpdf.Fpdf so callers can reference it without importing
// gofpdf directly. The underlying type is always *gofpdf.Fpdf.
type Fpdf = gofpdf.Fpdf

// DocumentOption configures a Document at construction time.
type DocumentOption func(*documentConfig)

type documentConfig struct {
	pageSize     string
	landscape    bool
	unitStr      string
	marginLeft   float64
	marginTop    float64
	marginRight  float64
	marginBottom float64
}

// WithPageSize sets the page size (e.g. "A4", "LETTER").
func WithPageSize(size string) DocumentOption {
	return func(c *documentConfig) { c.pageSize = size }
}

// WithOrientation sets landscape (true) or portrait (false).
func WithOrientation(landscape bool) DocumentOption {
	return func(c *documentConfig) { c.landscape = landscape }
}

// WithMargins overrides the default page margins in millimeters.
func WithMargins(left, top, right, bottom float64) DocumentOption {
	return func(c *documentConfig) {
		c.marginLeft = left
		c.marginTop = top
		c.marginRight = right
		c.marginBottom = bottom
	}
}

// NewDocument creates a PDF document with a single page ready to draw on.
//
// Fonts default to the standard Helvetica family. Callers add fonts via the
// embedded Fpdf (AddFont / AddUTF8Font) when custom typography is required.
func NewDocument(opts ...DocumentOption) *Document {
	cfg := documentConfig{
		pageSize:     "A4",
		marginLeft:   marginLeft,
		marginTop:    marginTop,
		marginRight:  marginRight,
		marginBottom: marginBottom,
	}
	for _, o := range opts {
		o(&cfg)
	}

	orientation := Portrait
	if cfg.landscape {
		orientation = Landscape
	}

	f := gofpdf.New(orientation, "mm", cfg.pageSize, "")
	f.SetMargins(cfg.marginLeft, cfg.marginTop, cfg.marginRight)
	f.SetAutoPageBreak(true, cfg.marginBottom+6)
	f.AddPage()
	f.SetFont("Helvetica", "", 9)

	return &Document{Fpdf: f}
}

// ContentWidth returns the usable width between the left and right margins.
func (d *Document) ContentWidth() float64 {
	w, _ := d.GetPageSize()
	return w - marginLeft - marginRight
}

// ContentLeft returns the left margin x-position.
func (d *Document) ContentLeft() float64 { return marginLeft }

// Table draws the given table at the current cursor position using opts for
// zebra striping, repeated headers and row height.
func (d *Document) Table(table export.Table, opts Options) {
	widths := resolveWidths(table.Columns, d.ContentWidth())
	drawHeader(d.Fpdf, table.Columns, widths)
	drawRows(d.Fpdf, table, widths, opts, opts.BrandHeader)
}

// TableAuto draws the table with the default options.
func (d *Document) TableAuto(table export.Table) {
	d.Table(table, DefaultOptions())
}

// Header draws a branded header block (logo + text lines) using the full content
// width at the current cursor position.
func (d *Document) Header(h Header) error {
	return drawBrandHeader(d.Fpdf, h, []float64{d.ContentWidth()})
}

// Title draws a simple text title + optional subtitle.
func (d *Document) Title(title, subtitle string) {
	drawTitleBlock(d.Fpdf, title, subtitle, d.ContentWidth())
}

// Paragraph writes wrapped body text at the current position and advances the
// cursor.
func (d *Document) Paragraph(text string) {
	d.SetFont("Helvetica", "", 9)
	d.SetTextColor(31, 41, 55)
	d.MultiCell(d.ContentWidth(), 5, text, "", "L", false)
}

// Spacer advances the cursor vertically by h millimeters.
func (d *Document) Spacer(h float64) {
	d.Ln(h)
}

// Divider draws a horizontal rule across the content width.
func (d *Document) Divider() {
	d.SetDrawColor(209, 213, 219)
	d.SetLineWidth(0.3)
	y := d.GetY()
	w, _ := d.GetPageSize()
	d.Line(marginLeft, y, w-marginRight, y)
	d.Ln(2)
}

// SetFooter registers a footer string and optional "Page X of Y" line drawn on
// every page.
func (d *Document) SetFooter(left string, pageNumbers bool) {
	total := d.PageNo()
	d.SetFooterFunc(func() {
		drawFooterText(d.Fpdf, left, pageNumbers, total)
	})
}

// SetPagelessFooter registers a callback invoked at the bottom of every page,
// giving full control over footer rendering.
func (d *Document) SetPagelessFooter(fn func(doc *Document, pageNo, total int)) {
	total := d.PageNo()
	d.SetFooterFunc(func() {
		fn(d, d.PageNo(), total)
	})
}

// Bytes finalizes the document and returns the raw PDF bytes.
func (d *Document) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := d.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Output streams the finished document to w.
func (d *Document) Output(w io.Writer) error {
	return d.Fpdf.Output(w)
}

// Save writes the finished document to a file path.
func (d *Document) Save(path string) error {
	data, err := d.Bytes()
	if err != nil {
		return err
	}
	return export.WriteFile(path, data, 0o644)
}

// MetaString formats t using the layout "2006-01-02 15:04". Helper for headers.
func MetaString(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}
