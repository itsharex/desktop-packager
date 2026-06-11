package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed templates/base/base.exe
var baseExe []byte

// resourceFooterMagic is the magic number appended after the zip data
const resourceFooterMagic uint32 = 0x5245534F // "RESO"

// resourceFooter is the 8-byte footer appended after the zip data
type resourceFooter struct {
	Magic  uint32 // resourceFooterMagic
	Offset uint32 // byte offset where zip data starts
}

// BuildApp orchestrates the entire build pipeline (no Go dependency)
func (a *App) BuildApp(config BuildConfig) error {
	a.emitProgress("准备工作目录", 5)

	// Step 1: Create temp working directory
	// 临时目录是全流程配置：用户填写就必须使用；留空才使用前端产物所在目录。
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
	defer os.RemoveAll(tempDir)

	// Step 2: Copy base exe
	a.emitProgress("复制基础程序", 15)
	exePath := filepath.Join(tempDir, config.AppName+".exe")
	if err := os.WriteFile(exePath, baseExe, 0755); err != nil {
		return fmt.Errorf("复制基础程序失败: %w", err)
	}

	// Step 3: Patch icon (must be done BEFORE appending resources,
	// because WriteToEXE creates a new PE file and discards appended data)
	a.emitProgress("修补应用图标", 30)
	if err := a.patchIcon(exePath, config); err != nil {
		return fmt.Errorf("修补图标失败: %w", err)
	}

	// Step 4: Create resource zip and append to exe
	a.emitProgress("打包前端资源", 60)
	if err := a.appendResources(exePath, config); err != nil {
		return fmt.Errorf("打包资源失败: %w", err)
	}

	// Step 5: Copy output to user-chosen location
	a.emitProgress("保存文件", 90)
	finalPath, err := a.copyOutputToDesktop(exePath, config.AppName)
	if err != nil {
		return fmt.Errorf("输出文件失败: %w", err)
	}

	a.emitProgress("构建完成!", 100)
	a.emitBuildComplete(finalPath)

	return nil
}

// appendResources creates a zip containing dist/ and proxy_config.json,
// then appends it to the exe with an 8-byte footer.
func (a *App) appendResources(exePath string, config BuildConfig) error {
	// Build zip in memory
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)

	// Add proxy config
	a.emitProgress("写入代理配置", 40)
	proxyJSON := buildProxyConfigJSON(config.ProxyRules)
	w, err := zw.Create("proxy_config.json")
	if err != nil {
		return fmt.Errorf("创建 zip 条目失败: %w", err)
	}
	if _, err := w.Write(proxyJSON); err != nil {
		return fmt.Errorf("写入代理配置失败: %w", err)
	}

	// Add app config
	appJSON := buildAppConfigJSON(config)
	w, err = zw.Create("app_config.json")
	if err != nil {
		return fmt.Errorf("创建 zip 条目失败: %w", err)
	}
	if _, err := w.Write(appJSON); err != nil {
		return fmt.Errorf("写入应用配置失败: %w", err)
	}

	// Add dist files
	a.emitProgress("复制前端文件", 50)
	if err := addDirToZip(zw, config.DistPath, "dist"); err != nil {
		return fmt.Errorf("打包前端文件失败: %w", err)
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("关闭 zip 失败: %w", err)
	}

	// Read current exe content
	exeData, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("读取 exe 失败: %w", err)
	}

	// Write: exe content + zip data + footer
	outFile, err := os.Create(exePath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	// Write original exe content
	if _, err := outFile.Write(exeData); err != nil {
		return fmt.Errorf("写入 exe 内容失败: %w", err)
	}

	// Write zip data
	zipOffset := uint32(len(exeData))
	if _, err := outFile.Write(zipBuf.Bytes()); err != nil {
		return fmt.Errorf("写入 zip 数据失败: %w", err)
	}

	// Write footer
	footer := resourceFooter{
		Magic:  resourceFooterMagic,
		Offset: zipOffset,
	}
	if err := binary.Write(outFile, binary.LittleEndian, &footer); err != nil {
		return fmt.Errorf("写入 footer 失败: %w", err)
	}

	return nil
}

