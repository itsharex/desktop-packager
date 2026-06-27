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
// baseExe 嵌入预编译的基础可执行文件
// 这个文件是通过 cmd/build-base 工具生成的，包含了 Wails 运行时环境
var baseExe []byte

// resourceFooterMagic 资源尾部的魔数标识
// 用于在可执行文件中定位嵌入的 ZIP 数据
const resourceFooterMagic uint32 = 0x5245534F // "RESO" 的 ASCII 码

// resourceFooter 资源尾部结构体（8字节）
// 附加在 ZIP 数据之后，用于定位 ZIP 数据的起始位置
type resourceFooter struct {
	Magic  uint32 // 魔数标识，固定为 resourceFooterMagic
	Offset uint32 // ZIP 数据在可执行文件中的起始偏移量
}

// BuildApp 执行完整的应用构建流程
// 功能：
//   - 创建临时工作目录
//   - 复制基础可执行文件
//   - 修补应用图标（修改 PE 资源）
//   - 打包前端资源（ZIP 格式追加到 exe）
//   - 让用户选择保存位置并输出
// 参数:
//   - config: 构建配置，包含应用名称、图标路径、前端产物路径、代理规则等
// 返回值:
//   - error: 错误信息
func (a *App) BuildApp(config BuildConfig) error {
	// 步骤 1: 准备工作目录
	a.emitProgress("准备工作目录", 5)

	// 确定临时目录：优先使用用户指定的目录，否则使用前端产物所在目录
	tempBase := strings.TrimSpace(config.TempPath)
	if tempBase == "" {
		tempBase = filepath.Dir(config.DistPath)
	}
	if err := os.MkdirAll(tempBase, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	// 创建临时目录用于构建过程
	tempDir, err := os.MkdirTemp(tempBase, "deploy-build-*")
	if err != nil {
		return fmt.Errorf("创建工作目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir) // 构建完成后清理临时目录

	// 步骤 2: 复制基础可执行文件
	a.emitProgress("复制基础程序", 15)
	exePath := filepath.Join(tempDir, config.AppName+".exe")
	if err := os.WriteFile(exePath, baseExe, 0755); err != nil {
		return fmt.Errorf("复制基础程序失败: %w", err)
	}

	// 步骤 3: 修补应用图标
	// 注意：必须在追加资源之前完成，因为 WriteToEXE 会创建新的 PE 文件并丢弃追加的数据
	a.emitProgress("修补应用图标", 30)
	if err := a.patchIcon(exePath, config); err != nil {
		return fmt.Errorf("修补图标失败: %w", err)
	}

	// 步骤 4: 创建资源 ZIP 并追加到 exe
	a.emitProgress("打包前端资源", 60)
	if err := a.appendResources(exePath, config); err != nil {
		return fmt.Errorf("打包资源失败: %w", err)
	}

	// 步骤 5: 让用户选择保存位置并复制文件
	a.emitProgress("保存文件", 90)
	finalPath, err := a.copyOutputToDesktop(exePath, config.AppName)
	if err != nil {
		return fmt.Errorf("输出文件失败: %w", err)
	}

	// 构建完成，发送完成事件
	a.emitProgress("构建完成!", 100)
	a.emitBuildComplete(finalPath)

	return nil
}

// appendResources 创建包含前端资源的 ZIP 并追加到可执行文件
// 功能：
//   - 在内存中创建 ZIP 文件
//   - 添加代理配置文件 (proxy_config.json)
//   - 添加应用配置文件 (app_config.json)
//   - 添加前端构建产物 (dist 目录)
//   - 将 ZIP 数据追加到 exe 文件末尾
//   - 添加 8 字节的尾部标识用于定位 ZIP 数据
// 参数:
//   - exePath: 可执行文件路径
//   - config: 构建配置
// 返回值:
//   - error: 错误信息
func (a *App) appendResources(exePath string, config BuildConfig) error {
	// 在内存中构建 ZIP 文件
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)

	// 添加代理配置文件
	a.emitProgress("写入代理配置", 40)
	proxyJSON := buildProxyConfigJSON(config.ProxyRules)
	w, err := zw.Create("proxy_config.json")
	if err != nil {
		return fmt.Errorf("创建 zip 条目失败: %w", err)
	}
	if _, err := w.Write(proxyJSON); err != nil {
		return fmt.Errorf("写入代理配置失败: %w", err)
	}

	// 添加应用配置文件
	appJSON := buildAppConfigJSON(config)
	w, err = zw.Create("app_config.json")
	if err != nil {
		return fmt.Errorf("创建 zip 条目失败: %w", err)
	}
	if _, err := w.Write(appJSON); err != nil {
		return fmt.Errorf("写入应用配置失败: %w", err)
	}

	// 添加前端构建产物
	a.emitProgress("复制前端文件", 50)
	if err := addDirToZip(zw, config.DistPath, "dist"); err != nil {
		return fmt.Errorf("打包前端文件失败: %w", err)
	}

	// 关闭 ZIP 写入器
	if err := zw.Close(); err != nil {
		return fmt.Errorf("关闭 zip 失败: %w", err)
	}

	// 读取当前 exe 内容
	exeData, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("读取 exe 失败: %w", err)
	}

	// 创建输出文件（覆盖原文件）
	outFile, err := os.Create(exePath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	// 写入原始 exe 内容
	if _, err := outFile.Write(exeData); err != nil {
		return fmt.Errorf("写入 exe 内容失败: %w", err)
	}

	// 写入 ZIP 数据，并记录偏移量
	zipOffset := uint32(len(exeData))
	if _, err := outFile.Write(zipBuf.Bytes()); err != nil {
		return fmt.Errorf("写入 zip 数据失败: %w", err)
	}

	// 写入 8 字节尾部标识
	// 结构：[魔数 4字节][偏移量 4字节]
	footer := resourceFooter{
		Magic:  resourceFooterMagic,
		Offset: zipOffset,
	}
	if err := binary.Write(outFile, binary.LittleEndian, &footer); err != nil {
		return fmt.Errorf("写入 footer 失败: %w", err)
	}

	return nil
}

// patchIcon 修改可执行文件的 PE 资源，嵌入自定义图标、清单和版本信息
// 功能：
//   - 支持 .png 和 .ico 格式的图标
//   - 修改 Windows PE 文件的资源段
//   - 设置应用图标（ID 1 和 ID 3）
//   - 设置 DPI 感知和通用控件清单
//   - 设置版本信息（产品名称、文件描述等）
// 参数:
//   - exePath: 可执行文件路径
//   - config: 构建配置
// 返回值:
//   - error: 错误信息
func (a *App) patchIcon(exePath string, config BuildConfig) error {
	// 如果没有设置图标，直接返回
	if config.IconPath == "" {
		return nil
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(config.IconPath))
	var icon *winres.Icon

	// 根据图标格式加载图标
	if ext == ".png" {
		// 加载 PNG 图标
		f, err := os.Open(config.IconPath)
		if err != nil {
			return fmt.Errorf("读取 PNG 图标失败: %w", err)
		}
		defer f.Close()

		// 解码 PNG 图片
		img, _, err := image.Decode(f)
		if err != nil {
			return fmt.Errorf("解码 PNG 图标失败: %w", err)
		}

		// 从 PNG 图片创建图标
		icon, err = winres.NewIconFromResizedImage(img, nil)
		if err != nil {
			return fmt.Errorf("创建图标失败: %w", err)
		}
	} else if ext == ".ico" {
		// 加载 ICO 图标
		f, err := os.Open(config.IconPath)
		if err != nil {
			return fmt.Errorf("读取 ICO 图标失败: %w", err)
		}
		defer f.Close()

		// 加载 ICO 文件
		icon, err = winres.LoadICO(f)
		if err != nil {
			return fmt.Errorf("加载 ICO 图标失败: %w", err)
		}
	} else {
		return fmt.Errorf("不支持的图标格式: %s", ext)
	}

	// 读取可执行文件内容
	exeData, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("读取 exe 失败: %w", err)
	}

	// 加载现有的 PE 资源
	rs, err := winres.LoadFromEXE(bytes.NewReader(exeData))
	if err != nil {
		// 如果加载失败，创建新的资源集
		rs = &winres.ResourceSet{}
	}

	// 设置图标资源
	// Windows Shell 使用最小图标 ID (ID 1) 作为 exe 图标
	// Wails 窗口标题栏固定读取 ID 3
	// 必须替换旧语言资源，否则标题栏可能继续使用 base.exe 的默认图标
	if err := replaceIconResource(rs, winres.ID(1), icon); err != nil {
		return fmt.Errorf("设置图标资源失败: %w", err)
	}
	if err := replaceIconResource(rs, winres.ID(3), icon); err != nil {
		return fmt.Errorf("设置窗口图标资源失败: %w", err)
	}

	// 设置应用程序清单
	rs.SetManifest(winres.AppManifest{
		DPIAwareness:        winres.DPIPerMonitorV2, // 启用 Per-Monitor V2 DPI 感知
		UseCommonControlsV6: true,                   // 启用通用控件 v6
	})

	// 设置版本信息
	vi := version.Info{
		FileVersion:    [4]uint16{1, 0, 0, 0}, // 文件版本
		ProductVersion: [4]uint16{1, 0, 0, 0}, // 产品版本
	}
	vi.Set(version.LangNeutral, version.ProductName, config.AppName)              // 产品名称
	vi.Set(version.LangNeutral, version.FileDescription, config.AppName)          // 文件描述
	vi.Set(version.LangNeutral, version.OriginalFilename, config.AppName+".exe")  // 原始文件名
	rs.SetVersionInfo(vi)

	// 将修改后的资源写入临时文件
	tmpPath := exePath + ".tmp"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}

	// 写入 PE 资源
	if err := rs.WriteToEXE(outFile, bytes.NewReader(exeData)); err != nil {
		outFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("写入 PE 资源失败: %w", err)
	}
	outFile.Close()

	// Windows 下 os.Rename 不能覆盖已存在的文件
	// 需要先删除原文件，然后重命名
	if err := os.Remove(exePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("删除原文件失败: %w", err)
	}

	// 重命名临时文件为最终文件
	if err := os.Rename(tmpPath, exePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	return nil
}

// replaceIconResource 替换指定图标资源的所有语言变体
// 功能：
//   - 查找现有的图标资源语言变体
//   - 删除旧的语言资源（避免 Windows 加载 base.exe 的默认图标）
//   - 设置新的图标资源（包括中性和默认语言）
// 参数:
//   - rs: PE 资源集
//   - resID: 图标资源 ID（1 或 3）
//   - icon: 要设置的图标
// 返回值:
//   - error: 错误信息
func replaceIconResource(rs *winres.ResourceSet, resID winres.ID, icon *winres.Icon) error {
	// 默认语言 ID 列表
	langIDs := []uint16{winres.LCIDNeutral, winres.LCIDDefault}
	oldLangIDs := make([]uint16, 0)

	// 添加语言 ID 的辅助函数（避免重复）
	addLangID := func(langID uint16) {
		for _, existing := range langIDs {
			if existing == langID {
				return
			}
		}
		langIDs = append(langIDs, langID)
	}

	// 遍历现有的图标资源，收集语言 ID
	rs.WalkType(winres.RT_GROUP_ICON, func(existingID winres.Identifier, langID uint16, _ []byte) bool {
		id, ok := existingID.(winres.ID)
		if !ok || id != resID {
			return true
		}
		oldLangIDs = append(oldLangIDs, langID)
		addLangID(langID)
		return true
	})

	// 先删除同 ID 的旧语言资源，避免 Windows 加载到 base.exe 自带默认图标
	for _, langID := range oldLangIDs {
		if err := rs.Set(winres.RT_GROUP_ICON, resID, langID, nil); err != nil {
			return err
		}
	}

	// 设置新的图标资源（所有语言变体）
	for _, langID := range langIDs {
		if err := rs.SetIconTranslation(resID, langID, icon); err != nil {
			return err
		}
	}
	return nil
}

// copyOutputToDesktop 让用户选择保存位置并复制生成的 exe 文件
// 功能：
//   - 打开原生文件保存对话框
//   - 默认文件名为应用名称 + .exe
//   - 将生成的 exe 复制到用户选择的位置
// 参数:
//   - srcPath: 源文件路径（临时目录中的 exe）
//   - appName: 应用名称（用于默认文件名）
// 返回值:
//   - string: 最终保存的文件路径
//   - error: 错误信息
func (a *App) copyOutputToDesktop(srcPath, appName string) (string, error) {
	// 打开文件保存对话框
	outputPath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "保存生成的应用",                    // 对话框标题
		DefaultFilename: appName + ".exe",                // 默认文件名
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "可执行文件", Pattern: "*.exe"}, // 文件过滤器
		},
	})
	if err != nil {
		return "", err
	}
	// 用户取消保存
	if outputPath == "" {
		return "", fmt.Errorf("用户取消了保存")
	}

	// 复制文件到目标位置
	if err := copyFile(srcPath, outputPath); err != nil {
		return "", err
	}

	return outputPath, nil
}

