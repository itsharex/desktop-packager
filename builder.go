package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"deploy-app/internal/nginxproxy"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed templates/base/base.exe
var baseExe []byte

const resourceFooterMagic uint32 = 0x5245534F // "RESO"

type resourceFooter struct {
	Magic  uint32
	Offset uint32
}

// BuildApp 执行完整的应用构建流程
func (a *App) BuildApp(config BuildConfig) error {
	if !a.beginBuild() {
		return fmt.Errorf("已有构建任务正在进行")
	}
	defer a.endBuild()

	if err := ValidateBuildConfig(config); err != nil {
		return err
	}
	appName, err := SanitizeAppName(config.AppName)
	if err != nil {
		return err
	}
	config.AppName = appName

	// Normalize proxy paths for nginx semantics.
	for i := range config.ProxyRules {
		config.ProxyRules[i].Path = nginxproxy.NormalizeLocation(config.ProxyRules[i].Path)
		config.ProxyRules[i].Target = strings.TrimSpace(config.ProxyRules[i].Target)
		config.ProxyRules[i].Rewrite = strings.TrimSpace(config.ProxyRules[i].Rewrite)
	}

	a.emitProgress("准备工作目录", 5)

	tempBase := strings.TrimSpace(config.TempPath)
	if tempBase == "" {
		tempBase = filepath.Dir(config.DistPath)
	}
	if err := os.MkdirAll(tempBase, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	tempDir, err := os.MkdirTemp(tempBase, "deploy-build-*")
	if err != nil {
		return fmt.Errorf("创建工作目录失败: %w", err)
	}
	a.trackTempDir(tempDir)
	defer func() {
		_ = os.RemoveAll(tempDir)
		a.untrackTempDir(tempDir)
	}()

	a.emitProgress("复制基础程序", 20)
	if len(baseExe) < 1024 || baseExe[0] != 'M' || baseExe[1] != 'Z' {
		return fmt.Errorf("内置 base.exe 无效或仍是占位文件，请先执行: go run ./cmd/build-base")
	}
	exePath := filepath.Join(tempDir, config.AppName+".exe")
	if err := os.WriteFile(exePath, baseExe, 0755); err != nil {
		return fmt.Errorf("复制基础程序失败: %w", err)
	}

	// Icon/version patch MUST happen before appending resources.
	a.emitProgress("修补应用图标与版本信息", 35)
	if err := a.patchIcon(exePath, config); err != nil {
		return fmt.Errorf("修补图标失败: %w", err)
	}

	a.emitProgress("打包前端资源", 55)
	if err := a.appendResources(exePath, config); err != nil {
		return fmt.Errorf("打包资源失败: %w", err)
	}

	a.emitProgress("保存文件", 85)
	finalPath, err := a.copyOutputToDesktop(exePath, config.AppName)
	if err != nil {
		return fmt.Errorf("输出文件失败: %w", err)
	}

	a.emitProgress("构建完成!", 100)
	a.emitBuildComplete(finalPath)
	return nil
}

// appendResources streams a resource zip and footer onto the exe without loading the whole exe into memory twice.
func (a *App) appendResources(exePath string, config BuildConfig) error {
	zipFile, err := os.CreateTemp(filepath.Dir(exePath), "resources-*.zip")
	if err != nil {
		return fmt.Errorf("创建临时 zip 失败: %w", err)
	}
	zipPath := zipFile.Name()
	defer func() {
		zipFile.Close()
		os.Remove(zipPath)
	}()

	zw := zip.NewWriter(zipFile)

	a.emitProgress("写入代理配置", 60)
	proxyJSON, err := buildProxyConfigJSON(config.ProxyRules)
	if err != nil {
		return err
	}
	if err := writeZipBytes(zw, "proxy_config.json", proxyJSON); err != nil {
		return err
	}

	appJSON, err := buildAppConfigJSON(config)
	if err != nil {
		return err
	}
	if err := writeZipBytes(zw, "app_config.json", appJSON); err != nil {
		return err
	}

	a.emitProgress("复制前端文件", 70)
	if err := addDirToZip(zw, config.DistPath, "dist"); err != nil {
		return fmt.Errorf("打包前端文件失败: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("关闭 zip 失败: %w", err)
	}
	if err := zipFile.Sync(); err != nil {
		return err
	}
	if _, err := zipFile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	exeStat, err := os.Stat(exePath)
	if err != nil {
		return fmt.Errorf("读取 exe 信息失败: %w", err)
	}
	zipOffset := exeStat.Size()
	if zipOffset > math.MaxUint32 {
		return fmt.Errorf("可执行文件过大，超出资源偏移上限")
	}

	out, err := os.OpenFile(exePath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return fmt.Errorf("打开 exe 追加写入失败: %w", err)
	}
	defer out.Close()

	a.emitProgress("写入资源数据", 80)
	if _, err := io.Copy(out, zipFile); err != nil {
		return fmt.Errorf("写入 zip 数据失败: %w", err)
	}

	footer := resourceFooter{
		Magic:  resourceFooterMagic,
		Offset: uint32(zipOffset),
	}
	if err := binary.Write(out, binary.LittleEndian, &footer); err != nil {
		return fmt.Errorf("写入 footer 失败: %w", err)
	}
	return nil
}

func writeZipBytes(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("创建 zip 条目失败: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", name, err)
	}
	return nil
}

func (a *App) patchIcon(exePath string, config BuildConfig) error {
	// Always refresh version metadata; icon is optional.
	exeData, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("读取 exe 失败: %w", err)
	}

	rs, err := winres.LoadFromEXE(bytes.NewReader(exeData))
	if err != nil {
		rs = &winres.ResourceSet{}
	}

	if config.IconPath != "" {
		icon, err := loadIcon(config.IconPath)
		if err != nil {
			return err
		}
		if err := replaceIconResource(rs, winres.ID(1), icon); err != nil {
			return fmt.Errorf("设置图标资源失败: %w", err)
		}
		if err := replaceIconResource(rs, winres.ID(3), icon); err != nil {
			return fmt.Errorf("设置窗口图标资源失败: %w", err)
		}
	}

	rs.SetManifest(winres.AppManifest{
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
	})

	fileVer := parseVersion(config.Version)
	vi := version.Info{
		FileVersion:    fileVer,
		ProductVersion: fileVer,
	}
	product := config.AppName
	desc := strings.TrimSpace(config.Description)
	if desc == "" {
		desc = config.AppName
	}
	company := strings.TrimSpace(config.Company)
	setVI := func(key, value string) {
		if value == "" {
			return
		}
		// Prefer en-US for Windows Explorer Details; fall back to neutral.
		if err := vi.Set(version.LangDefault, key, value); err != nil {
			_ = vi.Set(version.LangNeutral, key, value)
		}
	}
	setVI(version.ProductName, product)
	setVI(version.FileDescription, desc)
	setVI(version.OriginalFilename, config.AppName+".exe")
	// CompanyName 是“公司”，LegalCopyright 对应资源管理器“详细信息 → 版权”。
	// Windows 11 属性页默认展示版权，不一定展示公司名，因此两者都写入。
	setVI(version.CompanyName, company)
	if company != "" {
		setVI(version.LegalCopyright, "Copyright © "+strconv.Itoa(time.Now().Year())+" "+company)
	}
	if v := strings.TrimSpace(config.Version); v != "" {
		setVI(version.ProductVersion, v)
		setVI(version.FileVersion, v)
	}
	rs.SetVersionInfo(vi)

	tmpPath := exePath + ".tmp"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	if err := rs.WriteToEXE(outFile, bytes.NewReader(exeData)); err != nil {
		outFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("写入 PE 资源失败: %w", err)
	}
	outFile.Close()

	if err := os.Remove(exePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("删除原文件失败: %w", err)
	}
	if err := os.Rename(tmpPath, exePath); err != nil {
		return fmt.Errorf("替换 exe 失败: %w", err)
	}
	return nil
}

