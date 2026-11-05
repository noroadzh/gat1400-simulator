<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { controlApi } from '@/api/control'
import { controlWs } from '@/api/ws'

const subscribes = ref<unknown[]>([])
const dispositions = ref<unknown[]>([])
const error = ref('')
const loading = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [subs, disps] = await Promise.all([
      controlApi.getSubscriptions(),
      controlApi.getDispositions(),
    ])
    subscribes.value = Array.isArray(subs.items) ? subs.items : []
    dispositions.value = Array.isArray(disps.items) ? disps.items : []
  } catch (e) {
    error.value = (e as Error).message
    subscribes.value = []
    dispositions.value = []
  } finally {
    loading.value = false
  }
}

const subsJson = ref<string>('loading...')
const dispJson = ref<string>('loading...')

async function refreshJson() {
  await load()
  subsJson.value = JSON.stringify(subscribes.value, null, 2)
  dispJson.value = JSON.stringify(dispositions.value, null, 2)
}

let offWs: (() => void) | null = null
onMounted(() => {
  offWs = controlWs.on(() => {
    void refreshJson()
  })
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
        <div class="panel-actions">
          <span class="panel-count">{{ subscribes.length }}</span>
          <el-button size="small" :loading="loading" @click="refreshJson">Refresh</el-button>
        </div>
      </div>
      <p v-if="error" class="err">{{ error }}</p>
      <pre class="json-display">{{ subsJson }}</pre>
    </div>

    <div class="glass-card">
      <div class="panel-header">
        <span class="panel-title">Dispositions</span>
        <span class="panel-count">{{ dispositions.length }}</span>
      </div>
      <pre class="json-display">{{ dispJson }}</pre>
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

.panel-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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

.err {
  margin: 8px 16px 0;
  color: #f56c6c;
  font-size: 12px;
}
</style>
