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
	"sync"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 应用主结构体
type App struct {
	ctx          context.Context
	mu           sync.Mutex
	building     bool
	lastProgress int
	tempDirs     map[string]struct{}
}

func NewApp() *App {
	return &App{
		tempDirs: make(map[string]struct{}),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	a.cleanupTempDirs()
}

func (a *App) confirmClose(ctx context.Context) bool {
	result, err := wailsRuntime.MessageDialog(ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.QuestionDialog,
		Title:   "确认关闭",
		Message: "确定要关闭应用吗？",
	})
	if err != nil {
		// 对话框失败时允许关闭，避免卡死
		return false
	}
	return result != "Yes" && result != "Ok" && result != "OK"
}

func (a *App) beginBuild() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.building {
		return false
	}
	a.building = true
	a.lastProgress = 0
	return true
}

func (a *App) endBuild() {
	a.mu.Lock()
	a.building = false
	a.mu.Unlock()
}

func (a *App) trackTempDir(dir string) {
	if dir == "" {
		return
	}
	a.mu.Lock()
	a.tempDirs[dir] = struct{}{}
	a.mu.Unlock()
}

func (a *App) untrackTempDir(dir string) {
	a.mu.Lock()
	delete(a.tempDirs, dir)
	a.mu.Unlock()
}

func (a *App) cleanupTempDirs() {
	a.mu.Lock()
	dirs := make([]string, 0, len(a.tempDirs))
	for d := range a.tempDirs {
		dirs = append(dirs, d)
	}
	a.tempDirs = make(map[string]struct{})
	a.mu.Unlock()
	for _, d := range dirs {
		_ = os.RemoveAll(d)
	}
}

// OpenDistFolder 选择前端构建产物目录
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
	indexFile := filepath.Join(path, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		return "", fmt.Errorf("所选文件夹中未找到 index.html")
	}
	return path, nil
}

// OpenTempFolder 选择临时目录
func (a *App) OpenTempFolder() (string, error) {
	path, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择临时文件目录",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// UploadDistZip 选择并解压 ZIP 构建产物
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

	tempBase := strings.TrimSpace(tempPath)
	if tempBase == "" {
		tempBase = filepath.Dir(path)
	}
	if err := os.MkdirAll(tempBase, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	// Always track the outer temp root for cleanup.
	tempRoot, err := os.MkdirTemp(tempBase, "deploy-dist-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	a.trackTempDir(tempRoot)

	if err := extractZip(path, tempRoot); err != nil {
		_ = os.RemoveAll(tempRoot)
		a.untrackTempDir(tempRoot)
		return "", fmt.Errorf("解压失败: %w", err)
	}

	distDir := tempRoot
	entries, err := os.ReadDir(tempRoot)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		a.untrackTempDir(tempRoot)
		return "", fmt.Errorf("读取解压目录失败: %w", err)
	}
	if len(entries) == 1 && entries[0].IsDir() {
		distDir = filepath.Join(tempRoot, entries[0].Name())
	}

	indexFile := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		_ = os.RemoveAll(tempRoot)
		a.untrackTempDir(tempRoot)
		return "", fmt.Errorf("压缩包中未找到 index.html")
	}
	return distDir, nil
}

// GetDistInfo 扫描构建产物目录
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

// OpenIconFile 选择图标文件
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
	return path, nil
}

// ValidateIcon 验证图标并返回 base64 预览
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
	} else {
		// Basic ICO header validation: reserved(0), type(1), count>0
		if len(data) < 6 {
			return "", fmt.Errorf("无效的 ICO 文件")
		}
		if data[0] != 0 || data[1] != 0 || data[2] != 1 || data[3] != 0 {
			return "", fmt.Errorf("无效的 ICO 文件头")
		}
		count := int(data[4]) | int(data[5])<<8
		if count <= 0 {
			return "", fmt.Errorf("ICO 文件不包含图标图像")
		}
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	mime := "image/png"
	if ext == ".ico" {
		mime = "image/x-icon"
	}
	return "data:" + mime + ";base64," + b64, nil
}

// OpenOutputFolder 在资源管理器中打开输出目录
func (a *App) OpenOutputFolder(path string) error {
	if path == "" {
		return fmt.Errorf("输出路径为空")
	}
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("输出目录不存在: %w", err)
	}
	cmd := exec.Command("explorer", dir)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开目录失败: %w", err)
	}
	return nil
}

// SanitizeName 供前端预校验应用名
func (a *App) SanitizeName(name string) (string, error) {
	return SanitizeAppName(name)
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Reject absolute paths and path traversal (Zip Slip).
		name := filepath.ToSlash(f.Name)
		name = strings.TrimPrefix(name, "./")
		if name == "" || !filepath.IsLocal(filepath.FromSlash(name)) {
			return fmt.Errorf("非法的 zip 路径: %s", f.Name)
		}

		fpath := filepath.Join(dest, filepath.FromSlash(name))
		rel, err := filepath.Rel(dest, fpath)
		if err != nil || !filepath.IsLocal(rel) {
			return fmt.Errorf("非法的 zip 路径: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
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