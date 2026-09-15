package excel_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"golang-base/internal/pkg/export"
	"golang-base/internal/pkg/export/excel"
	"golang-base/internal/pkg/export/exporttest"
)

func TestExcelGenerateProducesValidWorkbook(t *testing.T) {
	data, err := excel.Generate(exporttest.SampleMeta(), exporttest.SampleTable(), excel.DefaultOptions())
	require.NoError(t, err)
	require.NotEmpty(t, data)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	require.NoError(t, err)
	require.Len(t, rows, 4)

	require.Equal(t, []string{"ID", "Name", "Email", "Active", "Created At"}, rows[0])
	require.Equal(t, []string{"1", "Alice", "alice@example.com", "true", "2024-01-02 03:04:05"}, rows[1])
	require.Equal(t, "3", rows[3][0])
}

func TestExcelFormatValueCases(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"string", "hello", "hello"},
		{"true", true, "true"},
		{"false", false, "false"},
		{"nil", nil, ""},
		{"int", 42, "42"},
		{"int64", int64(99), "99"},
		{"uint", uint(7), "7"},
		{"float whole", 10.0, "10"},
		{"float frac", 3.5, "3.5"},
		{"bytes", []byte("raw"), "raw"},
		{"time", time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC), "2024-05-06 07:08:09"},
		{"zero time", time.Time{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, export.FormatValue(tc.in))
		})
	}
}

func TestExcelNoColumns(t *testing.T) {
	_, err := excel.Generate(export.Meta{}, export.Table{}, excel.DefaultOptions())
	require.ErrorIs(t, err, excel.ErrNoColumns)
}

func TestExcelCustomSheetName(t *testing.T) {
	meta := exporttest.SampleMeta()
	meta.SheetName = "Users"
	data, err := excel.Generate(meta, exporttest.SampleTable(), excel.DefaultOptions())
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	require.Equal(t, "Users", f.GetSheetName(0))
}

func TestExcelShowTitle(t *testing.T) {
	opts := excel.DefaultOptions()
	opts.ShowTitle = true

	data, err := excel.Generate(exporttest.SampleMeta(), exporttest.SampleTable(), opts)
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	sheet := f.GetSheetName(0)
	require.Equal(t, "User Report", mustCell(t, f, sheet, "A1"))
	require.Equal(t, "Generated for testing", mustCell(t, f, sheet, "A2"))
	require.Equal(t, "ID", mustCell(t, f, sheet, "A3"))
	require.Equal(t, "Alice", mustCell(t, f, sheet, "B4"))
}

func mustCell(t *testing.T, f *excelize.File, sheet, cell string) string {
	v, err := f.GetCellValue(sheet, cell)
	require.NoError(t, err)
	return v
}
