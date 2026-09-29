<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { controlApi, type Stats, type SystemHealth } from '@/api/control'
import { controlWs } from '@/api/ws'
import StatCard from '@/components/StatCard.vue'
import LiveCaptureTable from '@/components/LiveCaptureTable.vue'

const stats = ref<Stats | null>(null)
const health = ref<SystemHealth | null>(null)
let timer: ReturnType<typeof setInterval> | null = null

async function load() {
  try {
    const [s, h] = await Promise.all([controlApi.getStats(), controlApi.getSystemHealth()])
    stats.value = s
    health.value = h
  } catch (e) {
    console.warn('dashboard load failed', e)
  }
}

const healthJson = computed(() =>
  health.value ? JSON.stringify(health.value, null, 2) : 'loading...'
)

onMounted(async () => {
  await load()
  timer = setInterval(load, 5000)
  controlWs.on(() => load())
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="dashboard">
    <div class="stats-grid">
      <StatCard label="Nodes" :value="stats?.nodes ?? 0" color="primary" />
      <StatCard label="Online" :value="stats?.online ?? 0" color="success" />
      <StatCard label="Scenarios" :value="stats?.scenarios ?? 0" color="warn" />
      <StatCard label="Running" :value="stats?.running ?? 0" color="danger" />
    </div>

    <div class="dashboard-grid">
      <LiveCaptureTable class="dashboard-feed" />
      <div class="glass-card health-panel">
        <div class="panel-header">
          <span class="panel-title">System Health</span>
        </div>
        <pre class="json-display">{{ healthJson }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 14px;
  min-height: 400px;
}

.dashboard-feed {
  height: 100%;
}

.health-panel {
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
}

.panel-header {
  padding: 14px 16px 10px;
  border-bottom: 1px solid var(--card-border);
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.json-display {
  flex: 1;
  margin: 0;
  padding: 14px 16px;
  font-family: 'Courier New', monospace;
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-muted);
  white-space: pre-wrap;
  word-break: break-word;
  overflow-y: auto;
}

@media (max-width: 1279px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}
</style>