// patchIcon modifies the PE resources of the exe to embed the icon,
// manifest, and version info.
func (a *App) patchIcon(exePath string, config BuildConfig) error {
	if config.IconPath == "" {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(config.IconPath))
	var icon *winres.Icon

	if ext == ".png" {
		f, err := os.Open(config.IconPath)
		if err != nil {
			return fmt.Errorf("读取 PNG 图标失败: %w", err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			return fmt.Errorf("解码 PNG 图标失败: %w", err)
		}

		icon, err = winres.NewIconFromResizedImage(img, nil)
		if err != nil {
			return fmt.Errorf("创建图标失败: %w", err)
		}
	} else if ext == ".ico" {
		f, err := os.Open(config.IconPath)
		if err != nil {
			return fmt.Errorf("读取 ICO 图标失败: %w", err)
		}
		defer f.Close()

		icon, err = winres.LoadICO(f)
		if err != nil {
			return fmt.Errorf("加载 ICO 图标失败: %w", err)
		}
	} else {
		return fmt.Errorf("不支持的图标格式: %s", ext)
	}

	// Read the exe content
	exeData, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("读取 exe 失败: %w", err)
	}

	// Load existing resources from exe
	rs, err := winres.LoadFromEXE(bytes.NewReader(exeData))
	if err != nil {
		// If loading fails, create new resource set
		rs = &winres.ResourceSet{}
	}

	// Windows Shell 取最小图标 ID 作为 exe 图标；Wails 窗口标题栏固定读取 ID 3。
	// 这里必须替换旧语言资源，否则标题栏可能继续命中 base.exe 的默认图标。
	if err := replaceIconResource(rs, winres.ID(1), icon); err != nil {
		return fmt.Errorf("设置图标资源失败: %w", err)
	}
	if err := replaceIconResource(rs, winres.ID(3), icon); err != nil {
		return fmt.Errorf("设置窗口图标资源失败: %w", err)
	}

	// Set manifest
	rs.SetManifest(winres.AppManifest{
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
	})

	// Set version info
	vi := version.Info{
		FileVersion:    [4]uint16{1, 0, 0, 0},
		ProductVersion: [4]uint16{1, 0, 0, 0},
	}
	vi.Set(version.LangNeutral, version.ProductName, config.AppName)
	vi.Set(version.LangNeutral, version.FileDescription, config.AppName)
	vi.Set(version.LangNeutral, version.OriginalFilename, config.AppName+".exe")
	rs.SetVersionInfo(vi)

	// Write exe with resources to a temp file
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

	// On Windows, os.Rename cannot overwrite an existing file.
	// We need to remove the original first, then rename.
	if err := os.Remove(exePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("删除原文件失败: %w", err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	return nil
}

// replaceIconResource replaces all language variants of a group icon resource.
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

	// 先清掉同 ID 的旧语言资源，避免 Windows 加载到 base.exe 自带默认图标。
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

// copyOutputToDesktop lets user choose where to save the built exe
func (a *App) copyOutputToDesktop(srcPath, appName string) (string, error) {
	outputPath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "保存生成的应用",
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

// emitProgress sends a build progress event to the frontend
func (a *App) emitProgress(step string, progress int) {
	wailsRuntime.EventsEmit(a.ctx, "build:progress", map[string]interface{}{
		"step":     step,
		"progress": progress,
	})
}

// emitBuildComplete sends the build completion event with the output path
func (a *App) emitBuildComplete(outputPath string) {
	wailsRuntime.EventsEmit(a.ctx, "build:complete", map[string]interface{}{
		"outputPath": outputPath,
	})
}

// buildProxyConfigJSON builds the proxy_config.json content
func buildProxyConfigJSON(rules []ProxyRule) []byte {
	if rules == nil {
		rules = []ProxyRule{}
	}

	var buf bytes.Buffer
	buf.WriteString("{\n  \"rules\": [\n")
	for i, rule := range rules {
		buf.WriteString(fmt.Sprintf(`    {"path": %q, "target": %q, "rewrite": %q, "enabled": %t}`,
			rule.Path, rule.Target, rule.Rewrite, rule.Enabled))
		if i < len(rules)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("  ]\n}\n")

	return buf.Bytes()
}

// buildAppConfigJSON builds the app_config.json content
func buildAppConfigJSON(config BuildConfig) []byte {
	width := config.WindowWidth
	if width < 400 || width > 3840 {
		width = 1024
	}
	height := config.WindowHeight
	if height < 300 || height > 2160 {
		height = 768
	}

	return []byte(fmt.Sprintf(`{
  "width": %d,
  "height": %d,
  "fullscreen": %t,
  "maximized": %t
}
`, width, height, config.WindowFullscreen, config.WindowMaximized))
}

// addDirToZip recursively adds a directory to a zip writer
func addDirToZip(zw *zip.Writer, srcDir, zipPrefix string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Convert to forward slashes for zip
		zipName := zipPrefix + "/" + filepath.ToSlash(relPath)

		if d.IsDir() {
			// Add directory entry (with trailing slash)
			_, err := zw.Create(zipName + "/")
			return err
		}

		// Add file
		w, err := zw.Create(zipName)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		return err
	})
}

// copyFile copies a single file
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
