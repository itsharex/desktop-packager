package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
// assets 嵌入前端构建产物（Vue 打包后的静态文件）
// 使用 Go 1.16+ 的 embed 指令，将 frontend/dist 目录嵌入到二进制文件中
var assets embed.FS

// main 程序入口函数
// 创建 Wails 桌面应用实例并启动
func main() {
	// 创建应用实例
	app := NewApp()

	// 配置并启动 Wails 应用
	err := wails.Run(&options.App{
		Title:  "桌面应用生成器",      // 窗口标题
		Width:  1024,                 // 窗口宽度
		Height: 768,                  // 窗口高度
		AssetServer: &assetserver.Options{
			Assets: assets,           // 嵌入的前端静态资源
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1}, // 窗口背景色（深蓝色）
		OnStartup:        app.startup,                              // 应用启动时的回调函数
		OnBeforeClose:    app.confirmClose,                         // 关闭窗口前的确认回调
		Bind: []interface{}{
			app, // 绑定 App 实例，使其方法可被前端调用
		},
	})

	// 如果启动失败，输出错误信息
	if err != nil {
		println("Error:", err.Error())
	}
}
