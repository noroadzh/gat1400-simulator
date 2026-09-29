<script setup lang="ts">
import { computed } from 'vue'

const RESOURCES = [
  { id: 1, name: 'Video', uri: '/VIAS/api/v1/Video', desc: '视频' },
  { id: 2, name: 'Image', uri: '/VIAS/api/v1/Image', desc: '图像' },
  { id: 3, name: 'File', uri: '/VIAS/api/v1/File', desc: '文件' },
  { id: 4, name: 'Person', uri: '/VIAS/api/v1/Person', desc: '人员' },
  { id: 5, name: 'Face', uri: '/VIAS/api/v1/Face', desc: '人脸' },
  { id: 6, name: 'MotorVehicle', uri: '/VIAS/api/v1/MotorVehicle', desc: '机动车' },
  { id: 7, name: 'NonMotorVehicle', uri: '/VIAS/api/v1/NonMotorVehicle', desc: '非机动车' },
  { id: 8, name: 'Thing', uri: '/VIAS/api/v1/Thing', desc: '物品' },
  { id: 9, name: 'Scene', uri: '/VIAS/api/v1/Scene', desc: '场景' },
  { id: 10, name: 'TrafficEquipment', uri: '/VIAS/api/v1/TrafficEquipment', desc: '交通设备' },
  { id: 11, name: 'APE', uri: '/VIAS/api/v1/APE', desc: '机动车通行事件' },
  { id: 12, name: 'Alarm', uri: '/VIAS/api/v1/Alarm', desc: '警情' }
]

const rows = computed(() => RESOURCES)
</script>

<template>
  <div class="resources-view">
    <div class="glass-card">
      <div class="panel-header">
        <span class="panel-title">Resource Types ({{ rows.length }})</span>
      </div>

      <el-table :data="rows" size="small">
        <el-table-column label="ID" prop="id" width="80" />
        <el-table-column label="Name" prop="name" width="200" />
        <el-table-column label="URI" prop="uri">
          <template #default="{ row }">
            <span class="mono-text">{{ row.uri }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Description" prop="desc" width="160" />
      </el-table>
    </div>
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

.mono-text {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: var(--text-muted);
}
</style>