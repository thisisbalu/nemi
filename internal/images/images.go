// Package images generates responsive, resized copies of raster images so the
// build can emit <img srcset> with intrinsic dimensions and lazy loading. It is
// pure Go (no cgo), so it cross-compiles with CGO_ENABLED=0 like the rest of
// Nemi. Format conversion (WebP/AVIF) is intentionally out of scope for now —
// it would require cgo or a wasm runtime.
package images

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// Variant is one generated width of a source image.
type Variant struct {
	URL   string // root-relative URL, e.g. /photo-480.jpg
	Width int
}

// Processed describes the responsive set produced for one source image,
// including its intrinsic dimensions (used for the width/height attributes that
// prevent layout shift).
type Processed struct {
	Width    int
	Height   int
	Variants []Variant // ascending by width; always ends with the original
}

// Supported reports whether urlPath points at a raster format we can resize.
// SVG and GIF are passed through untouched (vector / animation).
func Supported(urlPath string) bool {
	switch strings.ToLower(path.Ext(urlPath)) {
	case ".jpg", ".jpeg", ".png":
		return true
	}
	return false
}

// Process decodes the image served at urlPath (resolved under publicDir) and
// writes a resized copy for each width smaller than the image, alongside the
// original. It returns the intrinsic dimensions and the variant set (the
// original is always included as the largest variant).
func Process(publicDir, urlPath string, widths []int, quality int) (Processed, error) {
	srcPath := filepath.Join(publicDir, filepath.FromSlash(strings.TrimPrefix(urlPath, "/")))
	f, err := os.Open(srcPath)
	if err != nil {
		return Processed{}, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return Processed{}, err
	}
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw == 0 || ih == 0 {
		return Processed{}, fmt.Errorf("image %s has zero dimension", urlPath)
	}
	ext := path.Ext(urlPath)
	out := Processed{Width: iw, Height: ih}

	for _, w := range widths {
		if w >= iw {
			continue // never upscale
		}
		h := ih * w / iw
		if h < 1 {
			h = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)

		vURL := variantURL(urlPath, w)
		vPath := filepath.Join(publicDir, filepath.FromSlash(strings.TrimPrefix(vURL, "/")))
		if err := encode(vPath, ext, dst, quality); err != nil {
			return Processed{}, err
		}
		out.Variants = append(out.Variants, Variant{URL: vURL, Width: w})
	}
	out.Variants = append(out.Variants, Variant{URL: urlPath, Width: iw})
	return out, nil
}

// variantURL inserts the width before the extension: /a/photo.jpg + 480 →
// /a/photo-480.jpg.
func variantURL(urlPath string, w int) string {
	ext := path.Ext(urlPath)
	return fmt.Sprintf("%s-%d%s", strings.TrimSuffix(urlPath, ext), w, ext)
}

func encode(outPath, ext string, img image.Image, quality int) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
	case ".png":
		return png.Encode(f, img)
	}
	return fmt.Errorf("unsupported format %q", ext)
}
