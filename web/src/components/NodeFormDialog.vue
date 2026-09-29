<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Node } from '@/api/control'
import { controlApi } from '@/api/control'

const props = defineProps<{
  visible: boolean
  node?: Partial<Node> | null
}>()

const emit = defineEmits<{
  'update:visible': [val: boolean]
  saved: [node: Node]
}>()

const form = ref<Partial<Node>>({})
const saving = ref(false)

watch(
  () => props.visible,
  (v) => {
    if (v) {
      form.value = props.node ? { ...props.node } : {}
    }
  }
)

function close() {
  emit('update:visible', false)
}

async function save() {
  saving.value = true
  try {
    const result = await controlApi.upsertNode(form.value)
    emit('saved', result)
    close()
  } catch (e) {
    console.error('save node failed', e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="form?.id ? 'Edit Node' : 'New Node'"
    width="520px"
    @update:model-value="emit('update:visible', $event)"
    @closed="form = {}"
  >
    <el-form label-position="top" :model="form">
      <el-form-item label="Node ID" required>
        <el-input v-model="form.id" placeholder="e.g. vms-001" :disabled="!!form.id" />
      </el-form-item>
      <el-form-item label="Name" required>
        <el-input v-model="form.name" placeholder="e.g. Video Management System" />
      </el-form-item>
      <el-form-item label="Role">
        <el-select v-model="form.role" placeholder="Select role" style="width: 100%">
          <el-option label="IVS" value="IVS" />
          <el-option label="MGW" value="MGW" />
          <el-option label="RTM" value="RTM" />
          <el-option label="GATEWAY" value="GATEWAY" />
        </el-select>
      </el-form-item>
      <el-form-item label="Listen Port" required>
        <el-input-number v-model="form.listenPort" :min="1" :max="65535" style="width: 100%" />
      </el-form-item>
      <el-form-item label="Upstream URL">
        <el-input
          v-model="form.upstreamURL"
          placeholder="e.g. http://10.0.0.100:8080"
        />
      </el-form-item>
      <el-form-item label="Capabilities">
        <el-select v-model="form.capabilities" multiple placeholder="Select capabilities" style="width: 100%">
          <el-option label="FaceDetect" value="FaceDetect" />
          <el-option label="VehicleDetect" value="VehicleDetect" />
          <el-option label="TrafficEvent" value="TrafficEvent" />
          <el-option label="AlarmReport" value="AlarmReport" />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="close">Cancel</el-button>
      <el-button type="primary" :loading="saving" @click="save">Save</el-button>
    </template>
  </el-dialog>
</template>