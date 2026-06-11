<script lang="ts" setup>
import {ref} from 'vue'
import {NCard, NButton, NSpace, NInput, NFormItem, NForm, NImage, NAlert, NInputNumber, NSwitch, NDivider} from 'naive-ui'
import {useStore} from '../store'
import {OpenIconFile, ValidateIcon} from '../../wailsjs/go/main/App'

const store = useStore()
const formRef = ref()
const error = ref('')

const rules = {
  appName: [
    {required: true, message: '请输入应用名称', trigger: 'blur'},
    {min: 1, max: 50, message: '名称长度 1-50 个字符', trigger: 'blur'},
  ],
}

async function selectIcon() {
  error.value = ''
  try {
    const path = await OpenIconFile()
    if (!path) return
    const preview = await ValidateIcon(path)
    store.setIcon(path, preview)
  } catch (e: any) {
    error.value = e?.message || String(e)
  }
}

function clearIcon() {
  store.clearIcon()
}

function prevStep() {
  store.setCurrentStep(0)
}

async function nextStep() {
  try {
    await formRef.value?.validate()
    store.setCurrentStep(2)
  } catch {
    // validation failed
  }
}
</script>

<template>
  <div class="step-container">
    <div class="step-title">
      <h3>应用配置</h3>
      <p>设置桌面应用的名称和图标</p>
    </div>

    <NAlert v-if="error" type="error" closable @close="error = ''" style="margin-bottom: 16px">
      {{ error }}
    </NAlert>

    <NCard>
      <NForm
        ref="formRef"
        :model="{appName: store.state.appName}"
        :rules="rules"
        label-placement="left"
        label-width="80"
      >
        <NFormItem label="应用名称" path="appName">
          <NInput
            :value="store.state.appName"
            @update:value="store.setAppName($event)"
            placeholder="请输入桌面应用名称"
            maxlength="50"
            show-count
          />
        </NFormItem>

        <NFormItem label="应用图标">
          <div class="icon-section">
            <div v-if="store.state.iconPreview" class="icon-preview">
              <NImage
                :src="store.state.iconPreview"
                width="64"
                height="64"
                object-fit="contain"
                style="border: 1px solid #eee; border-radius: 4px;"
              />
              <div class="icon-info">
                <span class="icon-path">{{ store.state.iconPath }}</span>
                <NButton text type="error" @click="clearIcon" size="small">
                  移除图标
                </NButton>
              </div>
            </div>
            <div v-else class="icon-placeholder">
              <div class="icon-default">🖼️</div>
              <NButton @click="selectIcon" type="primary" ghost>
                选择图标文件
              </NButton>
              <p class="icon-hint">支持 .ico 和 .png 格式，建议 256x256 像素</p>
            </div>
          </div>
        </NFormItem>

        <NDivider />

        <NFormItem label="窗口宽度">
          <NInputNumber
            :value="store.state.windowWidth"
            @update:value="(v: number | null) => store.setWindowWidth(v ?? 1024)"
            :min="400"
            :max="3840"
            :step="50"
            style="width: 200px"
          />
        </NFormItem>

        <NFormItem label="窗口高度">
          <NInputNumber
            :value="store.state.windowHeight"
            @update:value="(v: number | null) => store.setWindowHeight(v ?? 768)"
            :min="300"
            :max="2160"
            :step="50"
            style="width: 200px"
          />
        </NFormItem>

        <NFormItem label="窗口选项">
          <div class="switch-row">
            <div class="switch-item">
              <span class="switch-label">全屏</span>
              <NSwitch
                :value="store.state.windowFullscreen"
                @update:value="store.setWindowFullscreen($event)"
                size="small"
              />
            </div>
            <div class="switch-item">
              <span class="switch-label">最大化</span>
              <NSwitch
                :value="store.state.windowMaximized"
                @update:value="store.setWindowMaximized($event)"
                size="small"
              />
            </div>
          </div>
        </NFormItem>
      </NForm>

      <div class="form-actions">
        <NSpace justify="space-between">
          <NButton @click="prevStep">上一步</NButton>
          <NButton type="primary" @click="nextStep">下一步</NButton>
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

.icon-section {
  width: 100%;
}

.icon-preview {
  display: flex;
  align-items: center;
  gap: 16px;
}

.icon-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.icon-path {
  font-size: 12px;
  color: #999;
  word-break: break-all;
  max-width: 300px;
}

.icon-placeholder {
  text-align: center;
  padding: 24px;
  border: 2px dashed #ddd;
  border-radius: 8px;
  width: 100%;
}

.icon-default {
  font-size: 48px;
  margin-bottom: 12px;
}

.icon-hint {
  font-size: 12px;
  color: #999;
  margin-top: 8px;
}

.form-actions {
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #eee;
}

.switch-row {
  display: flex;
  gap: 24px;
  align-items: center;
}

.switch-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.switch-label {
  font-size: 13px;
  color: #666;
  white-space: nowrap;
}

</style>
