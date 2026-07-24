package main

// ProxyRule 反向代理规则（nginx location + proxy_pass 语义）
type ProxyRule struct {
	Path    string `json:"path"`    // location 前缀，例如 "/api/"
	Target  string `json:"target"`  // proxy_pass 目标，例如 "http://localhost:8080/"
	Rewrite string `json:"rewrite"` // 可选，覆盖 target 的 URI 替换前缀
	Enabled bool   `json:"enabled"`
}

// BuildConfig 构建配置
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
	ConfirmClose     bool        `json:"confirmClose"`
	Version          string      `json:"version"`
	Description      string      `json:"description"`
	Company          string      `json:"company"`
}

// ProxyConfig 写入 proxy_config.json
type ProxyConfig struct {
	Rules []ProxyRule `json:"rules"`
}

// AppRuntimeConfig 写入 app_config.json
type AppRuntimeConfig struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Fullscreen   bool   `json:"fullscreen"`
	Maximized    bool   `json:"maximized"`
	ConfirmClose bool   `json:"confirmClose"`
	Version      string `json:"version"`
	Description  string `json:"description"`
	Company      string `json:"company"`
}

// DistInfo 构建产物信息
type DistInfo struct {
	Path      string `json:"path"`
	FileCount int    `json:"fileCount"`
	TotalSize int64  `json:"totalSize"`
	Valid     bool   `json:"valid"`
}