// Package excel renders export.Table datasets into XLSX workbooks.
//
// It is a thin, opinionated wrapper over excelize that applies consistent
// styling (header band, borders, zebra striping, freeze panes) so callers only
// describe columns and rows.
package excel

import (
	"bytes"
	"strings"

	"github.com/xuri/excelize/v2"

	"golang-base/internal/pkg/export"
)

const (
	defaultSheet = "Sheet1"

	// Styling constants.
	headerRowHeight  = 22.0
	defaultRowHeight = 18.0
	minColWidth      = 10.0
	maxColWidth      = 60.0
	colWidthPadding  = 2.5
)

// Options tunes generation behaviour.
type Options struct {
	// AutoFilter enables an autofilter dropdown across the header row.
	AutoFilter bool
	// FreezeHeader freezes the header row so it stays visible while scrolling.
	FreezeHeader bool
	// Zebra applies alternating row background colours.
	Zebra bool
	// ShowTitle writes the document title into row 1, shifting data down.
	ShowTitle bool
}

// DefaultOptions returns a sensible set of formatting defaults.
func DefaultOptions() Options {
	return Options{
		AutoFilter:   true,
		FreezeHeader: true,
		Zebra:        true,
		ShowTitle:    false,
	}
}

// Generate builds an XLSX workbook from the table and returns it as bytes.
func Generate(meta export.Meta, table export.Table, opts Options) ([]byte, error) {
	if len(table.Columns) == 0 {
		return nil, ErrNoColumns
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := meta.SheetName
	if sheet == "" {
		sheet = defaultSheet
	}
	if sheet != defaultSheet {
		if err := f.SetSheetName(defaultSheet, sheet); err != nil {
			return nil, err
		}
	}

	styles, err := buildStyles(f)
	if err != nil {
		return nil, err
	}

	headerRow := 1
	if opts.ShowTitle && meta.Title != "" {
		if err := writeTitle(f, sheet, meta, styles); err != nil {
			return nil, err
		}
		headerRow = 3
	}

	writeHeader(f, sheet, table.Columns, headerRow, styles)
	writeRows(f, sheet, table, headerRow, opts, styles)
	applyColumns(f, sheet, table.Columns)

	if opts.FreezeHeader {
		if err := f.SetPanes(sheet, &excelize.Panes{
			Freeze:      true,
			YSplit:      headerRow,
			TopLeftCell: cellRef(0, headerRow+1),
			ActivePane:  "bottomLeft",
		}); err != nil {
			return nil, err
		}
	}

	if opts.AutoFilter {
		last := cellRef(len(table.Columns)-1, headerRow+len(table.Rows))
		first := cellRef(0, headerRow)
		if err := f.AutoFilter(sheet, first+":"+last, nil); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeTitle(f *excelize.File, sheet string, meta export.Meta, styles cellStyles) error {
	if err := f.SetCellValue(sheet, "A1", meta.Title); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A1", "A1", styles.titleRow); err != nil {
		return err
	}
	if meta.Subtitle != "" {
		if err := f.SetCellValue(sheet, "A2", meta.Subtitle); err != nil {
			return err
		}
	}
	return nil
}

func writeHeader(f *excelize.File, sheet string, cols []export.Column, row int, style cellStyles) {
	f.SetRowHeight(sheet, row, headerRowHeight)
	for i, col := range cols {
		cell := cellRef(i, row)
		f.SetCellValue(sheet, cell, col.Header)
		f.SetCellStyle(sheet, cell, cell, style.header)
	}
}

func writeRows(f *excelize.File, sheet string, table export.Table, headerRow int, opts Options, styles cellStyles) {
	for r, row := range table.Rows {
		excelRow := headerRow + 1 + r
		style := styles.body
		if opts.Zebra && r%2 == 1 {
			style = styles.bodyAlt
		}
		f.SetRowHeight(sheet, excelRow, defaultRowHeight)
		for c, col := range table.Columns {
			cell := cellRef(c, excelRow)
			f.SetCellValue(sheet, cell, export.FormatValue(row[col.Key]))
			f.SetCellStyle(sheet, cell, cell, style)
		}
	}
}

func applyColumns(f *excelize.File, sheet string, cols []export.Column) {
	widths := make([]float64, len(cols))
	for i, col := range cols {
		width := col.Width
		if width <= 0 {
			width = estimateWidth(col.Header, col)
		}
		if width < minColWidth {
			width = minColWidth
		}
		if width > maxColWidth {
			width = maxColWidth
		}
		widths[i] = width
	}
	for i, width := range widths {
		name := columnLetter(i)
		f.SetColWidth(sheet, name, name, width)
	}
}

func estimateWidth(header string, col export.Column) float64 {
	if col.ContentLen > 0 {
		return col.ContentLen*8 + colWidthPadding
	}
	return float64(len(header)) + colWidthPadding + 6
}

func cellRef(col, row int) string {
	return columnLetter(col) + itoa(row)
}

func columnLetter(col int) string {
	name, _ := excelize.ColumnNumberToName(col + 1)
	return name
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	var b strings.Builder
	var digits []byte
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		b.WriteByte(digits[i])
	}
	return b.String()
}
