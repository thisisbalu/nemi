package ogimage

import (
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate(t *testing.T) {
	out := filepath.Join(t.TempDir(), "card.png")
	if err := Generate(out, "A Reasonably Long Title That Should Wrap Across Lines", "My Site"); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Width != width || cfg.Height != height {
		t.Errorf("card = %dx%d, want %dx%d", cfg.Width, cfg.Height, width, height)
	}
}

func TestGenerateEmptyTitle(t *testing.T) {
	out := filepath.Join(t.TempDir(), "card.png")
	if err := Generate(out, "", "Site"); err != nil {
		t.Errorf("empty title should still render: %v", err)
	}
}

func TestWrap(t *testing.T) {
	f, err := newFace([]byte(nil), 10)
	if err == nil {
		f.Close()
		t.Skip("expected parse failure on nil font")
	}
	// wrap is exercised via Generate; here just guard the empty case directly.
	if got := wrap("", nil, 100); got != nil {
		t.Errorf("wrap(\"\") = %v, want nil", got)
	}
}
