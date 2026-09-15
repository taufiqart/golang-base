// Package export is an opt-in example that demonstrates rendering a dataset to
// XLSX and PDF with internal/pkg/export, served over plain HTTP.
package export

import (
	"time"

	"golang-base/internal/pkg/export"
)

// Applicant is the sample dataset. The export tag supplies the column header;
// export:"-" keeps a field out of the report entirely.
type Applicant struct {
	ID        int64     `export:"ID"`
	Name      string    `export:"Name"`
	Email     string    `export:"Email"`
	Score     float64   `export:"Score"`
	Active    bool      `export:"Active"`
	CreatedAt time.Time `export:"Registered"`
	Notes     string    `export:"-"`
}

// sampleApplicants returns a deterministic dataset so downloaded files are
// stable and easy to eyeball.
func sampleApplicants() []Applicant {
	return []Applicant{
		{ID: 1, Name: "Alice", Email: "alice@example.com", Score: 91.5, Active: true, CreatedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)},
		{ID: 2, Name: "Bob", Email: "bob@example.com", Score: 78, Active: false, CreatedAt: time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com", Score: 84.25, Active: true, CreatedAt: time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)},
	}
}

// sampleMeta returns report metadata. The header only accepts left-aligned
// lines; logo is opt-in via Meta.LogoPath.
func sampleMeta() export.Meta {
	return export.Meta{
		Title:    "Applicant Report",
		Subtitle: "Generated from the export example",
		Footer:   "Confidential",
	}
}

// buildTable converts the sample slice into the shared export.Table shape.
func buildTable() (export.Table, error) {
	return export.FromStructs(sampleApplicants())
}
