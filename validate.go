package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"deploy-app/internal/nginxproxy"
)

var (
	invalidAppNameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	windowsReserved     = map[string]struct{}{
		"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
		"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
		"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
	}
	semverLike = regexp.MustCompile(`^\d+(\.\d+){0,3}$`)
)

// SanitizeAppName validates and normalizes a Windows-safe application name.
func SanitizeAppName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("应用名称不能为空")
	}
	if utf8.RuneCountInString(name) > 50 {
		return "", fmt.Errorf("应用名称长度不能超过 50 个字符")
	}
	if strings.Contains(name, "..") {
		return "", fmt.Errorf("应用名称不能包含 ..")
	}
	if invalidAppNameChars.MatchString(name) {
		return "", fmt.Errorf("应用名称包含非法字符（不能包含 <>:\"/\\|?* 和控制字符）")
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return "", fmt.Errorf("应用名称不能以空格或点结尾")
	}
	upper := strings.ToUpper(name)
	if _, ok := windowsReserved[upper]; ok {
		return "", fmt.Errorf("应用名称不能使用 Windows 保留名: %s", name)
	}
	// Also reject reserved names with extension-like suffixes: CON.txt
	if dot := strings.IndexByte(upper, '.'); dot > 0 {
		if _, ok := windowsReserved[upper[:dot]]; ok {
			return "", fmt.Errorf("应用名称不能使用 Windows 保留名: %s", name)
		}
	}
	return name, nil
}

// ValidateProxyRule validates a single proxy rule using nginx-compatible expectations.
func ValidateProxyRule(rule ProxyRule, index int) error {
	prefix := fmt.Sprintf("代理规则 #%d", index+1)
	if !rule.Enabled {
		return nil
	}
	path := nginxproxy.NormalizeLocation(rule.Path)
	if path == "" {
		return fmt.Errorf("%s: 路径前缀不能为空", prefix)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("%s: 路径前缀非法", prefix)
	}
	target := strings.TrimSpace(rule.Target)
	if target == "" {
		return fmt.Errorf("%s: 目标地址不能为空", prefix)
	}
	u, err := url.Parse(target)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s: 目标地址无效，需形如 http://host:port/ 或 https://host", prefix)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s: 目标地址仅支持 http/https", prefix)
	}
	if rw := strings.TrimSpace(rule.Rewrite); rw != "" {
		if !strings.HasPrefix(rw, "/") {
			return fmt.Errorf("%s: 重写路径应以 / 开头", prefix)
		}
		if strings.Contains(rw, "..") {
			return fmt.Errorf("%s: 重写路径非法", prefix)
		}
	}
	return nil
}

// ValidateBuildConfig validates the full build configuration before packaging.
func ValidateBuildConfig(config BuildConfig) error {
	name, err := SanitizeAppName(config.AppName)
	if err != nil {
		return err
	}
	_ = name

	dist := strings.TrimSpace(config.DistPath)
	if dist == "" {
		return fmt.Errorf("请先导入前端构建产物")
	}
	info, err := os.Stat(dist)
	if err != nil {
		return fmt.Errorf("构建产物目录无效: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("构建产物路径不是目录")
	}
	if _, err := os.Stat(filepath.Join(dist, "index.html")); err != nil {
		return fmt.Errorf("构建产物目录中未找到 index.html")
	}

	if config.IconPath != "" {
		if _, err := os.Stat(config.IconPath); err != nil {
			return fmt.Errorf("图标文件无效: %w", err)
		}
		ext := strings.ToLower(filepath.Ext(config.IconPath))
		if ext != ".ico" && ext != ".png" {
			return fmt.Errorf("图标仅支持 .ico 或 .png")
		}
	}

	if config.TempPath != "" {
		if err := os.MkdirAll(config.TempPath, 0755); err != nil {
			return fmt.Errorf("临时目录不可用: %w", err)
		}
	}

	if config.WindowWidth != 0 && (config.WindowWidth < 400 || config.WindowWidth > 3840) {
		return fmt.Errorf("窗口宽度需在 400-3840 之间")
	}
	if config.WindowHeight != 0 && (config.WindowHeight < 300 || config.WindowHeight > 2160) {
		return fmt.Errorf("窗口高度需在 300-2160 之间")
	}

	for i, rule := range config.ProxyRules {
		if err := ValidateProxyRule(rule, i); err != nil {
			return err
		}
	}

	if v := strings.TrimSpace(config.Version); v != "" && !semverLike.MatchString(v) {
		return fmt.Errorf("版本号格式无效，示例: 1.0.0")
	}
	return nil
}