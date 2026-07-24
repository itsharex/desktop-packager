<script lang="ts" setup>
import {ref} from 'vue'
import {
  NAlert,
  NButton,
  NConfigProvider,
  NDialogProvider,
  NInput,
  NMessageProvider,
  NModal,
  NSpace,
  NTooltip,
  zhCN,
  dateZhCN,
} from 'naive-ui'
import {useStore} from './store'
import {canEnterStep} from './validation'
import StepImport from './components/StepImport.vue'
import StepSettings from './components/StepSettings.vue'
import StepProxy from './components/StepProxy.vue'
import StepBuild from './components/StepBuild.vue'
import {OpenTempFolder} from '../wailsjs/go/main/App'

const store = useStore()
const showGlobalSettings = ref(false)
const globalSettingsError = ref('')

const steps = [
  {label: '导入构建产物', desc: '选择 dist 文件夹或 ZIP'},
  {label: '应用配置', desc: '设置名称和图标'},
  {label: '反向代理', desc: '配置 nginx 风格代理'},
  {label: '构建生成', desc: '生成桌面应用'},
]

function handleStepClick(index: number) {
  // Allow going back freely; forward only when prerequisites are met.
  if (index <= store.state.currentStep) {
    store.setCurrentStep(index)
    return
  }
  if (canEnterStep(index, {
    distPath: store.state.distPath,
    appName: store.state.appName,
  })) {
    store.setCurrentStep(index)
  }
}

async function selectTempFolder() {
  globalSettingsError.value = ''
  try {
    const path = await OpenTempFolder()
    if (!path) return
    store.setTempPath(path)
  } catch (e: any) {
    globalSettingsError.value = e?.message || String(e)
  }
}
</script>

<template>
  <NConfigProvider :locale="zhCN" :date-locale="dateZhCN">
    <NMessageProvider>
      <NDialogProvider>
        <div class="app-layout">
          <div class="sidebar">
            <div class="sidebar-header">
              <h2>应用生成器</h2>
              <NTooltip trigger="hover">
                <template #trigger>
                  <NButton
                    class="settings-icon-button"
                    quaternary
                    circle
                    size="small"
                    aria-label="全局配置"
                    @click="showGlobalSettings = true"
                  >
                    <svg viewBox="0 0 24 24" aria-hidden="true">
                      <path
                        d="M12 15.5A3.5 3.5 0 1 0 12 8a3.5 3.5 0 0 0 0 7.5Zm7.2-3.5c0-.4 0-.8-.1-1.2l2-1.5-2-3.4-2.4 1a8 8 0 0 0-2-1.2L14.4 3h-4.8l-.4 2.7a8 8 0 0 0-2 1.2l-2.4-1-2 3.4 2 1.5a8.2 8.2 0 0 0 0 2.4l-2 1.5 2 3.4 2.4-1a8 8 0 0 0 2 1.2l.4 2.7h4.8l.4-2.7a8 8 0 0 0 2-1.2l2.4 1 2-3.4-2-1.5c.1-.4.1-.8.1-1.2Z"
                      />
                    </svg>
                  </NButton>
                </template>
                全局配置
              </NTooltip>
            </div>
            <div class="step-list">
              <div
                v-for="(step, index) in steps"
                :key="index"
                class="step-item"
                :class="{
                  active: store.state.currentStep === index,
                  completed: index < store.state.currentStep,
                  disabled: index > store.state.currentStep && !canEnterStep(index, { distPath: store.state.distPath, appName: store.state.appName })
                }"
                @click="handleStepClick(index)"
              >
                <div class="step-number">
                  <span v-if="index < store.state.currentStep">✓</span>
                  <span v-else>{{ index + 1 }}</span>
                </div>
                <div class="step-info">
                  <div class="step-label">{{ step.label }}</div>
                  <div class="step-desc">{{ step.desc }}</div>
                </div>
              </div>
            </div>
          </div>

          <div class="main-content">
            <StepImport v-if="store.state.currentStep === 0" />
            <StepSettings v-else-if="store.state.currentStep === 1" />
            <StepProxy v-else-if="store.state.currentStep === 2" />
            <StepBuild v-else-if="store.state.currentStep === 3" />
          </div>
        </div>

        <NModal v-model:show="showGlobalSettings" preset="card" title="全局配置" style="width: 560px">
          <NAlert
            v-if="globalSettingsError"
            type="error"
            closable
            style="margin-bottom: 12px"
            @close="globalSettingsError = ''"
          >
            {{ globalSettingsError }}
          </NAlert>
          <div class="settings-field">
            <div class="settings-label">临时目录</div>
            <NInput
              :value="store.state.tempPath"
              @update:value="store.setTempPath($event)"
              placeholder="留空则使用当前前端产物或 ZIP 所在目录"
            />
            <div class="settings-hint">
              ZIP 解压、构建工作目录都会放在这里；如果留空，会使用当前导入文件所在目录。退出应用时会清理会话内创建的临时目录。
            </div>
          </div>
          <template #footer>
            <NSpace justify="space-between">
              <NButton @click="store.setTempPath('')">清空</NButton>
              <NSpace>
                <NButton @click="selectTempFolder">选择目录</NButton>
                <NButton type="primary" @click="showGlobalSettings = false">完成</NButton>
              </NSpace>
            </NSpace>
          </template>
        </NModal>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  width: 240px;
  min-width: 240px;
  background: #1e1e2e;
  color: #cdd6f4;
  display: flex;
  flex-direction: column;
  user-select: none;
}

.sidebar-header {
  padding: 18px 14px 14px 16px;
  border-bottom: 1px solid #313244;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.sidebar-header h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #cdd6f4;
}

.settings-icon-button {
  color: #cdd6f4;
  flex-shrink: 0;
}

.settings-icon-button svg {
  width: 17px;
  height: 17px;
  fill: currentColor;
}

.step-list {
  padding: 12px 8px;
  flex: 1;
}

.step-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: 4px;
}

.step-item:hover {
  background: #313244;
}

.step-item.active {
  background: #45475a;
}

.step-item.disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.step-item.completed .step-number {
  background: #a6e3a1;
  color: #1e1e2e;
}

.step-number {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #585b70;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  flex-shrink: 0;
}

.step-item.active .step-number {
  background: #89b4fa;
  color: #1e1e2e;
}

.step-info {
  flex: 1;
  min-width: 0;
}

.step-label {
  font-size: 14px;
  font-weight: 500;
  color: #cdd6f4;
}

.step-desc {
  font-size: 12px;
  color: #6c7086;
  margin-top: 2px;
}

.main-content {
  flex: 1;
  overflow-y: auto;
  background: #f8f9fa;
}

.settings-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.settings-label {
  color: #333;
  font-size: 14px;
  font-weight: 500;
}

.settings-hint {
  color: #888;
  font-size: 12px;
  line-height: 1.5;
}
</style>