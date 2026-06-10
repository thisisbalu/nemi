package config

import (
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Title       string      `toml:"title"`
	BaseURL     string      `toml:"baseURL"`
	Description string      `toml:"description"`
	Author      Author      `toml:"author"`
	Pages       PagesConfig `toml:"pages"`
	Menu        []MenuItem  `toml:"menu"`
	Paginate    int         `toml:"paginate"`
	Giscus      Giscus      `toml:"giscus"`
}

// Giscus configures GitHub-Discussions-backed comments (opt-in). Comments
// render only when both Repo and RepoID are set.
type Giscus struct {
	Repo       string `toml:"repo"`
	RepoID     string `toml:"repo_id"`
	Category   string `toml:"category"`
	CategoryID string `toml:"category_id"`
	Mapping    string `toml:"mapping"`
}

// Enabled reports whether comments should be rendered.
func (g Giscus) Enabled() bool {
	return g.Repo != "" && g.RepoID != ""
}

// MenuItem is one navigation entry from a [[menu]] table in nemi.toml.
type MenuItem struct {
	Name     string `toml:"name"`
	URL      string `toml:"url"`
	Weight   int    `toml:"weight"`
	External bool   `toml:"external"`
}

// IsExternal reports whether the link leaves the site, either because it was
// marked external or because the URL has an http(s) scheme. Templates use it
// to add target="_blank" rel="noopener".
func (m MenuItem) IsExternal() bool {
	return m.External ||
		strings.HasPrefix(m.URL, "http://") ||
		strings.HasPrefix(m.URL, "https://")
}

type Author struct {
	Name     string `toml:"name"`
	Tagline  string `toml:"tagline"`
	Bio      string `toml:"bio"`
	Email    string `toml:"email"`
	GitHub   string `toml:"github"`
	Twitter  string `toml:"twitter"`
	LinkedIn string `toml:"linkedin"`
}

type PagesConfig struct {
	About      bool `toml:"about"`
	Resume     bool `toml:"resume"`
	Projects   bool `toml:"projects"`
	OpenSource bool `toml:"open-source"`
	Blog       bool `toml:"blog"`
	Updates    bool `toml:"updates"`
	Now        bool `toml:"now"`
	Uses       bool `toml:"uses"`
	Speaking   bool `toml:"speaking"`
	Contact    bool `toml:"contact"`
}

func Load(path string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, err
	}
	// Order menu entries by weight; equal weights keep their file order.
	sort.SliceStable(cfg.Menu, func(i, j int) bool {
		return cfg.Menu[i].Weight < cfg.Menu[j].Weight
	})
	return cfg, nil
}
