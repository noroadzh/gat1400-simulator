<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { controlWs } from '@/api/ws'

interface Subscription {
  subscribeID: string
  title?: string
  subscribeType?: string
  resource?: { uri?: string; path?: string }
  target?: { device?: string; group?: string }
  status?: string
}

const subscribes = ref<Subscription[]>([])
const dispositions = ref<unknown[]>([])

// 后端尚未实现 /api/control/subscriptions（见 openspec 待交付 change）。
// 这里不发起请求，避免每条 WS 事件都触发一次 404；待接口落地后再接 controlApi。
async function load() {
  subscribes.value = []
  dispositions.value = []
}

const subsJson = ref<string>('loading...')
const dispJson = ref<string>('loading...')

async function refreshJson() {
  await load()
  subsJson.value = JSON.stringify(subscribes.value, null, 2)
  dispJson.value = JSON.stringify(dispositions.value, null, 2)
}

// 订阅 controlWs.on 返回的解绑函数，避免组件卸载后仍被 WS 事件触发
let offWs: (() => void) | null = null
onMounted(() => {
  offWs = controlWs.on(refreshJson)
  void refreshJson()
})
onUnmounted(() => {
  offWs?.()
  offWs = null
})
</script>

<template>
  <div class="subs-view">
    <div class="glass-card">
      <div class="panel-header">
        <span class="panel-title">Subscribes</span>
        <span class="panel-count">{{ subscribes.length }}</span>
      </div>
      <pre class="json-display">{{ subsJson }}</pre>
    </div>

    <div class="glass-card">
      <div class="panel-header">
        <span class="panel-title">Dispositions</span>
        <span class="panel-count">{{ dispositions.length }}</span>
      </div>
      <pre class="json-display">{{ dispJson }}</pre>
    </div>

    <div class="coming-soon glass-card">
      <p class="coming-soon-icon">
        <svg width="40" height="40" viewBox="0 0 24 24" fill="none">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 14.5v-9l6 4.5-6 4.5z" fill="currentColor"/>
        </svg>
      </p>
      <p class="coming-soon-text">Subscriptions / Dispositions 路由</p>
      <p class="coming-soon-sub">后端尚未实现 <code>/api/control/subscriptions</code> 接口</p>
      <p class="coming-soon-sub">计划在 #15 scenario-engine 中交付</p>
    </div>
  </div>
</template>

<style scoped>
.subs-view {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

@media (max-width: 1024px) {
  .subs-view {
    grid-template-columns: 1fr;
  }
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid var(--card-border);
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
}

.panel-count {
  font-size: 11px;
  color: var(--text-muted);
  background: rgba(91, 140, 255, 0.15);
  padding: 2px 8px;
  border-radius: 10px;
}

.json-display {
  margin: 0;
  padding: 14px 16px;
  font-family: 'Courier New', monospace;
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-muted);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 480px;
  overflow-y: auto;
}

.coming-soon {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 28px 16px;
  text-align: center;
}

.coming-soon-icon {
  margin: 0 0 4px;
  color: var(--primary);
  opacity: 0.55;
}

.coming-soon-text {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.coming-soon-sub {
  margin: 0;
  font-size: 11px;
  color: var(--text-muted);
}

.coming-soon-sub code {
  font-family: 'Courier New', monospace;
  color: var(--primary);
}
</style>