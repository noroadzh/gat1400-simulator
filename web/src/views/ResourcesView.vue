<script setup lang="ts">
/**
 * GA/T 1400.4 资源对象浏览器（数据驱动）。
 *
 * 交互流程：
 *   1. 打开自动 GET /api/control/resources，列出 12 种 Kind 卡片（含计数）
 *   2. 点击卡片选中（URL ?kind=Person 同步）→ 调 GET /api/control/resources/Person/list
 *   3. 行内字段上展示 ID 与原始 JSON；行尾支持删除（el-popconfirm）
 *   4. 顶部操作栏支持搜索（按 ID 字段名过滤）、POST 测试（手动注入资源对象）
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  controlApi,
  type ResourceKindMeta,
} from '../api/control'
import { KIND_METAS, extractObjects, pickID } from '../api/resources-meta'

const route = useRoute()
const router = useRouter()

const kinds = ref<ResourceKindMeta[]>([])
const selectedKind = ref<string>('')
// 挂载期间由 onMounted 显式触发首次 loadList，watch 不再响应，避免双打。
let inited = false
const listLoading = ref(false)
const envelope = ref<Record<string, unknown>>({})
const search = ref('')

// POST 测试对话框
const createDialogVisible = ref(false)
const createPayloadText = ref('')
const createLoading = ref(false)

// 选中 Kind 的元数据
const selectedMeta = computed<ResourceKindMeta | undefined>(() =>
  kinds.value.find((k) => k.kind === selectedKind.value),
)

// fallback 表（用于首屏 kinds 还没加载完成时的卡片渲染）
const fallbackMetas = computed(() => KIND_METAS)

// 解析后的对象数组
const objects = computed<Record<string, unknown>[]>(() => {
  if (!selectedMeta.value) return []
  return extractObjects(envelope.value, selectedMeta.value.kind)
})

// 主键字段
const idField = computed(() => selectedMeta.value?.idField ?? '')

// 搜索后的对象
const filteredObjects = computed<Record<string, unknown>[]>(() => {
  if (!search.value.trim()) return objects.value
  const kw = search.value.trim().toLowerCase()
  return objects.value.filter((o) => pickID(o, idField.value).toLowerCase().includes(kw))
})

async function loadKinds(): Promise<void> {
  try {
    const r = await controlApi.listResourceKinds()
    kinds.value = r.kinds
    // 仅在 selectedKind 为空时落默认值，不主动触发 selectedKind 变化（避免
    // 与 watch(selectedKind) 重复触发 loadList）。
    if (!selectedKind.value && kinds.value.length > 0) {
      selectKind(kinds.value[0].kind)
    }
  } catch (e) {
    ElMessage.error(`加载资源对象类型失败：${(e as Error).message}`)
  }
}

async function loadList(): Promise<void> {
  if (!selectedKind.value) return
  listLoading.value = true
  try {
    envelope.value = await controlApi.listResources(selectedKind.value)
  } catch (e) {
    ElMessage.error(`加载资源对象列表失败：${(e as Error).message}`)
    envelope.value = {}
  } finally {
    listLoading.value = false
  }
}

function selectKind(kind: string): void {
  selectedKind.value = kind
  router.replace({ query: { ...route.query, kind } })
}

async function handleDelete(obj: Record<string, unknown>): Promise<void> {
  const id = pickID(obj, idField.value)
  if (!id) {
    ElMessage.warning('对象缺少主键字段，无法删除')
    return
  }
  try {
    await controlApi.deleteResource(selectedKind.value, id)
    ElMessage.success(`已删除 ${selectedKind.value} ${id}`)
    await loadList()
    await loadKinds() // 刷新计数
  } catch (e) {
    ElMessage.error(`删除失败：${(e as Error).message}`)
  }
}

function openCreateDialog(): void {
  if (!selectedMeta.value) return
  // 预填一个最小可用的对象
  createPayloadText.value = JSON.stringify(
    {
      [`${selectedMeta.value.idField}`]: `${selectedMeta.value.kind.toUpperCase()}_000001`,
      Name: '示例资源对象',
    },
    null,
    2,
  )
  createDialogVisible.value = true
}

async function submitCreate(): Promise<void> {
  if (!selectedMeta.value) return
  let body: unknown
  try {
    body = JSON.parse(createPayloadText.value)
  } catch (e) {
    ElMessage.error(`JSON 解析失败：${(e as Error).message}`)
    return
  }
  createLoading.value = true
  try {
    // 协议端要求 <Kind>List.<Kind>Object[] 信封；前端组装这一层。
    const kind = selectedMeta.value.kind
    const envelopePayload = {
      [`${kind}List`]: {
        [`${kind}Object`]: Array.isArray(body) ? body : [body],
      },
    }
    await controlApi.createResource(kind, envelopePayload)
    ElMessage.success('资源对象已写入')
    createDialogVisible.value = false
    await loadList()
    await loadKinds()
  } catch (e) {
    ElMessage.error(`写入失败：${(e as Error).message}`)
  } finally {
    createLoading.value = false
  }
}

function uriFor(kind: string): string {
  const meta = kinds.value.find((k) => k.kind === kind) ?? fallbackMetas.value.find((m) => m.kind === kind)
  return meta ? `/VIID/${meta.collection}` : ''
}

onMounted(async () => {
  const queryKind = (route.query.kind as string | undefined) ?? ''
  if (queryKind) selectedKind.value = queryKind
  // loadKinds 期间 selectedKind 可能被改成默认值，此时 watch 会被 inited=false 挡掉，
  // 由下面的显式 loadList 统一发起一次请求，避免首屏双打。
  await loadKinds()
  inited = true
  await loadList()
})

watch(selectedKind, async () => {
  // 挂载期间由 onMounted 显式 loadList，避免双打。
  if (!inited) return
  await loadList()
})
</script>

<template>
  <div class="resources-view">
    <!-- 顶部 12 卡片网格 -->
    <div class="glass-card kind-grid-panel">
      <div class="panel-header">
        <div class="panel-title">资源对象类型（{{ kinds.length || fallbackMetas.length }} 种）</div>
        <div class="panel-sub">URI 前缀：/VIID/&lt;Collection&gt;（与协议端实现一致）</div>
      </div>
      <div class="kind-grid">
        <div
          v-for="meta in (kinds.length ? kinds : fallbackMetas)"
          :key="meta.kind"
          class="kind-card"
          :class="{ active: selectedKind === meta.kind }"
          @click="selectKind(meta.kind)"
        >
          <div class="kind-name">{{ meta.kind }}</div>
          <div class="kind-desc">{{ meta.description || meta.kind }}</div>
          <div class="kind-uri mono-text">{{ uriFor(meta.kind) }}</div>
          <div class="kind-count">
            <el-tag size="small" :type="(kinds.length ? (meta as ResourceKindMeta).count : 0) > 0 ? 'success' : 'info'">
              {{ kinds.length ? (meta as ResourceKindMeta).count : '—' }}
            </el-tag>
          </div>
        </div>
      </div>
    </div>

    <!-- 选中 Kind 后的列表面板 -->
    <div class="glass-card list-panel" v-if="selectedMeta">
      <div class="panel-header">
        <div class="panel-title-row">
          <div class="panel-title">{{ selectedMeta.kind }}（{{ objects.length }} 条）</div>
          <div class="panel-actions">
            <el-input
              v-model="search"
              placeholder="按主键搜索"
              clearable
              size="small"
              style="width: 200px"
            />
            <el-button size="small" type="primary" @click="openCreateDialog">POST 测试</el-button>
          </div>
        </div>
      </div>
      <el-table
        :data="filteredObjects"
        v-loading="listLoading"
        stripe
        size="small"
        empty-text="暂无对象，点击「POST 测试」注入一条"
      >
        <el-table-column
          :label="selectedMeta.idField"
          width="200"
        >
          <template #default="{ row }">
            <span class="mono-text">{{ pickID(row, selectedMeta.idField) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="原始 JSON" min-width="320">
          <template #default="{ row }">
            <pre class="json-cell">{{ JSON.stringify(row, null, 2) }}</pre>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="right">
          <template #default="{ row }">
            <el-popconfirm
              :title="`确认删除 ${pickID(row, selectedMeta.idField)}?`"
              @confirm="handleDelete(row)"
            >
              <template #reference>
                <el-button size="small" type="danger" plain>删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- POST 测试对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      :title="`POST 测试 · ${selectedMeta?.kind ?? ''}`"
      width="560px"
      destroy-on-close
    >
      <div class="dialog-hint mono-text">
        协议端标准信封：
        <pre>{{ `{ "${selectedMeta?.kind ?? ''}_List": { "${selectedMeta?.kind ?? ''}_Object": [<your object>] } }` }}</pre>
        上方已预填一个最小可用对象（仅含主键与 Name），可按需修改后提交。
      </div>
      <el-input
        v-model="createPayloadText"
        type="textarea"
        :rows="14"
        spellcheck="false"
      />
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="createLoading" @click="submitCreate">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.resources-view {
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
.panel-sub {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 4px;
}
.panel-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.panel-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}
.mono-text {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: var(--text-muted);
}
.kind-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
  padding: 14px 16px;
}
.kind-card {
  padding: 12px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--card-border);
  cursor: pointer;
  transition: all 0.15s ease;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.kind-card:hover {
  background: rgba(255, 255, 255, 0.06);
}
.kind-card.active {
  border-color: var(--el-color-primary);
  background: rgba(64, 158, 255, 0.1);
}
.kind-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.kind-desc {
  font-size: 11px;
  color: var(--text-muted);
}
.kind-uri {
  font-size: 10px;
  margin-top: 2px;
}
.kind-count {
  margin-top: 4px;
}
.json-cell {
  margin: 0;
  padding: 8px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  font-size: 11px;
  max-height: 120px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
.dialog-hint {
  margin-bottom: 10px;
  font-size: 11px;
  color: var(--text-muted);
}
.dialog-hint pre {
  margin: 4px 0;
  padding: 6px 8px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>