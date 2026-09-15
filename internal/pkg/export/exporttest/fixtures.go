// Package exporttest provides shared fixtures for export package tests.
package exporttest

import (
	"time"

	"golang-base/internal/pkg/export"
)

// SampleTable returns a representative dataset covering strings, numbers,
// booleans, times and nil values.
func SampleTable() export.Table {
	return export.Table{
		Columns: []export.Column{
			{Header: "ID", Key: "id", Width: 14, Align: "center"},
			{Header: "Name", Key: "name"},
			{Header: "Email", Key: "email"},
			{Header: "Active", Key: "active", Align: "center"},
			{Header: "Created At", Key: "created_at"},
		},
		Rows: []export.Row{
			{"id": 1, "name": "Alice", "email": "alice@example.com", "active": true, "created_at": time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)},
			{"id": 2, "name": "Bob", "email": "bob@example.com", "active": false, "created_at": time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)},
			{"id": 3, "name": "Charlie", "email": nil, "active": true, "created_at": nil},
		},
	}
}

// SampleMeta returns representative document metadata.
func SampleMeta() export.Meta {
	return export.Meta{
		Title:    "User Report",
		Subtitle: "Generated for testing",
		Footer:   "Confidential",
	}
}
