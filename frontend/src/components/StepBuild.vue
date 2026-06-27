<script lang="ts" setup>
/**
 * StepBuild 组件 - 构建生成步骤
 * 功能：
 * 1. 显示构建配置摘要
 * 2. 执行应用构建流程
 * 3. 显示构建进度
 * 4. 显示构建结果（成功或失败）
 */
import {onMounted, onUnmounted} from 'vue'
import {NCard, NButton, NSpace, NProgress, NAlert, NResult, NTimeline, NTimelineItem, NSpin} from 'naive-ui'
import {useStore} from '../store'
import {BuildApp, OpenOutputFolder} from '../../wailsjs/go/main/App'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import {main} from '../../wailsjs/go/models'

// 获取全局状态管理
const store = useStore()

// 事件监听器引用，用于在组件卸载时取消监听
let unlistenProgress: (() => void) | null = null
let unlistenComplete: (() => void) | null = null

/**
 * 组件挂载时注册事件监听器
 * 监听后端发送的构建进度和完成事件
 */
onMounted(() => {
  // 监听构建进度事件
  unlistenProgress = EventsOn('build:progress', (data: any) => {
    store.setBuildProgress(data.step, data.progress)
  })
  // 监听构建完成事件
  unlistenComplete = EventsOn('build:complete', (data: any) => {
    store.setBuildComplete(data.outputPath)
  })
})

/**
 * 组件卸载时取消事件监听器，防止内存泄漏
 */
onUnmounted(() => {
  unlistenProgress?.()
  unlistenComplete?.()
})

/**
 * 开始构建应用
 * 验证配置后调用后端 BuildApp 方法
 */
async function startBuild() {
  // 验证是否已导入构建产物
  if (!store.state.distPath) {
    store.setBuildError('请先导入前端构建产物')
    return
  }
  // 验证是否已设置应用名称
  if (!store.state.appName) {
    store.setBuildError('请先设置应用名称')
    return
  }

  // 设置构建状态为 true，重置相关状态
  store.setBuilding(true)

  try {
    // 调用后端构建方法，传入构建配置
    await BuildApp(main.BuildConfig.createFrom({
      appName: store.state.appName,           // 应用名称
      iconPath: store.state.iconPath,         // 图标路径
      distPath: store.state.distPath,         // 前端构建产物路径
      tempPath: store.state.tempPath,         // 临时目录路径
      proxyRules: [...store.state.proxyRules], // 代理规则（展开数组以传递副本）
      windowWidth: store.state.windowWidth,   // 窗口宽度
      windowHeight: store.state.windowHeight, // 窗口高度
      windowFullscreen: store.state.windowFullscreen, // 是否全屏
      windowMaximized: store.state.windowMaximized,   // 是否最大化
    }))
  } catch (e: any) {
    // 捕获构建错误并显示
    store.setBuildError(e?.message || String(e))
  }
}

/**
 * 打开输出目录
 * 使用系统文件管理器打开生成的应用所在目录
 */
async function openOutput() {
  if (store.state.outputPath) {
    try {
      await OpenOutputFolder(store.state.outputPath)
    } catch (e: any) {
      // 忽略打开目录的错误
    }
  }
}

/**
 * 重置构建状态
 * 用于重新构建时清除之前的状态
 */
function resetBuild() {
  store.setBuilding(false)
  store.setBuildProgress('', 0)
  store.setBuildError('')
  store.setOutputPath('')
}

/**
 * 返回上一步（代理配置步骤）
 */
function prevStep() {
  store.setCurrentStep(2)
}
</script>

<template>
  <div class="step-container">
    <div class="step-title">
      <h3>构建生成</h3>
      <p>确认配置无误后，点击构建按钮生成桌面应用</p>
    </div>

    <!-- Config summary -->
    <NCard title="构建配置" size="small" style="margin-bottom: 16px">
      <div class="config-summary">
        <div class="summary-item">
          <span class="label">应用名称:</span>
          <span class="value">{{ store.state.appName || '未设置' }}</span>
        </div>
        <div class="summary-item">
          <span class="label">构建产物:</span>
          <span class="value">{{ store.state.distPath || '未导入' }}</span>
        </div>
        <div class="summary-item">
          <span class="label">临时目录:</span>
          <span class="value">{{ store.state.tempPath || '未设置，使用当前导入目录' }}</span>
        </div>
        <div class="summary-item">
          <span class="label">应用图标:</span>
          <span class="value">{{ store.state.iconPath ? '已选择' : '使用默认' }}</span>
        </div>
        <div class="summary-item">
          <span class="label">代理规则:</span>
          <span class="value">{{ store.state.proxyRules.length }} 条</span>
        </div>
        <div class="summary-item">
          <span class="label">窗口大小:</span>
          <span class="value">{{ store.state.windowFullscreen ? '全屏' : store.state.windowMaximized ? '最大化' : store.state.windowWidth + ' x ' + store.state.windowHeight }}</span>
        </div>
      </div>
    </NCard>

    <!-- Build error -->
    <NAlert v-if="store.state.buildError" type="error" closable style="margin-bottom: 16px">
      {{ store.state.buildError }}
    </NAlert>

    <!-- Building in progress -->
    <NCard v-if="store.state.building">
      <div class="build-progress">
        <NSpin size="large" style="margin-bottom: 16px" />
        <NProgress
          type="line"
          :percentage="store.state.buildProgress"
          :processing="store.state.buildProgress < 100"
          style="margin-bottom: 12px"
        />
        <p class="progress-text">{{ store.state.buildStep }}</p>
      </div>
    </NCard>

    <!-- Build complete -->
    <NCard v-else-if="store.state.outputPath">
      <NResult
        status="success"
        title="构建成功!"
        :description="'输出路径: ' + store.state.outputPath"
      >
        <template #footer>
          <NSpace justify="center">
            <NButton type="primary" @click="openOutput">
              打开输出目录
            </NButton>
            <NButton @click="resetBuild">
              重新构建
            </NButton>
          </NSpace>
        </template>
      </NResult>
    </NCard>

    <!-- Ready to build -->
    <NCard v-else>
      <div class="build-ready">
        <div class="ready-icon">🚀</div>
        <p>一切准备就绪，点击下方按钮开始构建</p>
        <NSpace justify="center" style="margin-top: 16px">
          <NButton @click="prevStep">上一步</NButton>
          <NButton
            type="primary"
            size="large"
            @click="startBuild"
            :disabled="!store.state.distPath || !store.state.appName"
          >
            开始构建
          </NButton>
        </NSpace>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.step-container {
  padding: 32px;
  max-width: 600px;
  margin: 0 auto;
}

.step-title {
  margin-bottom: 24px;
}

.step-title h3 {
  margin: 0 0 8px;
  font-size: 20px;
  color: #333;
}

.step-title p {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.config-summary {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.summary-item {
  display: flex;
  gap: 8px;
  font-size: 14px;
}

.summary-item .label {
  color: #999;
  min-width: 80px;
}

.summary-item .value {
  color: #333;
  word-break: break-all;
}

.build-progress {
  text-align: center;
  padding: 32px 0;
}

.progress-text {
  color: #666;
  font-size: 14px;
  margin: 0;
}

.build-ready {
  text-align: center;
  padding: 48px 0;
}

.ready-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

.build-ready p {
  color: #666;
  font-size: 14px;
  margin: 0;
}
</style>
