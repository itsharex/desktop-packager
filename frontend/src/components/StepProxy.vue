<script lang="ts" setup>
/**
 * StepProxy 组件 - 反向代理配置步骤
 * 功能：
 * 1. 添加、删除、编辑代理规则
 * 2. 配置路径前缀、目标地址、路径重写
 * 3. 启用/禁用单条规则
 * 4. 显示配置说明和示例
 */
import {NCard, NButton, NSpace, NInput, NSwitch, NAlert, NEmpty, NTooltip} from 'naive-ui'
import {useStore} from '../store'

// 获取全局状态管理
const store = useStore()

/**
 * 添加新的代理规则
 * 默认规则：/api/ -> http://localhost:8080/
 */
function addRule() {
  store.addProxyRule()
}

/**
 * 删除指定索引的代理规则
 * @param index - 要删除的规则索引
 */
function removeRule(index: number) {
  store.removeProxyRule(index)
}

/**
 * 返回上一步（应用设置步骤）
 */
function prevStep() {
  store.setCurrentStep(1)
}

/**
 * 进入下一步（构建生成步骤）
 */
function nextStep() {
  store.setCurrentStep(3)
}
</script>

<template>
  <div class="step-container">
    <div class="step-title">
      <h3>反向代理配置</h3>
      <p>配置类似 nginx 的反向代理规则，解决前端跨域问题</p>
    </div>

    <NCard>
      <!-- Empty state -->
      <div v-if="store.state.proxyRules.length === 0" class="empty-state">
        <NEmpty description="暂无代理规则">
          <template #extra>
            <NButton type="primary" @click="addRule" dashed>
              + 添加代理规则
            </NButton>
          </template>
        </NEmpty>
      </div>

      <!-- Rule list -->
      <div v-else class="rule-list">
        <div class="rule-header">
          <span class="col-path">路径前缀</span>
          <span class="col-target">目标地址</span>
          <span class="col-rewrite">路径重写</span>
          <span class="col-enabled">启用</span>
          <span class="col-action">操作</span>
        </div>

        <div
          v-for="(rule, index) in store.state.proxyRules"
          :key="index"
          class="rule-row"
        >
          <div class="col-path">
            <NInput
              :value="rule.path"
              @update:value="store.updateProxyRule(index, {path: $event})"
              placeholder="/api/"
              size="small"
            />
          </div>
          <div class="col-target">
            <NInput
              :value="rule.target"
              @update:value="store.updateProxyRule(index, {target: $event})"
              placeholder="http://localhost:8080/"
              size="small"
            />
          </div>
          <div class="col-rewrite">
            <NTooltip trigger="hover">
              <template #trigger>
                <NInput
                  :value="rule.rewrite"
                  @update:value="store.updateProxyRule(index, {rewrite: $event})"
                  placeholder="可选"
                  size="small"
                />
              </template>
              路径重写规则，例如将 /api/users 重写为 /users
            </NTooltip>
          </div>
          <div class="col-enabled">
            <NSwitch
              :value="rule.enabled"
              @update:value="store.updateProxyRule(index, {enabled: $event})"
              size="small"
            />
          </div>
          <div class="col-action">
            <NButton text type="error" @click="removeRule(index)" size="small">
              删除
            </NButton>
          </div>
        </div>

        <div class="add-rule">
          <NButton type="primary" dashed @click="addRule" size="small">
            + 添加规则
          </NButton>
        </div>
      </div>

      <!-- Example -->
      <NAlert type="info" style="margin-top: 16px" title="配置说明">
        <p style="margin: 0; font-size: 13px;">
          <strong>路径重写规则（符合 nginx 标准行为）：</strong><br/>
          • 目标地址以 <code>/</code> 结尾 → 剥离路径前缀。例：目标 <code>http://localhost:8080/</code>，<code>/api/users</code> 代理到 <code>/users</code><br/>
          • 目标地址不以 <code>/</code> 结尾 → 保留路径前缀。例：目标 <code>http://localhost:8080</code>，<code>/api/users</code> 代理到 <code>/api/users</code><br/>
          • 重写为 <code>/v2</code> → 替换前缀。例：<code>/api/users</code> 代理到 <code>/v2/users</code><br/><br/>
          <strong>常见场景：</strong>前端请求 <code>/api/users</code>，后端接口是 <code>/users</code>，目标填 <code>http://localhost:8080/</code>（注意末尾斜杠）。
        </p>
      </NAlert>

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
  max-width: 800px;
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

.empty-state {
  padding: 60px 0;
  text-align: center;
}

.rule-list {
  width: 100%;
}

.rule-header {
  display: flex;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 2px solid #eee;
  font-size: 13px;
  font-weight: 600;
  color: #666;
}

.rule-row {
  display: flex;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
  align-items: center;
}

.col-path { flex: 1.2; }
.col-target { flex: 2; }
.col-rewrite { flex: 1.2; }
.col-enabled { width: 50px; text-align: center; }
.col-action { width: 50px; text-align: center; }

.add-rule {
  padding: 12px 0;
}

.form-actions {
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #eee;
}

code {
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 12px;
}
</style>
