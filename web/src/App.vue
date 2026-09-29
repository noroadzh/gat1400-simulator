<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import Sidebar from '@/components/Sidebar.vue'
import Topbar from '@/components/Topbar.vue'
import { controlWs } from '@/api/ws'

const emitWsRefresh = () => window.dispatchEvent(new CustomEvent('ws:event'))

onMounted(() => {
  controlWs.start()
  controlWs.on(emitWsRefresh)
})

onUnmounted(() => {
  controlWs.stop()
})
</script>

<template>
  <div class="app-shell">
    <Sidebar />
    <div class="main-area">
      <Topbar />
      <div class="page-content">
        <RouterView v-slot="{ Component }">
          <Transition name="fade" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </div>
    </div>
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  min-height: 100vh;
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  margin-left: 240px;
}

.page-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 768px) {
  .main-area {
    margin-left: 0;
  }
}
</style>