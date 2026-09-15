# pkg/export

Reusable PDF & Excel (XLSX) generation helpers.

A single dataset (`export.Table`) can be rendered to PDF **or** Excel with consistent headers, ordering, and formatting. For PDF, the package also provides a `Document` that exposes the raw gofpdf handle, so service code can customize any layout it needs.

```
internal/pkg/export/
├── export.go     # Shared types: Column, Row, Table, Meta
├── format.go     # FormatValue(): cell value normalization
├── struct.go     # FromStructs(): []struct -> Table
├── file.go       # WriteFile(): write file + auto-create dirs
├── errors.go     # ErrNotSlice
├── excel/        # Render to XLSX (excelize)
│   ├── excel.go
│   ├── style.go
│   └── download.go
└── pdf/          # Render to PDF (gofpdf)
    ├── document.go    # Document (builder + raw handle)
    ├── pdf.go         # Generate / Download (one-shot defaults)
    ├── header.go      # Header (logo + text)
    ├── brandheader.go
    └── download.go
```

---

## Quick Start

### Data -> Table

```go
type Applicant struct {
    ID     int64   `export:"ID"`
    Name   string  `export:"Name"`
    Score  float64 `export:"Score"`
    Secret string  `export:"-"`      // skipped
}

table, err := export.FromStructs(applicants)
```

`FromStructs` rules:
- Column header comes from the `export:"..."` tag; otherwise the Go field name.
- Fields tagged `export:"-"` or `json:"-"` are skipped; unexported fields are ignored.
- Numeric fields default to right alignment.
- Accepts a slice of struct or *struct.

You can also build a `Table` manually for full control:

```go
table := export.Table{
    Columns: []export.Column{
        {Header: "ID", Key: "id", Width: 14, Align: "center"},
        {Header: "Name", Key: "name"},          // Width 0 -> auto
        {Header: "Score", Key: "score", Align: "right", ContentLen: 2},
    },
    Rows: []export.Row{
        {"id": 1, "name": "Alice", "score": 9.5},
    },
}
```

### Excel

```go
meta := export.Meta{Title: "Applicant Report", SheetName: "Applicants"}

opts := excel.DefaultOptions()
opts.ShowTitle = true

// to bytes
xlsx, err := excel.Generate(meta, table, opts)
export.WriteFile("out.xlsx", xlsx, 0o644)

// or download directly via Fiber
return excel.Download(c, meta, table, opts, "applicants.xlsx")
```

`excel.Options`: `AutoFilter`, `FreezeHeader`, `Zebra`, `ShowTitle`.
In Excel, `Width` is measured in **characters**; `Width` 0 -> auto (derived from `ContentLen`).

### PDF - default format

```go
meta := export.Meta{
    Title:     "Applicant Report",
    Subtitle:  "Generated 2024-05-06",
    Footer:    "Confidential",
    Landscape: false,
}

data, err := pdf.Generate(meta, table, pdf.DefaultOptions())
return pdf.Download(c, meta, table, pdf.DefaultOptions(), "applicants.pdf")
```

`pdf.Options`: `PageSize`, `Zebra`, `RepeatHeader`, `ShowPageNumbers`, `MinRowHeight`, `BrandHeader`.
In PDF, `Width` is measured in **millimeters**; `Width` 0 -> auto, proportional to `ContentLen`.
The table header repeats automatically on page breaks (`RepeatHeader`).

### PDF - brand header (logo + text)

```go
opts := pdf.DefaultOptions()
opts.BrandHeader = pdf.Header{
    LogoBytes:  logoPNG,          // or LogoPath / LogoReader
    LogoFormat: "PNG",            // required for bytes/reader
    LogoWidth:  34,               // mm; height follows the aspect ratio
    LeftLines:  []string{"ACME Corp.", "123 Main St"},
    RightLines: []string{"Applicant Report", "Printed: 2024-05-06"},
    Repeat:     true,             // redraw on every page
    Divider:    true,             // rule below the header
}
pdf.Generate(meta, table, opts)
```

