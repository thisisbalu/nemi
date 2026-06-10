package images

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeTestPNG creates a w×h PNG at publicDir/<name> and returns its URL path.
func writeTestPNG(t *testing.T, publicDir, name string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	p := filepath.Join(publicDir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return "/" + name
}

func decodeDims(t *testing.T, path string) (int, int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Width, cfg.Height
}

func TestProcessGeneratesVariants(t *testing.T) {
	dir := t.TempDir()
	url := writeTestPNG(t, dir, "img/photo.png", 1000, 500)

	got, err := Process(dir, url, []int{480, 800, 1200}, 82)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if got.Width != 1000 || got.Height != 500 {
		t.Errorf("intrinsic dims = %dx%d, want 1000x500", got.Width, got.Height)
	}
	// 480 and 800 are < 1000 → generated; 1200 skipped (no upscale); original appended.
	if len(got.Variants) != 3 {
		t.Fatalf("variants = %d, want 3 (480, 800, original)", len(got.Variants))
	}
	wantWidths := []int{480, 800, 1000}
	for i, v := range got.Variants {
		if v.Width != wantWidths[i] {
			t.Errorf("variant[%d].Width = %d, want %d", i, v.Width, wantWidths[i])
		}
	}
	// the 480 variant should exist on disk with the right dimensions (aspect kept)
	w, h := decodeDims(t, filepath.Join(dir, "img", "photo-480.png"))
	if w != 480 || h != 240 {
		t.Errorf("resized 480 variant = %dx%d, want 480x240", w, h)
	}
	// the original is referenced, not rewritten
	if got.Variants[2].URL != url {
		t.Errorf("largest variant URL = %q, want original %q", got.Variants[2].URL, url)
	}
}

func TestProcessSmallImageOnlyOriginal(t *testing.T) {
	dir := t.TempDir()
	url := writeTestPNG(t, dir, "tiny.png", 300, 300)
	got, err := Process(dir, url, []int{480, 800}, 82)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(got.Variants) != 1 || got.Variants[0].Width != 300 {
		t.Errorf("small image should yield only the original, got %+v", got.Variants)
	}
}

func TestSupported(t *testing.T) {
	for _, c := range []struct {
		url  string
		want bool
	}{
		{"/a.jpg", true}, {"/a.jpeg", true}, {"/a.PNG", true},
		{"/a.svg", false}, {"/a.gif", false}, {"/a.webp", false},
	} {
		if Supported(c.url) != c.want {
			t.Errorf("Supported(%q) = %v, want %v", c.url, !c.want, c.want)
		}
	}
}
