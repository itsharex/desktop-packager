package main

// ProxyRule 反向代理规则结构体
// 定义单条代理规则，用于将前端请求转发到后端服务器
type ProxyRule struct {
	Path    string `json:"path"`    // 路径前缀，例如 "/api/"
	Target  string `json:"target"`  // 目标服务器地址，例如 "http://localhost:8080/"
	Rewrite string `json:"rewrite"` // 路径重写规则，例如 "/v2"（可选）
	Enabled bool   `json:"enabled"` // 是否启用此规则
}

// BuildConfig 构建配置结构体
// 包含生成桌面应用所需的所有配置信息
type BuildConfig struct {
	AppName          string      `json:"appName"`          // 应用名称
	IconPath         string      `json:"iconPath"`         // 图标文件路径（.ico 或 .png）
	DistPath         string      `json:"distPath"`         // 前端构建产物目录路径
	TempPath         string      `json:"tempPath"`         // 临时文件目录路径（可选）
	ProxyRules       []ProxyRule `json:"proxyRules"`       // 反向代理规则列表
	WindowWidth      int         `json:"windowWidth"`      // 窗口宽度（像素）
	WindowHeight     int         `json:"windowHeight"`     // 窗口高度（像素）
	WindowFullscreen bool        `json:"windowFullscreen"` // 是否全屏显示
	WindowMaximized  bool        `json:"windowMaximized"`  // 是否最大化显示
}

// ProxyConfig 代理配置结构体
// 用于生成代理配置文件 (proxy_config.json)
type ProxyConfig struct {
	Rules []ProxyRule `json:"rules"` // 代理规则列表
}

// DistInfo 构建产物信息结构体
// 包含导入的前端构建产物的统计信息
type DistInfo struct {
	Path      string `json:"path"`      // 构建产物目录路径
	FileCount int    `json:"fileCount"` // 文件数量
	TotalSize int64  `json:"totalSize"` // 文件总大小（字节）
	Valid     bool   `json:"valid"`     // 是否有效（包含 index.html）
}

