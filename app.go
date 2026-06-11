package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx      context.Context
	distPath string
	iconPath string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// confirmClose shows a native confirmation dialog before closing
func (a *App) confirmClose(ctx context.Context) bool {
	result, _ := wailsRuntime.MessageDialog(ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.QuestionDialog,
		Title:   "确认关闭",
		Message: "确定要关闭应用吗？",
	})
	return result != "Yes"
}

// OpenDistFolder opens a folder picker dialog and validates the selected dist folder
func (a *App) OpenDistFolder() (string, error) {
	path, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择前端构建产物文件夹",
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}

	// Validate index.html exists
	indexFile := filepath.Join(path, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		return "", fmt.Errorf("所选文件夹中未找到 index.html")
	}

	a.distPath = path
	return path, nil
}

// OpenTempFolder opens a folder picker dialog for temporary build files.
func (a *App) OpenTempFolder() (string, error) {
	path, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择临时文件目录",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// UploadDistZip opens a zip file picker, extracts to temp dir, and returns the path
func (a *App) UploadDistZip(tempPath string) (string, error) {
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择 ZIP 压缩包",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "ZIP 压缩包", Pattern: "*.zip"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}

	// ZIP 在第一步就会解压，必须优先使用用户指定目录，避免落到系统 C 盘 Temp。
	tempBase := strings.TrimSpace(tempPath)
	if tempBase == "" {
		tempBase = filepath.Dir(path)
	}
	if err := os.MkdirAll(tempBase, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	// Extract to temp directory
	tempDir, err := os.MkdirTemp(tempBase, "deploy-dist-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	err = extractZip(path, tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("解压失败: %w", err)
	}

	// If zip contains a single top-level folder, use that as root
	entries, _ := os.ReadDir(tempDir)
	if len(entries) == 1 && entries[0].IsDir() {
		tempDir = filepath.Join(tempDir, entries[0].Name())
	}

	// Validate index.html
	indexFile := filepath.Join(tempDir, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("压缩包中未找到 index.html")
	}

	a.distPath = tempDir
	return tempDir, nil
}

// GetDistInfo scans a directory and returns info about the dist files
func (a *App) GetDistInfo(path string) (*DistInfo, error) {
	info := &DistInfo{Path: path}

	indexFile := filepath.Join(path, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		return info, fmt.Errorf("目录中未找到 index.html")
	}

	var fileCount int
	var totalSize int64

	err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fileCount++
			fi, err := d.Info()
			if err == nil {
				totalSize += fi.Size()
			}
		}
		return nil
	})
	if err != nil {
		return info, err
	}

	info.FileCount = fileCount
	info.TotalSize = totalSize
	info.Valid = true
	return info, nil
}

// OpenIconFile opens a file picker for icon selection (.ico or .png)
func (a *App) OpenIconFile() (string, error) {
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择应用图标",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "图标文件", Pattern: "*.ico;*.png"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}

	a.iconPath = path
	return path, nil
}

// ValidateIcon validates the icon file and returns a base64 data URL for preview
func (a *App) ValidateIcon(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".ico" && ext != ".png" {
		return "", fmt.Errorf("不支持的图标格式: %s（请使用 .ico 或 .png）", ext)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取图标文件失败: %w", err)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("图标文件为空")
	}

	// Validate PNG dimensions
	if ext == ".png" {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return "", fmt.Errorf("无效的 PNG 文件: %w", err)
		}
		bounds := img.Bounds()
		if bounds.Dx() != bounds.Dy() {
			return "", fmt.Errorf("图标必须为正方形，当前尺寸: %dx%d", bounds.Dx(), bounds.Dy())
		}
		if bounds.Dx() < 64 {
			return "", fmt.Errorf("图标尺寸太小: %dx%d（建议至少 256x256）", bounds.Dx(), bounds.Dy())
		}
	}

	a.iconPath = path
	b64 := base64.StdEncoding.EncodeToString(data)
	mime := "image/png"
	if ext == ".ico" {
		mime = "image/x-icon"
	}
	return "data:" + mime + ";base64," + b64, nil
}

// OpenOutputFolder opens the output folder in Windows Explorer
func (a *App) OpenOutputFolder(path string) error {
	if path == "" {
		return fmt.Errorf("输出路径为空")
	}
	// Open the containing directory
	dir := filepath.Dir(path)
	return exec.Command("explorer", dir).Start()
}

// extractZip extracts a zip file to the destination directory
func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Prevent zip slip
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的 zip 路径: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
