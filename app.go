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

// App 应用主结构体
// 包含应用运行时的上下文和状态信息
type App struct {
	ctx      context.Context // Wails 运行时上下文，用于调用运行时 API
	distPath string          // 前端构建产物目录路径（dist 文件夹）
	iconPath string          // 应用图标文件路径（.ico 或 .png）
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup 应用启动时的回调函数
// 在 Wails 应用初始化完成后调用，保存运行时上下文
// 参数:
//   - ctx: Wails 运行时上下文，用于后续调用窗口API、事件系统等
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// confirmClose 关闭窗口前的确认对话框
// 当用户点击关闭按钮时触发，显示原生确认对话框
// 返回值:
//   - true: 阻止关闭（用户点击"取消"或关闭对话框）
//   - false: 允许关闭（用户点击"确定"）
func (a *App) confirmClose(ctx context.Context) bool {
	// 显示原生确认对话框
	result, _ := wailsRuntime.MessageDialog(ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.QuestionDialog, // 对话框类型：问题
		Title:   "确认关闭",                   // 对话框标题
		Message: "确定要关闭应用吗？",            // 对话框内容
	})
	// 如果用户点击"Yes"，返回false表示允许关闭；否则返回true阻止关闭
	return result != "Yes"
}

// OpenDistFolder 打开文件夹选择对话框，让用户选择前端构建产物目录
// 功能：
//   - 弹出原生文件夹选择对话框
//   - 验证所选文件夹是否包含 index.html（前端应用的入口文件）
//   - 保存路径到 App 实例中
// 返回值:
//   - string: 选择的文件夹路径，取消选择时返回空字符串
//   - error: 错误信息，验证失败时返回错误
func (a *App) OpenDistFolder() (string, error) {
	// 打开原生文件夹选择对话框
	path, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择前端构建产物文件夹", // 对话框标题
	})
	if err != nil {
		return "", err
	}
	// 用户取消选择，返回空字符串
	if path == "" {
		return "", nil
	}

	// 验证所选文件夹是否包含 index.html
	indexFile := filepath.Join(path, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		return "", fmt.Errorf("所选文件夹中未找到 index.html")
	}

	// 保存路径到实例
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

