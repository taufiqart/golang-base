// Package export provides reusable document generation utilities.
//
// It exposes two sub-packages:
//
//   - export/excel: spreadsheet (XLSX) generation via excelize
//   - export/pdf:   PDF generation via gofpdf
//
// Both share the common types declared here (Column, Table, Meta) so a
// single dataset can be rendered to either format with consistent headers,
// ordering and formatting.
package export

// Column describes a single table column shared across exporters.
//
// Width semantics differ per exporter:
//   - Excel: width is measured in characters (roughly 7px each).
//   - PDF:   width is measured in millimeters. When zero, the column is
//     auto-sized to fit the available page width, proportional to ContentLen.
type Column struct {
	// Header is the human-readable column title.
	Header string
	// Key is the stable identifier used to look up a cell value in a Row.
	Key string
	// Width is the rendered width. See the package doc for units.
	Width float64
	// Align is the horizontal alignment: "left", "center" or "right".
	// Defaults to "left" when empty.
	Align string
	// ContentLen is a relative weight used for auto width distribution when
	// Width is zero. Defaults to 1.
	ContentLen float64
}

// Row maps a Column.Key to its cell value. Values may be string, numeric,
// bool, time.Time or nil; each exporter formats them appropriately.
type Row map[string]interface{}

// Table is a dataset plus its column layout. It is the primary input to both
// the Excel and PDF generators.
type Table struct {
	Columns []Column
	Rows    []Row
}

// Meta carries document-level metadata (title, subtitle, branding) shared by
// both exporters.
type Meta struct {
	// Title is rendered as the document heading.
	Title string
	// Subtitle is rendered beneath the title (optional).
	Subtitle string
	// Footer is rendered at the bottom of every PDF page (optional).
	Footer string
	// SheetName is the Excel worksheet name. Defaults to "Sheet1".
	SheetName string
	// Landscape requests landscape orientation. Only used by the PDF exporter.
	Landscape bool
}
