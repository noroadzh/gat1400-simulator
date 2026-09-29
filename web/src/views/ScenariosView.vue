<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Scenario } from '@/api/control'
import { controlApi } from '@/api/control'

const scenarios = ref<Scenario[]>([])
const busy = ref<string | null>(null)

async function load() {
  try {
    scenarios.value = await controlApi.getScenarios()
  } catch (e) {
    console.warn('load scenarios failed', e)
  }
}

async function start(id: string) {
  busy.value = id
  try {
    await controlApi.startScenario(id)
    await load()
  } catch (e) {
    console.error('start failed', e)
  } finally {
    busy.value = null
  }
}

async function stop(id: string) {
  busy.value = id
  try {
    await controlApi.stopScenario(id)
    await load()
  } catch (e) {
    console.error('stop failed', e)
  } finally {
    busy.value = null
  }
}

function statusTag(status: Scenario['status']): { type: 'success' | 'warning' | 'info'; label: string } {
  if (status === 'running') return { type: 'success', label: 'RUNNING' }
  if (status === 'stopped') return { type: 'warning', label: 'STOPPED' }
  return { type: 'info', label: 'IDLE' }
}

onMounted(load)
</script>

<template>
  <div class="scenarios-view">
    <div class="glass-card">
      <div class="panel-header">
        <span class="panel-title">Scenarios ({{ scenarios.length }})</span>
      </div>

      <el-table :data="scenarios" size="small">
        <el-table-column label="ID" prop="id" width="120" />
        <el-table-column label="Name" prop="name" min-width="180" />
        <el-table-column label="Tags" min-width="160">
          <template #default="{ row }">
            <el-tag
              v-for="t in row.tags ?? []"
              :key="t"
              size="small"
              type="info"
              class="mr-4"
            >
              {{ t }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Nodes" min-width="120">
          <template #default="{ row }">
            <span class="mono-text muted">{{ (row.nodes ?? []).join(', ') }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Status" width="120">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status).type" size="small">
              {{ statusTag(row.status).label }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="180" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              type="success"
              :loading="busy === row.id"
              @click="start(row.id)"
            >
              Start
            </el-button>
            <el-button
              size="small"
              type="warning"
              :loading="busy === row.id"
              @click="stop(row.id)"
            >
              Stop
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<style scoped>
.scenarios-view {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.panel-header {
  padding: 14px 16px;
  border-bottom: 1px solid var(--card-border);
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
}

.mono-text {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.muted {
  color: var(--text-muted);
}

.mr-4 {
  margin-right: 4px;
  margin-bottom: 2px;
}
</style>