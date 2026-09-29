<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import type { Capture } from '@/api/control'
import { controlApi } from '@/api/control'
import { controlWs } from '@/api/ws'

const captures = ref<Capture[]>([])
const loading = ref(false)
let timer: ReturnType<typeof setInterval> | null = null
let offWs: (() => void) | null = null

async function load() {
  try {
    captures.value = await controlApi.getCaptures(20)
  } catch (e) {
    console.warn('load captures failed', e)
  }
}

async function refresh() {
  loading.value = true
  await load()
  loading.value = false
}

onMounted(async () => {
  await refresh()
  timer = setInterval(refresh, 3000)
  offWs = controlWs.on(() => refresh())
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  offWs?.()
  offWs = null
})

function formatDuration(ms: number): string {
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(2)}s`
}

function statusColor(code: number): string {
  if (code < 300) return 'var(--success)'
  if (code < 400) return 'var(--warn)'
  return 'var(--danger)'
}
</script>

<template>
  <div class="glass-card live-capture">
    <div class="panel-header">
      <span class="panel-title">Live Captures</span>
      <span class="live-dot"></span>
    </div>
    <el-table
      :data="captures"
      :loading="loading"
      size="small"
      style="width: 100%"
      :row-class-name="() => 'capture-row'"
    >
      <el-table-column label="Time" width="80">
        <template #default="{ row }">
          <span class="mono-text">{{ row.timestamp.slice(11, 19) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="Dir" width="60">
        <template #default="{ row }">
          <el-tag
            :type="row.direction === 'inbound' ? 'primary' : 'warning'"
            size="small"
          >
            {{ row.direction === 'inbound' ? 'IN' : 'OUT' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Method" width="70">
        <template #default="{ row }">
          <span class="mono-text">{{ row.method }}</span>
        </template>
      </el-table-column>
      <el-table-column label="Path">
        <template #default="{ row }">
          <span class="mono-text path-text">{{ row.path }}</span>
        </template>
      </el-table-column>
      <el-table-column label="Status" width="70">
        <template #default="{ row }">
          <span :style="{ color: statusColor(row.status) }">
            {{ row.status }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="Duration" width="80">
        <template #default="{ row }">
          <span class="mono-text">{{ formatDuration(row.duration) }}</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.live-capture {
  padding: 0;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px 10px;
  border-bottom: 1px solid var(--card-border);
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--danger);
  animation: blink 1.5s infinite;
  margin-left: auto;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.mono-text {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.path-text {
  color: var(--text-muted);
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
}

:deep(.capture-row:hover) {
  background: rgba(91, 140, 255, 0.06) !important;
}
</style>