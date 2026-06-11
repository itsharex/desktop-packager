// build-base is a development tool that pre-compiles the base Wails application
// shell into an exe. This exe is then embedded into the deploy-app so that
// end users don't need a Go environment to build their apps.
//
// Usage: go run ./cmd/build-base
//
// The output is written to templates/base/base.exe
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

type templateData struct {
	AppName    string
	Width      int
	Height     int
	Fullscreen bool
	Maximized  bool
}

func main() {
	// When run via "go run ./cmd/build-base" from project root,
	// the working directory is already the project root.
	projectRoot, err := os.Getwd()
	if err != nil {
		fatal("获取工作目录失败: %v", err)
	}

	templateDir := filepath.Join(projectRoot, "templates", "generated-app")
	outputDir := filepath.Join(projectRoot, "templates", "base")
	outputExe := filepath.Join(outputDir, "base.exe")

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fatal("创建输出目录失败: %v", err)
	}

	// Create temp working directory
	tempDir, err := os.MkdirTemp("", "build-base-*")
	if err != nil {
		fatal("创建工作目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Println("工作目录:", tempDir)

	// Template data with placeholder values
	data := templateData{
		AppName:    "BaseApp",
		Width:      1024,
		Height:     768,
		Fullscreen: false,
		Maximized:  false,
	}

	// Render templates
	templates := []string{"main.go", "app.go", "loader.go", "error_windows.go", "go.mod"}
	for _, name := range templates {
		tmplPath := filepath.Join(templateDir, name+".tmpl")
		outputPath := filepath.Join(tempDir, name)

		if err := renderTemplate(tmplPath, outputPath, data); err != nil {
			fatal("渲染模板 %s 失败: %v", name, err)
		}
		fmt.Printf("  ✓ 渲染 %s\n", name)
	}

	// Create an empty proxy_config.json (base app has no proxy rules)
	proxyConfig := `{"rules": []}`
	if err := os.WriteFile(filepath.Join(tempDir, "proxy_config.json"), []byte(proxyConfig), 0644); err != nil {
		fatal("写入 proxy_config.json 失败: %v", err)
	}

	// Create an empty dist directory (base app has no frontend)
	distDir := filepath.Join(tempDir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		fatal("创建 dist 目录失败: %v", err)
	}
	// Write a placeholder index.html so embed has something to work with
	placeholder := `<!DOCTYPE html><html><body><h1>Resource not found</h1></body></html>`
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte(placeholder), 0644); err != nil {
		fatal("写入 placeholder 失败: %v", err)
	}

	// Run go mod tidy
	fmt.Println("运行 go mod tidy...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("go mod tidy 失败: %v", err)
	}

	// Run go build
	fmt.Println("编译 base.exe...")
	cmd = exec.Command("go", "build",
		"-tags", "production",
		"-ldflags", "-w -s -H windowsgui",
		"-o", outputExe,
		".",
	)
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("go build 失败: %v", err)
	}

	// Get file size
	stat, err := os.Stat(outputExe)
	if err != nil {
		fatal("获取输出文件信息失败: %v", err)
	}

	fmt.Printf("\n✓ base.exe 已生成: %s (%.1f MB)\n", outputExe, float64(stat.Size())/(1024*1024))
}

func renderTemplate(tmplPath, outputPath string, data interface{}) error {
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		return err
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(content))
	if err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "错误: "+format+"\n", args...)
	os.Exit(1)
}
