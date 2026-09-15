package excel

import (
	"errors"

	"github.com/xuri/excelize/v2"
)

// Errors returned by the excel package.
var (
	// ErrNoColumns is returned when a table has no columns defined.
	ErrNoColumns = errors.New("excel: table has no columns")
)

// cellStyles holds the style IDs used across the worksheet.
type cellStyles struct {
	header   int
	body     int
	bodyAlt  int
	titleRow int
}

type colorRGB string

const (
	colorHeaderBG  colorRGB = "1F2937" // slate-800
	colorHeaderFG  colorRGB = "FFFFFF"
	colorAltBG     colorRGB = "F3F4F6" // gray-100
	colorBorder    colorRGB = "D1D5DB" // gray-300
	colorTitleText colorRGB = "111827" // gray-900
)

func buildStyles(f *excelize.File) (cellStyles, error) {
	var s cellStyles

	header, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: string(colorHeaderFG), Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{string(colorHeaderBG)}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    thinBorders(),
	})
	if err != nil {
		return s, err
	}
	s.header = header

	body, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: false},
		Border:    thinBorders(),
	})
	if err != nil {
		return s, err
	}
	s.body = body

	bodyAlt, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{string(colorAltBG)}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorders(),
	})
	if err != nil {
		return s, err
	}
	s.bodyAlt = bodyAlt

	title, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: string(colorTitleText)},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return s, err
	}
	s.titleRow = title

	return s, nil
}

func thinBorders() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: string(colorBorder), Style: 1},
		{Type: "right", Color: string(colorBorder), Style: 1},
		{Type: "top", Color: string(colorBorder), Style: 1},
		{Type: "bottom", Color: string(colorBorder), Style: 1},
	}
}
