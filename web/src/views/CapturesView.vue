<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { Capture } from '@/api/control'
import { controlApi } from '@/api/control'

const all = ref<Capture[]>([])
const filter = ref('')

async function load() {
  try {
    all.value = await controlApi.getCaptures(200)
  } catch (e) {
    console.warn('load captures failed', e)
  }
}

const filtered = computed(() => {
  const q = filter.value.trim().toLowerCase()
  if (!q) return all.value
  return all.value.filter(
    (c) =>
      c.path.toLowerCase().includes(q) ||
      c.method.toLowerCase().includes(q) ||
      String(c.status).includes(q)
  )
})

function exportJsonl() {
  const text = all.value.map((c) => JSON.stringify(c)).join('\n')
  download(text, 'captures.jsonl', 'application/jsonl')
}

function exportHar() {
  const har = {
    log: {
      version: '1.2',
      creator: { name: 'gat1400-simulator', version: '0.1.0' },
      entries: all.value.map((c) => ({
        startedDateTime: c.timestamp,
        time: c.duration,
        response: { status: c.status },
        request: { method: c.method, url: c.path },
        serverIPAddress: '',
        _serverPort: 14080
      }))
    }
  }
  download(JSON.stringify(har, null, 2), 'captures.har', 'application/json')
}

function download(text: string, name: string, mime: string) {
  const blob = new Blob([text], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(load)
</script>

<template>
  <div class="captures-view">
    <div class="glass-card">
      <div class="panel-header">
        <span class="panel-title">Captures ({{ filtered.length }})</span>
        <div class="actions">
          <el-input
            v-model="filter"
            placeholder="filter path/method/status…"
            clearable
            style="width: 220px"
          />
          <el-button size="small" @click="exportJsonl">JSONL</el-button>
          <el-button size="small" type="primary" @click="exportHar">HAR</el-button>
        </div>
      </div>

      <el-table :data="filtered" size="small">
        <el-table-column label="Time" width="160">
          <template #default="{ row }">
            <span class="mono-text">{{ row.timestamp }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Dir" width="70">
          <template #default="{ row }">
            <el-tag :type="row.direction === 'inbound' ? 'primary' : 'warning'" size="small">
              {{ row.direction === 'inbound' ? 'IN' : 'OUT' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Method" prop="method" width="90" />
        <el-table-column label="Path" prop="path" min-width="240" />
        <el-table-column label="Status" prop="status" width="90" />
        <el-table-column label="Duration" width="100">
          <template #default="{ row }">
            <span class="mono-text">{{ row.duration }}ms</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<style scoped>
.captures-view {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid var(--card-border);
  gap: 12px;
  flex-wrap: wrap;
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
}

.actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.mono-text {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}
</style>