package pdf

import (
	_ "embed"

	"github.com/go-pdf/fpdf"
)

// The quotation PDF is set in Plus Jakarta Sans (PLAN §5.5), the app's own typeface. The files are the static
// Regular and Bold cuts from the designer's repository (github.com/tokotype/PlusJakartaSans, fonts/ttf), licensed
// under the SIL Open Font License 1.1 (fonts/OFL.txt). fpdf embeds only the glyphs a document uses, so a quotation
// stays a few tens of kB.
var (
	//go:embed fonts/PlusJakartaSans-Regular.ttf
	embeddedRegular []byte
	//go:embed fonts/PlusJakartaSans-Bold.ttf
	embeddedBold []byte
)

// fontSet is the family the renderer writes with and how a UTF-8 string
// becomes what that family can print.
type fontSet struct {
	family    string
	translate func(string) string
	utf8      bool
}

// setupFonts registers the fonts on a new document. core keeps fpdf's built-in Helvetica (cp1252 text, nothing
// embedded): tests use it to read the content streams as plain text, which an embedded font writes as glyph ids.
func setupFonts(doc *fpdf.Fpdf, core bool) fontSet {
	if core {
		return fontSet{family: "Helvetica", translate: doc.UnicodeTranslatorFromDescriptor("")}
	}
	doc.AddUTF8FontFromBytes("PlusJakartaSans", "", embeddedRegular)
	doc.AddUTF8FontFromBytes("PlusJakartaSans", "B", embeddedBold)
	return fontSet{family: "PlusJakartaSans", translate: func(s string) string { return s }, utf8: true}
}
