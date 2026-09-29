<script setup lang="ts">
import { ref, onMounted } from 'vue'
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

async function load() {
  try {
    const res = await fetch('/api/control/subscriptions')
    if (res.ok) {
        const data = await res.json()
        if (Array.isArray(data)) subscribes.value = data
        else if (data && Array.isArray((data as any).subscribes))
          subscribes.value = (data as any).subscribes
        if (data && Array.isArray((data as any).dispositions))
          dispositions.value = (data as any).dispositions
      }
  } catch (e) {
    console.warn('load subs failed', e)
  }
}

const subsJson = ref<string>('loading...')
const dispJson = ref<string>('loading...')

async function refreshJson() {
  await load()
  subsJson.value = JSON.stringify(subscribes.value, null, 2)
  dispJson.value = JSON.stringify(dispositions.value, null, 2)
}

onMounted(refreshJson)
controlWs.on(refreshJson)
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
</style>