// UploadDistZip 打开 ZIP 文件选择对话框，解压并返回解压后的路径
// 功能：
//   - 弹出原生文件选择对话框，限制为 .zip 文件
//   - 将 ZIP 文件解压到临时目录
//   - 智能处理 ZIP 结构（支持单文件夹或直接包含文件的格式）
//   - 验证解压后的内容是否包含 index.html
// 参数:
//   - tempPath: 用户指定的临时目录路径，为空时使用 ZIP 文件所在目录
// 返回值:
//   - string: 解压后的目录路径
//   - error: 错误信息
func (a *App) UploadDistZip(tempPath string) (string, error) {
	// 打开文件选择对话框
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择 ZIP 压缩包", // 对话框标题
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "ZIP 压缩包", Pattern: "*.zip"}, // 文件过滤器
		},
	})
	if err != nil {
		return "", err
	}
	// 用户取消选择
	if path == "" {
		return "", nil
	}

	// 确定临时目录：优先使用用户指定目录，否则使用 ZIP 文件所在目录
	// 这样可以避免解压到系统 C 盘 Temp 目录
	tempBase := strings.TrimSpace(tempPath)
	if tempBase == "" {
		tempBase = filepath.Dir(path)
	}
	if err := os.MkdirAll(tempBase, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 创建临时目录用于解压
	tempDir, err := os.MkdirTemp(tempBase, "deploy-dist-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 解压 ZIP 文件
	err = extractZip(path, tempDir)
	if err != nil {
		os.RemoveAll(tempDir) // 解压失败时清理临时目录
		return "", fmt.Errorf("解压失败: %w", err)
	}

	// 智能处理 ZIP 结构：如果 ZIP 内只有一个文件夹，使用该文件夹作为根目录
	// 例如：ZIP 内有 dist/ 文件夹，则使用 dist/ 作为根目录
	entries, _ := os.ReadDir(tempDir)
	if len(entries) == 1 && entries[0].IsDir() {
		tempDir = filepath.Join(tempDir, entries[0].Name())
	}

	// 验证解压后的内容是否包含 index.html（前端应用入口文件）
	indexFile := filepath.Join(tempDir, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		os.RemoveAll(tempDir) // 验证失败时清理临时目录
		return "", fmt.Errorf("压缩包中未找到 index.html")
	}

	// 保存路径到实例
	a.distPath = tempDir
	return tempDir, nil
}

// GetDistInfo 扫描目录并返回构建产物的详细信息
// 功能：
//   - 验证目录是否包含 index.html
//   - 统计目录中的文件数量
//   - 计算所有文件的总大小
// 参数:
//   - path: 要扫描的目录路径
// 返回值:
//   - *DistInfo: 包含目录信息的结构体（路径、文件数、总大小、是否有效）
//   - error: 错误信息
func (a *App) GetDistInfo(path string) (*DistInfo, error) {
	// 创建信息结构体
	info := &DistInfo{Path: path}

	// 验证 index.html 是否存在
	indexFile := filepath.Join(path, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		return info, fmt.Errorf("目录中未找到 index.html")
	}

	// 统计文件数量和总大小
	var fileCount int
	var totalSize int64

	// 递归遍历目录
	err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() { // 只统计文件，不统计目录
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

	// 填充信息并返回
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

// ValidateIcon 验证图标文件并返回 base64 编码的预览 URL
// 功能：
//   - 验证文件格式是否为 .ico 或 .png
//   - 验证文件是否为空
//   - 对于 PNG 文件，验证是否为正方形且尺寸不小于 64x64
//   - 将图标文件转换为 base64 格式用于前端预览
// 参数:
//   - path: 图标文件路径
// 返回值:
//   - string: base64 编码的 data URL（格式：data:image/xxx;base64,...）
//   - error: 错误信息
func (a *App) ValidateIcon(path string) (string, error) {
	// 获取文件扩展名并验证格式
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".ico" && ext != ".png" {
		return "", fmt.Errorf("不支持的图标格式: %s（请使用 .ico 或 .png）", ext)
	}

	// 读取文件内容
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取图标文件失败: %w", err)
	}

	// 验证文件是否为空
	if len(data) == 0 {
		return "", fmt.Errorf("图标文件为空")
	}

	// 对于 PNG 文件，验证尺寸
	if ext == ".png" {
		// 解码 PNG 图片
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return "", fmt.Errorf("无效的 PNG 文件: %w", err)
		}
		bounds := img.Bounds()
		// 验证是否为正方形
		if bounds.Dx() != bounds.Dy() {
			return "", fmt.Errorf("图标必须为正方形，当前尺寸: %dx%d", bounds.Dx(), bounds.Dy())
		}
		// 验证最小尺寸
		if bounds.Dx() < 64 {
			return "", fmt.Errorf("图标尺寸太小: %dx%d（建议至少 256x256）", bounds.Dx(), bounds.Dy())
		}
	}

	// 保存图标路径
	a.iconPath = path

	// 转换为 base64 编码
	b64 := base64.StdEncoding.EncodeToString(data)
	// 根据文件类型设置 MIME 类型
	mime := "image/png"
	if ext == ".ico" {
		mime = "image/x-icon"
	}
	// 返回 data URL 格式
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

// extractZip 解压 ZIP 文件到目标目录
// 功能：
//   - 打开并解析 ZIP 文件
//   - 防止 Zip Slip 路径穿越攻击
//   - 递归解压所有文件和目录
//   - 保持原始文件权限
// 参数:
//   - src: ZIP 文件路径
//   - dest: 目标目录路径
// 返回值:
//   - error: 错误信息
func extractZip(src, dest string) error {
	// 打开 ZIP 文件
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// 遍历 ZIP 中的所有文件
	for _, f := range r.File {
		// 构建目标文件路径
		fpath := filepath.Join(dest, f.Name)

		// 安全检查：防止 Zip Slip 路径穿越攻击
		// 确保解压后的路径在目标目录内
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的 zip 路径: %s", f.Name)
		}

		// 如果是目录，创建目录
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// 创建文件的父目录
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// 创建目标文件，保持原始文件权限
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		// 打开 ZIP 中的文件
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		// 将 ZIP 文件内容复制到目标文件
		_, err = io.Copy(outFile, rc)

		// 关闭文件句柄
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