If `BrandHeader` is left empty (zero value), the PDF uses the plain `meta.Title`/`meta.Subtitle` block instead.

### PDF - fully custom from the service

`pdf.NewDocument()` returns a `*pdf.Document` that embeds `*gofpdf.Fpdf`. Every gofpdf capability is immediately available (`doc.SetFont`, `doc.CellFormat`, `doc.ImageOptions`, `doc.TransformRotate`, etc.), while the package helpers for tables/headers/footers remain usable.

```go
doc := pdf.NewDocument(
    pdf.WithPageSize("A4"),
    pdf.WithOrientation(false),
    pdf.WithMargins(15, 15, 15, 18),
)

// Branded header (package helper)
doc.Header(pdf.Header{
    LogoBytes: logoPNG, LogoFormat: "PNG", LogoWidth: 30,
    LeftLines:  []string{"ACME Corp.", "Jakarta"},
    RightLines: []string{"INVOICE", "#INV-2026-0001"},
    Divider:    true,
})

// Manual block through the raw gofpdf handle
doc.SetFont("Helvetica", "B", 9)
doc.CellFormat(60, 6, "Bill To:", "", 1, "L", false, 0, "")
doc.MultiCell(doc.ContentWidth(), 5, "Acme Corp\n45 Sudirman St.", "", "L", false)

// Table (package helper)
doc.TableAuto(table)

// Footer + page numbers
doc.SetFooter("Confidential", true)

out, _ := doc.Bytes()          // or doc.Save("invoice.pdf") / doc.Output(w)
```

**Available `Document` helpers:**

| Method | Purpose |
|--------|---------|
| `Header(Header)` | Branded header (logo + text) |
| `Title(title, subtitle)` | Title + subtitle |
| `Table(table, Options)` / `TableAuto(table)` | Render a table |
| `Paragraph(text)` | Auto-wrapped body text |
| `Divider()` | Horizontal rule |
| `Spacer(mm)` | Advance the cursor vertically |
| `SetFooter(left, pageNumbers)` | Simple footer + "Page X of Y" |
| `SetPagelessFooter(fn)` | Fully custom per-page footer |
| `ContentWidth() / ContentLeft()` | Content area size |
| `Bytes() / Save(path) / Output(w)` | Final output |

**gofpdf capabilities available directly via `doc.Fpdf`** (examples, not an exhaustive list):
font & color (`SetFont`, `SetTextColor`, `SetFillColor`), shapes (`Rect`, `Circle`, `Ellipse`, `Line`, `Polygon`), images (`ImageOptions`, `RegisterImage`), transforms (`TransformRotate`, `TransformScale`), layers/watermarks (`AddLayer`, `SetAlpha`), templates (`CreateTemplate`, `UseTemplate`), links (`AddLink`, `Link`), metadata (`SetTitle`, `SetAuthor`), protection (`SetProtection`), and `HTMLBasicNew()`.

---

## Quick Reference

```go
// Cell values accept: string, bool, int/uint/float (all widths), time.Time, []byte, nil.
// FormatValue(v) normalizes them for both exporters.

export.FromStructs([]T) (Table, error)   // ErrNotSlice when not a slice
export.WriteFile(path, data, perm) error // auto-creates parent dirs

excel.Generate(meta, table, opts) ([]byte, error)
excel.Download(c, meta, table, opts, filename) error
pdf.Generate(meta, table, opts) ([]byte, error)
pdf.Download(c, meta, table, opts, filename) error
```

Sentinel errors:
- `export.ErrNotSlice` - `FromStructs` input is not a slice.
- `excel.ErrNoColumns` / `pdf.ErrNoColumns` - empty `Table.Columns`.

---

## Notes

- Cell values accept common query/JSON types; `nil` renders as an empty string.
- For Excel, `SheetName` defaults to `"Sheet1"`.
- `Meta.Landscape` only affects the PDF exporter.
- Neither exporter performs I/O except reading the logo (`LogoPath`) - all results are returned as `[]byte`.
