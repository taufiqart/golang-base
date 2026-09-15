package export

import (
	"github.com/gofiber/fiber/v3"

	excelpkg "golang-base/internal/pkg/export/excel"
	pdfpkg "golang-base/internal/pkg/export/pdf"
)

// Handler serves the sample report as a downloadable file.
type Handler struct{}

// NewHandler creates the example handler.
func NewHandler() *Handler { return &Handler{} }

// RegisterRoutes wires the two download endpoints.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/export/applicants.xlsx", h.DownloadXLSX)
	router.Get("/export/applicants.pdf", h.DownloadPDF)
}

// DownloadXLSX renders the sample dataset as a spreadsheet attachment.
func (h *Handler) DownloadXLSX(c fiber.Ctx) error {
	table, err := buildTable()
	if err != nil {
		return err
	}

	opts := excelpkg.DefaultOptions()
	opts.ShowTitle = true

	return excelpkg.Download(c, sampleMeta(), table, opts, "applicants.xlsx")
}

// DownloadPDF renders the sample dataset as a PDF attachment with a text header.
func (h *Handler) DownloadPDF(c fiber.Ctx) error {
	table, err := buildTable()
	if err != nil {
		return err
	}

	opts := pdfpkg.DefaultOptions()
	opts.BrandHeader = pdfpkg.Header{
		LeftLines:  []string{"ACME Corp.", "123 Main St"},
		RightLines: []string{"Applicant Report"},
		Repeat:     true,
		Divider:    true,
	}

	return pdfpkg.Download(c, sampleMeta(), table, opts, "applicants.pdf")
}
