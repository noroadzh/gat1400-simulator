<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { controlApi } from '@/api/control'

const catalog = ref('')
const servers = ref('')
const vias = ref('')
const loading = ref(false)

async function refresh() {
  loading.value = true
  try {
    const [apes, srv, tasks, cap] = await Promise.all([
      controlApi.getCatalog('APEs'),
      controlApi.getVIIDServers(),
      controlApi.getVIASTasks(),
      controlApi.getVIASCapability('AnalysisCapability'),
    ])
    catalog.value = JSON.stringify(apes, null, 2)
    servers.value = JSON.stringify(srv, null, 2)
    vias.value = JSON.stringify({ tasks, capability: cap }, null, 2)
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function createTask() {
  try {
    await controlApi.postVIASTask({
      TaskList: { TaskObject: [{ TaskName: 'ui-demo-task' }] },
    })
    ElMessage.success('VIAS task created')
    await refresh()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(() => {
  void refresh()
})
</script>

<template>
  <div class="extra-view">
    <div class="toolbar glass-card">
      <el-button size="small" :loading="loading" @click="refresh">Refresh</el-button>
      <el-button size="small" type="primary" @click="createTask">Create VIAS Task</el-button>
    </div>
    <div class="grid">
      <div class="glass-card">
        <div class="panel-header"><span class="panel-title">Catalog APEs</span></div>
        <pre class="json-display">{{ catalog }}</pre>
      </div>
      <div class="glass-card">
        <div class="panel-header"><span class="panel-title">VIIDServers</span></div>
        <pre class="json-display">{{ servers }}</pre>
      </div>
      <div class="glass-card wide">
        <div class="panel-header"><span class="panel-title">VIAS Tasks + Capability</span></div>
        <pre class="json-display">{{ vias }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.extra-view { display: flex; flex-direction: column; gap: 14px; }
.toolbar { display: flex; gap: 8px; padding: 12px 14px; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.wide { grid-column: 1 / -1; }
@media (max-width: 1024px) { .grid { grid-template-columns: 1fr; } }
.panel-header { padding: 12px 14px; border-bottom: 1px solid var(--card-border); }
.panel-title { font-weight: 600; }
.json-display { margin: 0; padding: 12px; max-height: 360px; overflow: auto; font-size: 12px; }
</style>
