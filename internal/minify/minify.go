// Package minify shrinks HTML, CSS, and JS output for production builds.
package minify

import (
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
)

var m = func() *minify.M {
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("application/javascript", js.Minify)
	return m
}()

// HTML minifies an HTML document. Whitespace inside <pre>/<textarea> is
// preserved, so highlighted code blocks are untouched.
func HTML(b []byte) ([]byte, error) { return m.Bytes("text/html", b) }

// CSS minifies a stylesheet.
func CSS(b []byte) ([]byte, error) { return m.Bytes("text/css", b) }

// JS minifies a script.
func JS(b []byte) ([]byte, error) { return m.Bytes("application/javascript", b) }
