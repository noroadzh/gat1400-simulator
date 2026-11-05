<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { controlApi } from '@/api/control'
import { controlWs } from '@/api/ws'

const subscribes = ref<unknown[]>([])
const dispositions = ref<unknown[]>([])
const subNotifs = ref<unknown[]>([])
const dispNotifs = ref<unknown[]>([])
const error = ref('')
const loading = ref(false)
const createOpen = ref(false)
const createKind = ref<'subscribe' | 'disposition'>('subscribe')
const createBody = ref('{\n  "SubscribeID": "S-demo",\n  "Title": "demo"\n}')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [subs, disps, sn, dn] = await Promise.all([
      controlApi.getSubscriptions(),
      controlApi.getDispositions(),
      controlApi.getSubscribeNotifications(),
      controlApi.getDispositionNotifications(),
    ])
    subscribes.value = Array.isArray(subs.items) ? subs.items : []
    dispositions.value = Array.isArray(disps.items) ? disps.items : []
    subNotifs.value = Array.isArray(sn.items) ? sn.items : []
    dispNotifs.value = Array.isArray(dn.items) ? dn.items : []
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

async function submitCreate() {
  try {
    const body = JSON.parse(createBody.value)
    if (createKind.value === 'subscribe') await controlApi.postSubscribe(body)
    else await controlApi.postDisposition(body)
    ElMessage.success('created')
    createOpen.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function removeSub(row: Record<string, unknown>) {
  const id = String(row.SubscribeID ?? '')
  if (!id) return
  await ElMessageBox.confirm(`Delete subscribe ${id}?`, 'Confirm')
  await controlApi.deleteSubscribe(id)
  await load()
}

async function removeDisp(row: Record<string, unknown>) {
  const id = String(row.DispositionID ?? '')
  if (!id) return
  await ElMessageBox.confirm(`Delete disposition ${id}?`, 'Confirm')
  await controlApi.deleteDisposition(id)
  await load()
}

let offWs: (() => void) | null = null
onMounted(() => {
  offWs = controlWs.on(() => {
    void load()
  })
  void load()
})
onUnmounted(() => {
  offWs?.()
  offWs = null
})
</script>

<template>
  <div class="subs-view">
    <div class="toolbar glass-card">
      <el-button type="primary" size="small" @click="createKind = 'subscribe'; createBody = '{\n  \"SubscribeID\": \"S-demo\",\n  \"Title\": \"demo\"\n}'; createOpen = true">New Subscribe</el-button>
      <el-button size="small" @click="createKind = 'disposition'; createBody = '{\n  \"DispositionID\": \"D-demo\",\n  \"Title\": \"alarm\"\n}'; createOpen = true">New Disposition</el-button>
      <el-button size="small" :loading="loading" @click="load">Refresh</el-button>
      <p v-if="error" class="err">{{ error }}</p>
    </div>

    <div class="grid">
      <div class="glass-card">
        <div class="panel-header">
          <span class="panel-title">Subscribes</span>
          <span class="panel-count">{{ subscribes.length }}</span>
        </div>
        <el-table :data="subscribes as Record<string, unknown>[]" size="small" height="280">
          <el-table-column prop="SubscribeID" label="ID" min-width="120" />
          <el-table-column prop="Title" label="Title" min-width="100" />
          <el-table-column label="" width="80">
            <template #default="{ row }">
              <el-button link type="danger" size="small" @click="removeSub(row)">Del</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="glass-card">
        <div class="panel-header">
          <span class="panel-title">Dispositions</span>
          <span class="panel-count">{{ dispositions.length }}</span>
        </div>
        <el-table :data="dispositions as Record<string, unknown>[]" size="small" height="280">
          <el-table-column prop="DispositionID" label="ID" min-width="120" />
          <el-table-column prop="Title" label="Title" min-width="100" />
          <el-table-column label="" width="80">
            <template #default="{ row }">
              <el-button link type="danger" size="small" @click="removeDisp(row)">Del</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="glass-card">
        <div class="panel-header">
          <span class="panel-title">Subscribe Notifications</span>
          <span class="panel-count">{{ subNotifs.length }}</span>
        </div>
        <pre class="json-display">{{ JSON.stringify(subNotifs, null, 2) }}</pre>
      </div>

      <div class="glass-card">
        <div class="panel-header">
          <span class="panel-title">Disposition Notifications</span>
          <span class="panel-count">{{ dispNotifs.length }}</span>
        </div>
        <pre class="json-display">{{ JSON.stringify(dispNotifs, null, 2) }}</pre>
      </div>
    </div>

    <el-dialog v-model="createOpen" :title="createKind === 'subscribe' ? 'Create Subscribe' : 'Create Disposition'" width="560px">
      <el-input v-model="createBody" type="textarea" :rows="10" />
      <template #footer>
        <el-button @click="createOpen = false">Cancel</el-button>
        <el-button type="primary" @click="submitCreate">Submit</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.subs-view { display: flex; flex-direction: column; gap: 14px; }
.toolbar { display: flex; gap: 8px; align-items: center; padding: 12px 14px; flex-wrap: wrap; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
@media (max-width: 1024px) { .grid { grid-template-columns: 1fr; } }
.panel-header { display: flex; justify-content: space-between; padding: 12px 14px; border-bottom: 1px solid var(--card-border); }
.panel-title { font-weight: 600; }
.panel-count { opacity: 0.7; font-size: 12px; }
.json-display { margin: 0; padding: 12px; max-height: 260px; overflow: auto; font-size: 12px; }
.err { color: var(--el-color-danger); margin: 0; }
</style>
