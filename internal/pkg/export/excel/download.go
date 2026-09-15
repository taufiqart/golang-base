package excel

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"golang-base/internal/pkg/export"
)

// MimeType is the MIME type for XLSX files.
const MimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// Download generates the workbook and writes it to the Fiber response as an
// attachment. When filename is empty a timestamped default is used.
func Download(c fiber.Ctx, meta export.Meta, table export.Table, opts Options, filename string) error {
	data, err := Generate(meta, table, opts)
	if err != nil {
		return err
	}
	if filename == "" {
		filename = fmt.Sprintf("export-%s.xlsx", time.Now().Format("20060102-150405"))
	}
	c.Set(fiber.HeaderContentType, MimeType)
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Send(data)
}
