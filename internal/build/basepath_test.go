package build

import "testing"

func TestRewriteBasePathHTML(t *testing.T) {
	in := `<a href="/blog/">x</a> <link href="/style.css"> ` +
		`<img src="/a.jpg" srcset="/a-480.jpg 480w, /a.jpg 960w"> ` +
		`<a href="https://x.io/y">ext</a> <script src="//cdn.example/x.js"></script> ` +
		`<style>body{background:url(/bg.png)}</style>`
	got := rewriteBasePath(in, "/nemi", true)

	wants := []string{
		`href="/nemi/blog/"`,
		`href="/nemi/style.css"`,
		`src="/nemi/a.jpg"`,
		`srcset="/nemi/a-480.jpg 480w, /nemi/a.jpg 960w"`,
		`url(/nemi/bg.png)`,
		`href="https://x.io/y"`,    // absolute untouched
		`src="//cdn.example/x.js"`, // protocol-relative untouched
	}
	for _, w := range wants {
		if !contains(got, w) {
			t.Errorf("missing %q in:\n%s", w, got)
		}
	}
}

func TestRewriteBasePathCSS(t *testing.T) {
	got := rewriteBasePath(`a{background:url("/img/x.png")} b{background:url(/y.png)}`, "/nemi", false)
	for _, w := range []string{`url("/nemi/img/x.png")`, `url(/nemi/y.png)`} {
		if !contains(got, w) {
			t.Errorf("missing %q in %s", w, got)
		}
	}
}

func TestRewriteBasePathEmptyNoop(t *testing.T) {
	in := `<a href="/blog/">x</a>`
	if got := rewriteBasePath(in, "", true); got != in {
		t.Errorf("empty base should be a no-op, got %s", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
