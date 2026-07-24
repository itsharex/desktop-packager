import {reactive, readonly} from 'vue'

export interface ProxyRule {
  path: string
  target: string
  rewrite: string
  enabled: boolean
}

export interface AppState {
  distPath: string
  distFileCount: number
  distTotalSize: number

  appName: string
  iconPath: string
  iconPreview: string
  tempPath: string
  version: string
  description: string
  company: string
  confirmClose: boolean

  proxyRules: ProxyRule[]

  building: boolean
  buildProgress: number
  buildStep: string
  outputPath: string
  buildError: string

  currentStep: number

  windowWidth: number
  windowHeight: number
  windowFullscreen: boolean
  windowMaximized: boolean
}

const state = reactive<AppState>({
  distPath: '',
  distFileCount: 0,
  distTotalSize: 0,

  appName: '',
  iconPath: '',
  iconPreview: '',
  tempPath: '',
  version: '1.0.0',
  description: '',
  company: '',
  confirmClose: true,

  proxyRules: [],

  building: false,
  buildProgress: 0,
  buildStep: '',
  outputPath: '',
  buildError: '',

  currentStep: 0,

  windowWidth: 1024,
  windowHeight: 768,
  windowFullscreen: false,
  windowMaximized: false,
})

export function useStore() {
  return {
    state: readonly(state),

    setDist(path: string, fileCount: number, totalSize: number) {
      state.distPath = path
      state.distFileCount = fileCount
      state.distTotalSize = totalSize
    },

    clearDist() {
      state.distPath = ''
      state.distFileCount = 0
      state.distTotalSize = 0
    },

    setAppName(name: string) {
      state.appName = name
    },

    setIcon(path: string, preview: string) {
      state.iconPath = path
      state.iconPreview = preview
    },

    clearIcon() {
      state.iconPath = ''
      state.iconPreview = ''
    },

    setTempPath(path: string) {
      state.tempPath = path
    },

    setVersion(version: string) {
      state.version = version
    },

    setDescription(description: string) {
      state.description = description
    },

    setCompany(company: string) {
      state.company = company
    },

    setConfirmClose(confirmClose: boolean) {
      state.confirmClose = confirmClose
    },


    addProxyRule() {
      state.proxyRules.push({
        path: '/api/',
        target: 'http://localhost:8080/',
        rewrite: '',
        enabled: true,
      })
    },

    removeProxyRule(index: number) {
      state.proxyRules.splice(index, 1)
    },

    updateProxyRule(index: number, rule: Partial<ProxyRule>) {
      Object.assign(state.proxyRules[index], rule)
    },

    setBuilding(building: boolean) {
      state.building = building
      if (building) {
        state.buildError = ''
        state.outputPath = ''
        state.buildProgress = 0
        state.buildStep = ''
      }
    },

    setBuildProgress(step: string, progress: number) {
      state.buildStep = step
      state.buildProgress = progress
    },

    setBuildComplete(outputPath: string) {
      state.building = false
      state.outputPath = outputPath
      state.buildProgress = 100
      state.buildStep = '构建完成!'
    },

    setBuildError(error: string) {
      state.building = false
      state.buildError = error
    },

    setOutputPath(path: string) {
      state.outputPath = path
    },

    setCurrentStep(step: number) {
      state.currentStep = step
    },

    setWindowWidth(width: number) {
      state.windowWidth = width
    },

    setWindowHeight(height: number) {
      state.windowHeight = height
    },

    setWindowFullscreen(fullscreen: boolean) {
      state.windowFullscreen = fullscreen
      if (fullscreen) {
        state.windowMaximized = false
      }
    },

    setWindowMaximized(maximized: boolean) {
      state.windowMaximized = maximized
      if (maximized) {
        state.windowFullscreen = false
      }
    },
  }
}