package export_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"golang-base/internal/pkg/export"
)

type record struct {
	ID        int64     `json:"id" export:"ID"`
	Name      string    `json:"name"`
	Score     float64   `json:"score"`
	Hidden    string    `json:"-"`
	CreatedAt time.Time `json:"created_at" export:"Created"`
}

func TestFromStructsBuildsColumnsAndRows(t *testing.T) {
	items := []record{
		{ID: 1, Name: "Alice", Score: 9.5, Hidden: "x", CreatedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)},
		{ID: 2, Name: "Bob", Score: 7, Hidden: "y"},
	}

	table, err := export.FromStructs(items)
	require.NoError(t, err)
	require.Len(t, table.Columns, 4)

	require.Equal(t, "ID", table.Columns[0].Header)
	require.Equal(t, "right", table.Columns[0].Align)
	require.Equal(t, "Name", table.Columns[1].Header)
	require.Equal(t, "Score", table.Columns[2].Header)
	require.Equal(t, "Created", table.Columns[3].Header)

	require.Len(t, table.Rows, 2)
	require.Equal(t, int64(1), table.Rows[0]["ID"])
	require.Equal(t, "Alice", table.Rows[0]["Name"])
	require.Equal(t, "", export.FormatValue(table.Rows[1]["CreatedAt"]))
}

func TestFromStructsEmptySlice(t *testing.T) {
	table, err := export.FromStructs([]record{})
	require.NoError(t, err)
	require.Empty(t, table.Rows)
	require.Len(t, table.Columns, 4)
}

func TestFromStructsPointerSlice(t *testing.T) {
	items := []*record{{ID: 1, Name: "Alice"}}
	table, err := export.FromStructs(items)
	require.NoError(t, err)
	require.Len(t, table.Rows, 1)
	require.Equal(t, "Alice", table.Rows[0]["Name"])
}

func TestFromStructsRejectsNonSlice(t *testing.T) {
	_, err := export.FromStructs(record{})
	require.ErrorIs(t, err, export.ErrNotSlice)
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/nested/out.bin"
	require.NoError(t, export.WriteFile(path, []byte("data"), 0))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "data", string(data))
}