// emitProgress 发送构建进度事件到前端
// 通过 Wails 事件系统向前端推送构建进度
// 参数:
//   - step: 当前步骤描述
//   - progress: 进度百分比（0-100）
func (a *App) emitProgress(step string, progress int) {
	wailsRuntime.EventsEmit(a.ctx, "build:progress", map[string]interface{}{
		"step":     step,     // 步骤描述
		"progress": progress, // 进度百分比
	})
}

// emitBuildComplete 发送构建完成事件到前端
// 通过 Wails 事件系统向前端推送构建完成信息
// 参数:
//   - outputPath: 输出文件路径
func (a *App) emitBuildComplete(outputPath string) {
	wailsRuntime.EventsEmit(a.ctx, "build:complete", map[string]interface{}{
		"outputPath": outputPath, // 输出文件路径
	})
}

// buildProxyConfigJSON 构建代理配置文件内容 (proxy_config.json)
// 功能：
//   - 将代理规则列表转换为 JSON 格式
//   - 使用手动构建的方式确保格式正确
// 参数:
//   - rules: 代理规则列表
// 返回值:
//   - []byte: JSON 格式的配置内容
func buildProxyConfigJSON(rules []ProxyRule) []byte {
	// 处理空规则列表
	if rules == nil {
		rules = []ProxyRule{}
	}

	// 手动构建 JSON（避免使用 json.Marshal 以确保格式一致）
	var buf bytes.Buffer
	buf.WriteString("{\n  \"rules\": [\n")
	for i, rule := range rules {
		// 使用 %q 确保字符串正确转义
		buf.WriteString(fmt.Sprintf(`    {"path": %q, "target": %q, "rewrite": %q, "enabled": %t}`,
			rule.Path, rule.Target, rule.Rewrite, rule.Enabled))
		// 如果不是最后一条规则，添加逗号
		if i < len(rules)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("  ]\n}\n")

	return buf.Bytes()
}

// buildAppConfigJSON 构建应用配置文件内容 (app_config.json)
// 功能：
//   - 验证窗口大小参数（最小 400x300，最大 3840x2160）
//   - 生成 JSON 格式的配置文件
// 参数:
//   - config: 构建配置
// 返回值:
//   - []byte: JSON 格式的配置内容
func buildAppConfigJSON(config BuildConfig) []byte {
	// 验证并限制窗口宽度
	width := config.WindowWidth
	if width < 400 || width > 3840 {
		width = 1024 // 默认宽度
	}

	// 验证并限制窗口高度
	height := config.WindowHeight
	if height < 300 || height > 2160 {
		height = 768 // 默认高度
	}

	// 生成 JSON 配置
	return []byte(fmt.Sprintf(`{
  "width": %d,
  "height": %d,
  "fullscreen": %t,
  "maximized": %t
}
`, width, height, config.WindowFullscreen, config.WindowMaximized))
}

// addDirToZip 递归地将目录添加到 ZIP 写入器
// 功能：
//   - 遍历源目录中的所有文件和子目录
//   - 将路径转换为正斜杠格式（ZIP 标准）
//   - 添加目录条目（带尾部斜杠）
//   - 添加文件内容
// 参数:
//   - zw: ZIP 写入器
//   - srcDir: 源目录路径
//   - zipPrefix: ZIP 内的路径前缀
// 返回值:
//   - error: 错误信息
func addDirToZip(zw *zip.Writer, srcDir, zipPrefix string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// 转换为正斜杠格式（ZIP 标准）
		zipName := zipPrefix + "/" + filepath.ToSlash(relPath)

		if d.IsDir() {
			// 添加目录条目（带尾部斜杠）
			_, err := zw.Create(zipName + "/")
			return err
		}

		// 添加文件
		w, err := zw.Create(zipName)
		if err != nil {
			return err
		}

		// 打开源文件
		f, err := os.Open(path)
		if err != nil {
			return err
		}

		// 复制文件内容到 ZIP
		_, err = io.Copy(w, f)
		f.Close() // 立即关闭文件，不要在循环中使用 defer
		return err
	})
}

// copyFile 复制单个文件
// 功能：
//   - 打开源文件
//   - 创建目标文件的父目录
//   - 创建目标文件
//   - 复制文件内容
// 参数:
//   - src: 源文件路径
//   - dest: 目标文件路径
// 返回值:
//   - error: 错误信息
func copyFile(src, dest string) error {
	// 打开源文件
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// 创建目标文件的父目录
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	// 创建目标文件
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// 复制文件内容
	_, err = io.Copy(out, in)
	return err
}
