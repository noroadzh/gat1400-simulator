<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { controlApi } from '@/api/control'

const json = ref<string>('loading…')

async function load() {
  try {
    const cfg = await controlApi.getConfig()
    json.value = JSON.stringify(cfg, null, 2)
  } catch (e) {
    console.warn('load config failed', e)
    json.value = `// failed to load config: ${(e as Error).message}`
  }
}

onMounted(load)
</script>

<template>
  <div class="config-view">
    <div class="glass-card config-card">
      <div class="panel-header">
        <span class="panel-title">Active Config</span>
        <el-button size="small" @click="load">Refresh</el-button>
      </div>
      <pre class="json-display">{{ json }}</pre>
    </div>
  </div>
</template>

<style scoped>
.config-view {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 140px);
}

.config-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
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

.json-display {
  flex: 1;
  margin: 0;
  padding: 16px 20px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.55;
  color: var(--text-muted);
  white-space: pre-wrap;
  word-break: break-word;
  overflow-y: auto;
}
</style>