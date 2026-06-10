// Package data loads structured data files (YAML, TOML, JSON) from a site's
// data/ directory and exposes them to templates as a nested map.
package data

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// decoders maps a lower-cased file extension to its unmarshal function.
var decoders = map[string]func([]byte, any) error{
	".yaml": yaml.Unmarshal,
	".yml":  yaml.Unmarshal,
	".json": json.Unmarshal,
	".toml": toml.Unmarshal,
}

// Load walks dir and decodes every supported file into a nested map keyed by
// directory segments and the file's base name. For example data/social.yaml is
// reachable as {{.Site.Data.social}} and data/team/lead.json as
// {{.Site.Data.team.lead}}. A missing dir yields an empty (non-nil) map.
func Load(dir string) (map[string]any, error) {
	root := map[string]any{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		decode, ok := decoders[strings.ToLower(filepath.Ext(path))]
		if !ok {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var value any
		if err := decode(raw, &value); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		rel, _ := filepath.Rel(dir, path)
		insert(root, keysFor(rel), value)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return root, nil
}

// keysFor turns a relative path into the map key path, dropping the extension:
// "team/lead.json" -> ["team", "lead"].
func keysFor(rel string) []string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, filepath.Ext(rel))
	return strings.Split(rel, "/")
}

// insert stores value at the nested key path, creating intermediate maps.
func insert(root map[string]any, keys []string, value any) {
	m := root
	for _, k := range keys[:len(keys)-1] {
		next, ok := m[k].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[k] = next
		}
		m = next
	}
	m[keys[len(keys)-1]] = value
}
