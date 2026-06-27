import {reactive, readonly} from 'vue'

/**
 * ProxyRule 代理规则接口
 * 定义单条反向代理规则的结构
 */
export interface ProxyRule {
  path: string      // 路径前缀，例如 "/api/"
  target: string    // 目标服务器地址，例如 "http://localhost:8080/"
  rewrite: string   // 路径重写规则（可选）
  enabled: boolean  // 是否启用此规则
}

/**
 * AppState 应用状态接口
 * 定义整个应用的状态结构，分为以下几个部分：
 * 1. 导入配置（dist 产物）
 * 2. 应用设置（名称、图标）
 * 3. 代理规则
 * 4. 构建状态
 * 5. 导航状态
 * 6. 窗口设置
 */
export interface AppState {
  // 步骤 1: 导入构建产物
  distPath: string        // 前端构建产物目录路径
  distFileCount: number   // 文件数量
  distTotalSize: number   // 文件总大小（字节）

  // 步骤 2: 应用设置
  appName: string         // 应用名称
  iconPath: string        // 图标文件路径
  iconPreview: string     // 图标预览的 base64 URL
  tempPath: string        // 临时文件目录路径

  // 步骤 3: 代理规则
  proxyRules: ProxyRule[] // 反向代理规则列表

  // 步骤 4: 构建状态
  building: boolean       // 是否正在构建
  buildProgress: number   // 构建进度（0-100）
  buildStep: string       // 当前构建步骤描述
  outputPath: string      // 输出文件路径
  buildError: string      // 构建错误信息

  // 导航状态
  currentStep: number     // 当前步骤索引（0-3）

  // 生成应用的窗口设置
  windowWidth: number     // 窗口宽度（像素）
  windowHeight: number    // 窗口高度（像素）
  windowFullscreen: boolean // 是否全屏
  windowMaximized: boolean  // 是否最大化
}

/**
 * 应用全局状态
 * 使用 Vue 3 的 reactive 创建响应式状态对象
 * 所有组件都可以通过 useStore() 访问和修改此状态
 */
const state = reactive<AppState>({
  // 步骤 1: 导入配置
  distPath: '',           // 前端构建产物路径
  distFileCount: 0,       // 文件数量
  distTotalSize: 0,       // 文件总大小

  // 步骤 2: 应用设置
  appName: '',            // 应用名称
  iconPath: '',           // 图标路径
  iconPreview: '',        // 图标预览 URL
  tempPath: '',           // 临时目录路径

  // 步骤 3: 代理规则
  proxyRules: [],         // 代理规则列表

  // 步骤 4: 构建状态
  building: false,        // 是否正在构建
  buildProgress: 0,       // 构建进度
  buildStep: '',          // 当前步骤
  outputPath: '',         // 输出路径
  buildError: '',         // 错误信息

  // 导航
  currentStep: 0,         // 当前步骤（0-3）

  // 窗口设置
  windowWidth: 1024,      // 默认宽度
  windowHeight: 768,      // 默认高度
  windowFullscreen: false, // 默认不全屏
  windowMaximized: false,  // 默认不最大化
})

/**
 * useStore 应用状态管理 Hook
 * 返回只读状态和修改状态的方法
 * 所有组件通过此函数访问全局状态
 *
 * 使用示例：
 * ```ts
 * const store = useStore()
 * store.setAppName('My App')
 * console.log(store.state.appName) // 'My App'
 * ```
 */
