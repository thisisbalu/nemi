// Package ogimage renders 1200×630 social-share cards (Open Graph images) for
// pages. It is pure Go — text is drawn with the embedded Go fonts via
// x/image/font, so there is no cgo and no font file to ship.
package ogimage

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	width  = 1200
	height = 630
	pad    = 80
	maxLn  = 5 // cap title lines so long titles don't overflow
)

// Near-monochrome dark palette, matching Nemi's dark theme — reads well on most
// social timelines.
var (
	colBg    = color.RGBA{0x20, 0x21, 0x25, 0xff}
	colText  = color.RGBA{0xe7, 0xe7, 0xe9, 0xff}
	colMuted = color.RGBA{0xa1, 0xa1, 0xaa, 0xff}
)

func newFace(ttf []byte, size float64) (font.Face, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

// Generate writes a social card PNG to outPath with the given title (wrapped)
// and a subtitle (e.g. the site name) along the bottom.
func Generate(outPath, title, subtitle string) error {
	titleFace, err := newFace(gobold.TTF, 64)
	if err != nil {
		return err
	}
	defer titleFace.Close()
	subFace, err := newFace(goregular.TTF, 30)
	if err != nil {
		return err
	}
	defer subFace.Close()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(colBg), image.Point{}, draw.Src)

	// Title — top-aligned, wrapped to the content width.
	lines := wrap(title, titleFace, width-2*pad)
	if len(lines) > maxLn {
		lines = lines[:maxLn]
	}
	td := &font.Drawer{Dst: img, Src: image.NewUniform(colText), Face: titleFace}
	lineH := titleFace.Metrics().Height.Ceil() + 12
	y := pad + titleFace.Metrics().Ascent.Ceil()
	for _, ln := range lines {
		td.Dot = fixed.P(pad, y)
		td.DrawString(ln)
		y += lineH
	}

	// Accent tick + subtitle along the bottom.
	draw.Draw(img, image.Rect(pad, height-pad-54, pad+56, height-pad-50),
		image.NewUniform(colText), image.Point{}, draw.Src)
	if subtitle != "" {
		sd := &font.Drawer{Dst: img, Src: image.NewUniform(colMuted), Face: subFace}
		sd.Dot = fixed.P(pad, height-pad)
		sd.DrawString(subtitle)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// wrap greedily breaks s into lines no wider than maxW pixels in face f.
func wrap(s string, f font.Face, maxW int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		test := cur + " " + w
		if font.MeasureString(f, test).Ceil() <= maxW {
			cur = test
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	return append(lines, cur)
}
