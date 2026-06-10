package renderer

import (
	"fmt"
	"html/template"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/thisisbalu/nemi/internal/content"
)

// funcMap is the set of functions available to every template.
var funcMap = template.FuncMap{
	"not":        func(v bool) bool { return !v },
	"now":        time.Now,
	"dateFormat": dateFormat,
	"lower":      strings.ToLower,
	"upper":      strings.ToUpper,
	"title":      title,
	"urlize":     content.Urlize,
	"truncate":   truncate,
	"first":      first,
	"where":      where,
	"sortByDate": sortByDate,
	"uniqueTags": uniqueTags,
}

// uniqueTags returns the sorted, de-duplicated set of tags used across pages.
func uniqueTags(pages []content.Page) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range pages {
		for _, t := range p.Tags {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sort.Strings(out)
	return out
}

// dateFormat renders t using a Go reference layout, e.g.
// {{dateFormat "Jan 2, 2006" .Page.Date}}.
func dateFormat(layout string, t time.Time) string {
	return t.Format(layout)
}

// title upper-cases the first letter of each whitespace-separated word.
func title(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// truncate shortens s to at most n runes, backing up to the last word
// boundary and appending an ellipsis. Strings already within n are unchanged.
func truncate(n int, s string) string {
	runes := []rune(s)
	if n < 0 || len(runes) <= n {
		return s
	}
	cut := string(runes[:n])
	if i := strings.LastIndex(cut, " "); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ") + "…"
}

// first returns the first n elements of a slice, or the whole slice when it is
// shorter. Non-slice values are returned unchanged.
func first(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	if n < 0 {
		n = 0
	}
	if n > v.Len() {
		n = v.Len()
	}
	return v.Slice(0, n).Interface()
}

// where filters pages, keeping those whose named field equals value. Comparison
// is by string form, so {{where .Site.Pages "Featured" true}} and
// {{where .Site.Pages "Section" "blog"}} both work. Unknown fields match nothing.
func where(pages []content.Page, field string, value any) []content.Page {
	want := fmt.Sprintf("%v", value)
	var out []content.Page
	for _, p := range pages {
		fv := reflect.ValueOf(p).FieldByName(field)
		if !fv.IsValid() {
			continue
		}
		if fmt.Sprintf("%v", fv.Interface()) == want {
			out = append(out, p)
		}
	}
	return out
}
