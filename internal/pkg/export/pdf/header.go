package pdf

import "io"

// Header describes a branded page header rendered above the table. All fields
// are optional; when the header is zero-valued the default text title is used.
type Header struct {
	// Logo is the image source. Accepts a file path (set LogoPath instead) or
	// raw bytes via LogoReader/LogoBytes.
	LogoBytes []byte
	// LogoPath is a filesystem path to the logo image (png/jpg/gif).
	LogoPath string
	// LogoReader is a custom reader for the logo image. Takes precedence over
	// LogoBytes and LogoPath when set.
	LogoReader io.Reader
	// LogoFormat is the gofpdf image type: "PNG", "JPG" or "GIF". Required when
	// providing raw bytes through LogoBytes or LogoReader.
	LogoFormat string
	// LogoWidth is the rendered logo width in millimeters. Height is scaled to
	// preserve aspect ratio.
	LogoWidth float64
	// LeftLines are text lines aligned to the left of the header (e.g. company
	// name, address).
	LeftLines []string
	// RightLines are text lines aligned to the right of the header (e.g. report
	// code, print date).
	RightLines []string
	// Repeat redraws this header on every page. When false it is drawn once on
	// the first page only.
	Repeat bool
	// Divider draws a horizontal rule below the header block.
	Divider bool
}
