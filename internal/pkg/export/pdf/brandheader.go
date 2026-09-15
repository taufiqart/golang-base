package pdf

import (
	"bytes"
	"fmt"
	"image"
	"os"

	"github.com/jung-kurt/gofpdf"
)

func hasBrandHeader(h Header) bool {
	return h.LogoBytes != nil || h.LogoPath != "" || h.LogoReader != nil ||
		len(h.LeftLines) > 0 || len(h.RightLines) > 0
}

func drawBrandHeader(p *gofpdf.Fpdf, h Header, widths []float64) error {
	imgName, err := registerLogo(p, h)
	if err != nil {
		return err
	}

	width := tableWidth(widths)
	top := p.GetY()

	const logoH = 14.0
	const lineH = 5.0
	var logoW float64

	if imgName != "" {
		nw, nh, err := imageSize(h)
		if err == nil && nh > 0 {
			logoW = logoH * (float64(nw) / float64(nh))
		} else {
			logoW = logoH
		}
		if h.LogoWidth > 0 {
			logoW = h.LogoWidth
		}
		p.ImageOptions(imgName, marginLeft, top, logoW, logoH, false, imageOpts(h.LogoFormat), 0, "")
	}

	textLeft := marginLeft
	if imgName != "" {
		textLeft = marginLeft + logoW + 4
	}
	textWidth := width - (textLeft - marginLeft) - rightTextWidth(h)

	renderLines(p, h.LeftLines, textLeft, top, textWidth, "L")
	renderRight(p, h.RightLines, marginLeft+width, top, "R")

	blockHeight := logoH
	if lines := maxLines(h); float64(lines)*lineH > blockHeight {
		blockHeight = float64(lines) * lineH
	}
	p.SetY(top + blockHeight + 2)

	if h.Divider {
		p.SetDrawColor(209, 213, 219)
		p.SetLineWidth(0.3)
		y := p.GetY()
		p.Line(marginLeft, y, marginLeft+width, y)
		p.Ln(2)
	}

	return nil
}

func renderLines(p *gofpdf.Fpdf, lines []string, x, y, width float64, align string) {
	for i, line := range lines {
		if i == 0 {
			p.SetFont("Helvetica", "B", 11)
			p.SetTextColor(17, 24, 39)
		} else {
			p.SetFont("Helvetica", "", 8)
			p.SetTextColor(107, 114, 128)
		}
		lineY := y + float64(i)*5
		p.SetXY(x, lineY)
		p.CellFormat(width, 5, line, "", 2, align, false, 0, "")
	}
}

func renderRight(p *gofpdf.Fpdf, lines []string, right, y float64, align string) {
	for i, line := range lines {
		if i == 0 {
			p.SetFont("Helvetica", "B", 9)
			p.SetTextColor(17, 24, 39)
		} else {
			p.SetFont("Helvetica", "", 8)
			p.SetTextColor(107, 114, 128)
		}
		lineY := y + float64(i)*5
		p.SetXY(right-60, lineY)
		p.CellFormat(60, 5, line, "", 2, align, false, 0, "")
	}
}

func rightTextWidth(h Header) float64 {
	if len(h.RightLines) == 0 {
		return 0
	}
	return 62
}

func maxLines(h Header) int {
	if len(h.LeftLines) > len(h.RightLines) {
		return len(h.LeftLines)
	}
	return len(h.RightLines)
}

func registerLogo(p *gofpdf.Fpdf, h Header) (string, error) {
	switch {
	case h.LogoReader != nil:
		opts := imageOpts(h.LogoFormat)
		p.RegisterImageOptionsReader(logoKey(h), opts, h.LogoReader)
		return logoKey(h), nil
	case h.LogoBytes != nil:
		opts := imageOpts(h.LogoFormat)
		p.RegisterImageOptionsReader(logoKey(h), opts, bytes.NewReader(h.LogoBytes))
		return logoKey(h), nil
	case h.LogoPath != "":
		f, err := os.Open(h.LogoPath)
		if err != nil {
			return "", err
		}
		defer f.Close()
		opts := imageOpts(h.LogoFormat)
		if opts.ImageType == "" {
			opts.ImageType = formatFromPath(h.LogoPath)
		}
		p.RegisterImageOptionsReader(h.LogoPath, opts, f)
		return h.LogoPath, nil
	default:
		return "", nil
	}
}

func imageOpts(format string) gofpdf.ImageOptions {
	return gofpdf.ImageOptions{ImageType: format, ReadDpi: false}
}

func formatFromPath(path string) string {
	switch ext := lower(extOf(path)); ext {
	case ".png":
		return "PNG"
	case ".jpg", ".jpeg":
		return "JPG"
	case ".gif":
		return "GIF"
	default:
		return "PNG"
	}
}

func logoKey(h Header) string {
	if h.LogoPath != "" {
		return h.LogoPath
	}
	return fmt.Sprintf("logo-%p", h.LogoBytes)
}

func imageSize(h Header) (int, int, error) {
	var r *bytes.Reader
	switch {
	case h.LogoBytes != nil:
		r = bytes.NewReader(h.LogoBytes)
	case h.LogoPath != "":
		data, err := os.ReadFile(h.LogoPath)
		if err != nil {
			return 0, 0, err
		}
		r = bytes.NewReader(data)
	default:
		return 0, 0, fmt.Errorf("no logo source")
	}
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func extOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' {
			break
		}
	}
	return ""
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