func loadIcon(path string) (*winres.Icon, error) {
	ext := strings.ToLower(filepath.Ext(path))
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("读取图标失败: %w", err)
	}
	defer f.Close()

	switch ext {
	case ".png":
		img, _, err := image.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("解码 PNG 图标失败: %w", err)
		}
		icon, err := winres.NewIconFromResizedImage(img, nil)
		if err != nil {
			return nil, fmt.Errorf("创建图标失败: %w", err)
		}
		return icon, nil
	case ".ico":
		icon, err := winres.LoadICO(f)
		if err != nil {
			return nil, fmt.Errorf("加载 ICO 图标失败: %w", err)
		}
		return icon, nil
	default:
		return nil, fmt.Errorf("不支持的图标格式: %s", ext)
	}
}

func parseVersion(v string) [4]uint16 {
	out := [4]uint16{1, 0, 0, 0}
	v = strings.TrimSpace(v)
	if v == "" {
		return out
	}
	parts := strings.Split(v, ".")
	for i := 0; i < len(parts) && i < 4; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 || n > 65535 {
			return [4]uint16{1, 0, 0, 0}
		}
		out[i] = uint16(n)
	}
	return out
}

func replaceIconResource(rs *winres.ResourceSet, resID winres.ID, icon *winres.Icon) error {
	langIDs := []uint16{winres.LCIDNeutral, winres.LCIDDefault}
	oldLangIDs := make([]uint16, 0)
	addLangID := func(langID uint16) {
		for _, existing := range langIDs {
			if existing == langID {
				return
			}
		}
		langIDs = append(langIDs, langID)
	}
	rs.WalkType(winres.RT_GROUP_ICON, func(existingID winres.Identifier, langID uint16, _ []byte) bool {
		id, ok := existingID.(winres.ID)
		if !ok || id != resID {
			return true
		}
		oldLangIDs = append(oldLangIDs, langID)
		addLangID(langID)
		return true
	})
	for _, langID := range oldLangIDs {
		if err := rs.Set(winres.RT_GROUP_ICON, resID, langID, nil); err != nil {
			return err
		}
	}
	for _, langID := range langIDs {
		if err := rs.SetIconTranslation(resID, langID, icon); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) copyOutputToDesktop(srcPath, appName string) (string, error) {
	outputPath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "保存应用",
		DefaultFilename: appName + ".exe",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "可执行文件", Pattern: "*.exe"},
		},
	})
	if err != nil {
		return "", err
	}
	if outputPath == "" {
		return "", fmt.Errorf("用户取消了保存")
	}
	if err := copyFile(srcPath, outputPath); err != nil {
		return "", err
	}
	return outputPath, nil
}

