<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { controlWs } from '@/api/ws'

const route = useRoute()

const wsStatus = computed(() => {
  const s = controlWs.status
  if (s === 'open') return { text: 'live · ws ok', color: 'var(--success)' }
  if (s === 'connecting') return { text: 'live · connecting…', color: 'var(--warn)' }
  return { text: 'offline', color: 'var(--danger)' }
})
</script>

<template>
  <header class="topbar glass-card">
    <div class="topbar-left">
      <h1 class="page-title gradient-text">
        {{ (route.meta.title as string) ?? 'GA/T 1400 Simulator' }}
      </h1>
    </div>
    <div class="topbar-right">
      <span class="ws-badge" :style="{ '--ws-color': wsStatus.color }">
        <span class="ws-dot"></span>
        {{ wsStatus.text }}
      </span>
    </div>
  </header>
</template>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 24px;
  border-radius: 0;
  border-left: none;
  border-right: none;
  border-top: none;
  backdrop-filter: blur(14px);
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  margin: 0;
  letter-spacing: 0.02em;
}

.ws-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--ws-color, var(--text-muted));
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 20px;
  padding: 4px 12px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.ws-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--ws-color, var(--danger));
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(0.85); }
}
</style>