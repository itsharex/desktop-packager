<script lang="ts" setup>
import {onMounted, onUnmounted} from 'vue'
import {NCard, NButton, NSpace, NProgress, NAlert, NResult, NTimeline, NTimelineItem, NSpin} from 'naive-ui'
import {useStore} from '../store'
import {BuildApp, OpenOutputFolder} from '../../wailsjs/go/main/App'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import {main} from '../../wailsjs/go/models'

const store = useStore()

let unlistenProgress: (() => void) | null = null
let unlistenComplete: (() => void) | null = null

onMounted(() => {
  unlistenProgress = EventsOn('build:progress', (data: any) => {
    store.setBuildProgress(data.step, data.progress)
  })
  unlistenComplete = EventsOn('build:complete', (data: any) => {
    store.setBuildComplete(data.outputPath)
  })
})

onUnmounted(() => {
  unlistenProgress?.()
  unlistenComplete?.()
})

async function startBuild() {
  if (!store.state.distPath) {
    store.setBuildError('请先导入前端构建产物')
    return
  }
  if (!store.state.appName) {
    store.setBuildError('请先设置应用名称')
    return
  }

  store.setBuilding(true)

  try {
    await BuildApp(main.BuildConfig.createFrom({
      appName: store.state.appName,
      iconPath: store.state.iconPath,
      distPath: store.state.distPath,
      tempPath: store.state.tempPath,
      proxyRules: [...store.state.proxyRules],
      windowWidth: store.state.windowWidth,
      windowHeight: store.state.windowHeight,
      windowFullscreen: store.state.windowFullscreen,
      windowMaximized: store.state.windowMaximized,
    }))
  } catch (e: any) {
    store.setBuildError(e?.message || String(e))
  }
}

async function openOutput() {
  if (store.state.outputPath) {
    try {
      await OpenOutputFolder(store.state.outputPath)
    } catch (e: any) {
      // ignore
    }
  }
}

function resetBuild() {
  store.setBuilding(false)
  store.setBuildProgress('', 0)
  store.setBuildError('')
  store.setOutputPath('')
}

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