export function useStore() {
  return {
    // 只读状态（防止组件直接修改）
    state: readonly(state),

    // ========== 步骤 1: 导入配置 ==========

    /**
     * 设置前端构建产物信息
     * @param path - 构建产物目录路径
     * @param fileCount - 文件数量
     * @param totalSize - 文件总大小（字节）
     */
    setDist(path: string, fileCount: number, totalSize: number) {
      state.distPath = path
      state.distFileCount = fileCount
      state.distTotalSize = totalSize
    },

    /**
     * 清除构建产物信息
     * 用于重新选择构建产物时
     */
    clearDist() {
      state.distPath = ''
      state.distFileCount = 0
      state.distTotalSize = 0
    },

    // ========== 步骤 2: 应用设置 ==========

    /**
     * 设置应用名称
     * @param name - 应用名称
     */
    setAppName(name: string) {
      state.appName = name
    },

    /**
     * 设置应用图标
     * @param path - 图标文件路径
     * @param preview - 图标的 base64 预览 URL
     */
    setIcon(path: string, preview: string) {
      state.iconPath = path
      state.iconPreview = preview
    },

    /**
     * 清除应用图标
     * 用于移除已选择的图标
     */
    clearIcon() {
      state.iconPath = ''
      state.iconPreview = ''
    },

    /**
     * 设置临时文件目录路径
     * @param path - 临时目录路径
     */
    setTempPath(path: string) {
      state.tempPath = path
    },

    // ========== 步骤 3: 代理规则 ==========

    /**
     * 添加新的代理规则
     * 默认添加一个指向 localhost:8080 的规则
     */
    addProxyRule() {
      state.proxyRules.push({
        path: '/api/',                      // 默认路径前缀
        target: 'http://localhost:8080/',   // 默认目标地址
        rewrite: '',                        // 默认不重写
        enabled: true,                      // 默认启用
      })
    },

    /**
     * 删除指定索引的代理规则
     * @param index - 要删除的规则索引
     */
    removeProxyRule(index: number) {
      state.proxyRules.splice(index, 1)
    },

    /**
     * 更新指定索引的代理规则
     * @param index - 要更新的规则索引
     * @param rule - 要更新的字段（部分更新）
     */
    updateProxyRule(index: number, rule: Partial<ProxyRule>) {
      Object.assign(state.proxyRules[index], rule)
    },

    // ========== 步骤 4: 构建状态 ==========

    /**
     * 设置构建状态
     * @param building - 是否正在构建
     */
    setBuilding(building: boolean) {
      state.building = building
      // 开始构建时，重置相关状态
      if (building) {
        state.buildError = ''
        state.outputPath = ''
        state.buildProgress = 0
        state.buildStep = ''
      }
    },

    /**
     * 更新构建进度
     * @param step - 当前步骤描述
     * @param progress - 进度百分比（0-100）
     */
    setBuildProgress(step: string, progress: number) {
      state.buildStep = step
      state.buildProgress = progress
    },

    /**
     * 标记构建完成
     * @param outputPath - 输出文件路径
     */
    setBuildComplete(outputPath: string) {
      state.building = false
      state.outputPath = outputPath
      state.buildProgress = 100
      state.buildStep = '构建完成!'
    },

    /**
     * 设置构建错误
     * @param error - 错误信息
     */
    setBuildError(error: string) {
      state.building = false
      state.buildError = error
    },

    /**
     * 设置输出路径
     * @param path - 输出文件路径
     */
    setOutputPath(path: string) {
      state.outputPath = path
    },

    // ========== 导航 ==========

    /**
     * 设置当前步骤
     * @param step - 步骤索引（0-3）
     */
    setCurrentStep(step: number) {
      state.currentStep = step
    },

    // ========== 窗口设置 ==========

    /**
     * 设置窗口宽度
     * @param width - 宽度（像素）
     */
    setWindowWidth(width: number) {
      state.windowWidth = width
    },

    /**
     * 设置窗口高度
     * @param height - 高度（像素）
     */
    setWindowHeight(height: number) {
      state.windowHeight = height
    },

    /**
     * 设置全屏状态
     * @param fullscreen - 是否全屏
     */
    setWindowFullscreen(fullscreen: boolean) {
      state.windowFullscreen = fullscreen
    },

    /**
     * 设置最大化状态
     * @param maximized - 是否最大化
     */
    setWindowMaximized(maximized: boolean) {
      state.windowMaximized = maximized
    },
  }
}
