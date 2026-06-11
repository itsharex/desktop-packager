package main

// ProxyRule represents a single reverse proxy rule
type ProxyRule struct {
	Path    string `json:"path"`
	Target  string `json:"target"`
	Rewrite string `json:"rewrite"`
	Enabled bool   `json:"enabled"`
}

// BuildConfig holds all configuration for building a desktop app
type BuildConfig struct {
	AppName          string      `json:"appName"`
	IconPath         string      `json:"iconPath"`
	DistPath         string      `json:"distPath"`
	TempPath         string      `json:"tempPath"`
	ProxyRules       []ProxyRule `json:"proxyRules"`
	WindowWidth      int         `json:"windowWidth"`
	WindowHeight     int         `json:"windowHeight"`
	WindowFullscreen bool        `json:"windowFullscreen"`
	WindowMaximized  bool        `json:"windowMaximized"`
}

// ProxyConfig holds the proxy configuration for the generated app
type ProxyConfig struct {
	Rules []ProxyRule `json:"rules"`
}

// DistInfo holds information about an imported dist directory
type DistInfo struct {
	Path      string `json:"path"`
	FileCount int    `json:"fileCount"`
	TotalSize int64  `json:"totalSize"`
	Valid     bool   `json:"valid"`
}