func (a *App) emitProgress(step string, progress int) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	a.mu.Lock()
	if progress < a.lastProgress {
		progress = a.lastProgress
	}
	a.lastProgress = progress
	a.mu.Unlock()
	wailsRuntime.EventsEmit(a.ctx, "build:progress", map[string]interface{}{
		"step":     step,
		"progress": progress,
	})
}

func (a *App) emitBuildComplete(outputPath string) {
	wailsRuntime.EventsEmit(a.ctx, "build:complete", map[string]interface{}{
		"outputPath": outputPath,
	})
}

func buildProxyConfigJSON(rules []ProxyRule) ([]byte, error) {
	if rules == nil {
		rules = []ProxyRule{}
	}
	// Normalize before marshal.
	normalized := make([]ProxyRule, 0, len(rules))
	for _, r := range rules {
		r.Path = nginxproxy.NormalizeLocation(r.Path)
		r.Target = strings.TrimSpace(r.Target)
		r.Rewrite = strings.TrimSpace(r.Rewrite)
		normalized = append(normalized, r)
	}
	cfg := ProxyConfig{Rules: normalized}
	return json.MarshalIndent(cfg, "", "  ")
}

func buildAppConfigJSON(config BuildConfig) ([]byte, error) {
	width := config.WindowWidth
	if width < 400 || width > 3840 {
		width = 1024
	}
	height := config.WindowHeight
	if height < 300 || height > 2160 {
		height = 768
	}
	cfg := AppRuntimeConfig{
		Width:        width,
		Height:       height,
		Fullscreen:   config.WindowFullscreen,
		Maximized:    config.WindowMaximized,
		ConfirmClose: config.ConfirmClose,
		Version:      strings.TrimSpace(config.Version),
		Description:  strings.TrimSpace(config.Description),
		Company:      strings.TrimSpace(config.Company),
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func addDirToZip(zw *zip.Writer, srcDir, zipPrefix string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		zipName := zipPrefix + "/" + filepath.ToSlash(relPath)
		if d.IsDir() {
			_, err := zw.Create(zipName + "/")
			return err
		}
		w, err := zw.Create(zipName)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		f.Close()
		return copyErr
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}