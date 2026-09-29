<script setup lang="ts">
defineProps<{
  label: string
  value: number | string
  color?: 'primary' | 'success' | 'warn' | 'danger'
  unit?: string
  icon?: string
}>()

const colorMap = {
  primary: 'var(--primary)',
  success: 'var(--success)',
  warn: 'var(--warn)',
  danger: 'var(--danger)'
}
</script>

<template>
  <div class="stat-card glass-card">
    <div class="stat-label">{{ label }}</div>
    <div
      class="stat-value"
      :style="{
        background: color ? colorMap[color] : 'var(--gradient-primary)',
        '-webkit-background-clip': color ? 'text' : 'text',
        'background-clip': 'text',
        '-webkit-text-fill-color': 'transparent'
      }"
    >
      {{ value }}<span v-if="unit" class="stat-unit">{{ unit }}</span>
    </div>
    <div v-if="icon" class="stat-icon">
      <component :is="icon" />
    </div>
  </div>
</template>

<style scoped>
.stat-card {
  position: relative;
  overflow: hidden;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  cursor: default;
}

.stat-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 12px 32px rgba(91, 140, 255, 0.15);
}

.stat-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  line-height: 1;
}

.stat-unit {
  font-size: 14px;
  font-weight: 500;
  margin-left: 2px;
  opacity: 0.7;
}

.stat-icon {
  position: absolute;
  top: 12px;
  right: 14px;
  opacity: 0.15;
  color: var(--primary);
  font-size: 36px;
}
</style>