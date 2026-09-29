<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Node } from '@/api/control'
import { controlApi } from '@/api/control'
import NodeFormDialog from '@/components/NodeFormDialog.vue'

const nodes = ref<Node[]>([])
const dialogVisible = ref(false)
const editingNode = ref<Partial<Node> | null>(null)

async function load() {
  try {
    nodes.value = await controlApi.getNodes()
  } catch (e) {
    console.warn('load nodes failed', e)
  }
}

function openNew() {
  editingNode.value = null
  dialogVisible.value = true
}

function openEdit(node: Node) {
  editingNode.value = { ...node }
  dialogVisible.value = true
}

async function handleDelete(node: Node) {
  if (!confirm(`Delete node ${node.name}?`)) return
  try {
    await controlApi.deleteNode(node.id)
    await load()
  } catch (e) {
    console.error('delete failed', e)
  }
}

onMounted(load)
</script>

<template>
  <div class="nodes-view">
    <div class="glass-card">
      <div class="panel-header">
        <span class="panel-title">Nodes ({{ nodes.length }})</span>
        <el-button type="primary" @click="openNew">+ New Node</el-button>
      </div>

      <el-table :data="nodes" size="small">
        <el-table-column label="ID" prop="id" width="140" />
        <el-table-column label="Name" prop="name" min-width="180" />
        <el-table-column label="Role" prop="role" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.role }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Port" prop="listenPort" width="80" />
        <el-table-column label="Upstream" prop="upstreamURL" min-width="180">
          <template #default="{ row }">
            <span class="mono-text muted">{{ row.upstreamURL }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Status" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.status === 'online' ? 'success' : 'danger'"
              size="small"
            >
              {{ row.status ?? 'unknown' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row as Node)">Edit</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row as Node)">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <NodeFormDialog
      v-model:visible="dialogVisible"
      :node="editingNode"
      @saved="load"
    />
  </div>
</template>

<style scoped>
.nodes-view {
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
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.mono-text {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.muted {
  color: var(--text-muted);
}
</style>