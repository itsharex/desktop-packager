# Deploy App

![Platform](https://img.shields.io/badge/platform-Windows-blue)
![Wails](https://img.shields.io/badge/Wails-v2-ff69b4)
![Vue](https://img.shields.io/badge/Vue-3-42b883)
![License](https://img.shields.io/badge/license-MIT-green)

Deploy App 是一个基于 Wails 的 Windows 桌面应用打包工具。它可以把已经构建好的前端项目（Vue、React、Angular、静态站点等）打包成独立的 `.exe` 应用，并支持自定义图标、窗口大小和反向代理规则。

这个项目的目标很直接：让前端应用在不改业务代码的情况下，快速生成一个可分发的 Windows 桌面程序。

## 界面预览

### 导入构建产物

![导入构建产物](assets/img/step-1.png)

### 应用配置

![应用配置](assets/img/step-2.png)

### 反向代理配置

![反向代理配置](assets/img/step-3.png)

### 构建生成

![构建生成](assets/img/step-4.png)

## 功能特性

- 单文件输出：生成一个独立的 `.exe` 文件，便于复制和分发。
- 零 Go 依赖打包：最终用户使用打包工具时，不需要安装 Go 环境。
- 自定义应用图标：支持 `.ico` 和 `.png` 图标，并同步到 exe 图标和窗口标题栏图标。
- 窗口配置：支持窗口宽度、高度、最大化、全屏，以及关闭前确认。
- 版本信息：支持写入版本号、描述、公司/组织到 PE 版本资源。
- 反向代理：内置与 nginx `location` + `proxy_pass` 对齐的路径语义，适合接口跨域或本地服务转发。
- SPA 路由回退：前端 History 路由刷新未知路径时回退到 `index.html`。
- 静态资源嵌入：前端 `dist` 文件会被打进 exe 内部，运行时自动加载。
- ZIP 导入：支持直接选择 `dist` 文件夹，也支持上传构建产物 ZIP 包（带 Zip Slip 防护与临时目录清理）。

## 适用场景

- 将内部管理后台快速封装为 Windows 桌面应用。
- 给纯前端项目生成可双击运行的交付包。
- 需要把前端静态资源和少量代理配置打包进一个 exe。
- 希望分发给非技术用户，不要求对方安装 Node.js、Go 或命令行工具。

## 快速使用

### 使用发布版本

1. 从 GitHub Releases 下载最新的 `deploy-app.exe`。
2. 双击运行。
3. 导入前端项目的 `dist` 目录，或上传包含构建产物的 ZIP 包。
4. 设置应用名称、图标、窗口大小和代理规则。
5. 点击“开始构建”，选择保存位置，生成最终 exe。

### ZIP 包格式

支持两种常见结构：

```text
dist.zip
  index.html
  assets/
  ...
```

```text
dist.zip
  dist/
    index.html
    assets/
    ...
```

## 从源码运行

### 环境要求

- Windows 10/11 64-bit
- Go 1.23+
- Node.js 20+
- Wails CLI v2

### 安装依赖

```bash
cd frontend
npm install
cd ..
```

### 启动开发模式

```bash
wails dev
```

### 构建前端

```bash
cd frontend
npm run build
cd ..
```

### 重新生成基础运行壳（重要）

修改 `templates/generated-app/` 后，必须重新生成基础 exe，否则打包工具仍会使用旧壳：

```bash
go run ./cmd/build-base
```

输出位置：

```text
templates/base/base.exe
```

> 注意：仓库中的 `templates/base/base.exe` 需要是真实可运行的 Wails 壳。如果只有占位文件，请先执行上面的命令生成。

### 构建 Deploy App

```bash
wails build
```

构建产物默认输出到：

```text
build/bin/deploy-app.exe
```

## 使用流程

### 1. 导入前端构建产物

选择前端项目构建后的 `dist` 文件夹，或上传 ZIP 压缩包。目录中必须包含 `index.html`。

可在全局设置中指定临时目录；未设置时，ZIP 解压和构建临时文件会落在导入目录附近，并在关闭应用时清理。

### 2. 配置应用信息

| 字段 | 说明 |
| --- | --- |
| 应用名称 | 作为 exe 文件名，禁止 Windows 非法字符与保留名（如 `CON`） |
| 版本号 | 如 `1.0.0`，写入 PE 版本信息 |
| 描述 | 可选，写入文件说明（FileDescription） |
| 公司/组织 | 可选，写入公司名，并生成详细信息中的版权（LegalCopyright） |
| 图标 | `.ico` 或正方形 `.png`（建议 ≥256） |
| 窗口 | 宽高 / 最大化 / 全屏 |
| 关闭前确认 | 生成应用退出时是否弹确认框 |

### 3. 配置反向代理（与 nginx 对齐）

代理规则对应 nginx 的：

```nginx
location <路径前缀> {
    proxy_pass <目标地址>;
}
```

“重写为”非空时，等价于覆盖 `proxy_pass` 的 URI 部分。

#### 路径处理规则

| 配置方式 | nginx 对应 | 行为 | 示例 |
| --- | --- | --- | --- |
| 目标地址**无路径** | `proxy_pass http://host:8080;` | 保留完整请求路径 | `/api/users` → `/api/users` |
| 目标地址以 `/` 结尾 | `proxy_pass http://host:8080/;` | 剥离 location 前缀后拼接 | `/api/users` → `/users` |
| 目标地址带前缀路径 | `proxy_pass http://host:8080/v2/;` | 用该路径替换 location 前缀 | `/api/users` → `/v2/users` |
| 设置“重写为” | 覆盖 proxy_pass URI | 用重写前缀替换 location 前缀 | 重写 `/v2`：`/api/users` → `/v2/users` |

示例：

```text
路径前缀: /api/
目标地址: http://localhost:8080/
重写为:   留空
```

当前端请求 `/api/users` 时，会代理到 `http://localhost:8080/users`。

再如：

```text
路径前缀: /api/
目标地址: http://localhost:8080
重写为:   /v2
```

请求 `/api/users` → `http://localhost:8080/v2/users`（注意是 `/v2/users` 而不是 `/v2users`）。

#### 请求头

生成应用的反向代理会设置：

- `Host` = 上游 host（类似 `$proxy_host`）
- `X-Forwarded-For` / `X-Real-IP`
- `X-Forwarded-Proto`
- `X-Forwarded-Host`（原始 Host）

并配置合理超时，支持长连接与 WebSocket 升级场景。

### 4. 构建生成

确认配置后点击“开始构建”，工具会：

1. 复制基础运行壳 `base.exe`
2. 写入图标与版本资源
3. 追加资源 zip（`dist/` + `proxy_config.json` + `app_config.json`）与 footer
4. 弹出保存对话框输出最终 exe

## 工作原理

Deploy App 内置一个预编译的 Wails 基础运行壳 `base.exe`。构建时不会现场编译用户应用，而是把前端资源和配置追加到 `base.exe` 尾部：

```text
base.exe
  +
resource.zip
  dist/
  proxy_config.json
  app_config.json
  +
footer
  magic:  RESO
  offset: zip 起始位置
```

生成的 exe 启动后会从自身文件末尾读取资源 zip，懒加载 `dist`（不把整个 zip 读进内存），启用 SPA fallback，并启动内置代理。

## 项目结构

```text
.
├── app.go                    # Wails 后端绑定方法
├── builder.go                # 打包流程、资源追加、图标/版本修补
├── config.go                 # 构建配置结构
├── validate.go               # 应用名 / 代理规则 / 构建配置校验
├── main.go                   # Deploy App 入口
├── internal/nginxproxy/      # 与生成壳共用的 nginx 路径算法
├── cmd/build-base/           # 生成基础运行壳的工具
├── frontend/                 # Vue 前端界面
├── templates/base/           # 预编译基础 exe（gitignore 例外保留）
├── templates/generated-app/  # 基础运行壳源码模板
└── assets/img/               # README 截图资源
```

## 开发注意

- 修改 `templates/generated-app/*.tmpl` 后必须执行 `go run ./cmd/build-base`。
- `templates/base/base.exe` 在 `.gitignore` 中通过 `!templates/base/base.exe` 例外保留，避免被 `*.exe` 规则忽略。
- `internal/nginxproxy` 与 `templates/generated-app/proxy.go.tmpl` 中的路径算法需保持同步。
- 前端步骤有前置校验：未导入 dist / 应用名非法时不能跳步构建。

## 常见问题

### 生成的 exe 打开后提示“加载资源失败”

通常是 exe 被损坏、被二次修改，构建过程没有完整写入资源，或 `base.exe` 仍是占位文件。请先 `go run ./cmd/build-base`，再重新构建。

### 反向代理不生效

检查：

1. 路径前缀是否以 `/` 开头，例如 `/api/`。
2. 目标地址是否为 `http://` 或 `https://`。
3. 目标地址末尾是否带 `/`：这会决定是否剥离路径前缀（与 nginx 相同）。
4. “重写为”若填写，必须以 `/` 开头，且会覆盖目标地址中的路径部分。

### 图标没有生效

优先使用标准 `.ico` 文件，或使用 256x256 以上的正方形 `.png`。如果 Windows 资源管理器仍显示旧图标，可能是系统图标缓存导致，可以换一个输出文件名后再查看。

### History 路由刷新 404

生成应用已启用 SPA fallback：无扩展名路径或 `.html` 在资源中不存在时回退 `index.html`；静态资源扩展名（`.js` / `.css` / 图片字体等）仍返回真实 404。

### 是否支持 macOS 或 Linux

当前只支持 Windows。要支持其他平台，需要分别准备对应平台的基础运行壳和资源写入逻辑。

## 贡献

欢迎提交 Issue 和 Pull Request。建议在提交前先说明要解决的问题、使用场景和预期行为，方便保持功能边界清晰。

本项目偏工具型应用，代码修改建议遵循这些原则：

- 优先保持实现简单直接。
- UI 交互以清晰、稳定、可重复操作为主。
- 新功能尽量补充 README 或界面说明。
- 修改模板后请重新运行 `go run ./cmd/build-base`。
- 路径代理逻辑变更时，同步更新 `internal/nginxproxy` 与 `proxy.go.tmpl`，并补充测试用例。

## 许可证

本项目基于 [MIT License](LICENSE) 开源。