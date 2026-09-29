<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'

const collapsed = ref(false)
const route = useRoute()

const navItems = [
  { path: '/', label: 'Dashboard', icon: 'DataLine' },
  { path: '/nodes', label: 'Nodes', icon: 'Connection' },
  { path: '/scenarios', label: 'Scenarios', icon: 'VideoCamera' },
  { path: '/resources', label: 'Resources', icon: 'Files' },
  { path: '/subscriptions', label: 'Subscriptions', icon: 'Bell' },
  { path: '/captures', label: 'Captures', icon: 'FolderOpened' },
  { path: '/config', label: 'Config', icon: 'Setting' }
]
</script>

<template>
  <nav class="sidebar glass-card" :class="{ collapsed }">
    <div class="sidebar-header">
      <div class="logo-mark">
        <svg width="28" height="28" viewBox="0 0 32 32">
          <defs>
            <linearGradient id="logo-g" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#5b8cff" />
              <stop offset="100%" stop-color="#8a5bff" />
            </linearGradient>
          </defs>
          <rect width="32" height="32" rx="6" fill="#0b1020" />
          <path
            d="M8 8h16v3H11v3h10v3H11v3h10v3H11v3h13"
            fill="none"
            stroke="url(#logo-g)"
            stroke-width="2.2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </div>
      <span v-if="!collapsed" class="logo-text gradient-text">GAT1400</span>
    </div>

    <ul class="nav-list">
      <li v-for="item in navItems" :key="item.path">
        <RouterLink
          :to="item.path"
          class="nav-item"
          :class="{ active: route.path === item.path }"
        >
          <span class="nav-icon">
            <component :is="item.icon" />
          </span>
          <span v-if="!collapsed" class="nav-label">{{ item.label }}</span>
        </RouterLink>
      </li>
    </ul>

    <button class="collapse-btn" @click="collapsed = !collapsed">
      <component :is="collapsed ? 'DArrowRight' : 'DArrowLeft'" />
    </button>
  </nav>
</template>

<style scoped>
.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  height: 100vh;
  width: 240px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-radius: 0;
  border-right: 1px solid var(--card-border);
  border-top: none;
  border-left: none;
  border-bottom: none;
  z-index: 100;
  transition: width 0.3s ease;
  overflow: hidden;
}

.sidebar.collapsed {
  width: 64px;
}

.sidebar-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 16px 12px;
  border-bottom: 1px solid var(--card-border);
  margin-bottom: 8px;
}

.logo-text {
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.05em;
  white-space: nowrap;
}

.nav-list {
  list-style: none;
  margin: 0;
  padding: 0 8px;
  flex: 1;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  color: var(--text-muted);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s ease;
  margin-bottom: 2px;
  white-space: nowrap;
}

.nav-item:hover {
  background: rgba(91, 140, 255, 0.1);
  color: var(--text-primary);
  transform: translateX(2px);
}

.nav-item.active {
  background: rgba(91, 140, 255, 0.15);
  color: var(--primary);
  border-left: 2px solid var(--primary);
}

.nav-icon {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  width: 20px;
}

.nav-label {
  flex: 1;
}

.collapse-btn {
  margin: 8px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--card-border);
  border-radius: 8px;
  color: var(--text-muted);
  cursor: pointer;
  padding: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.collapse-btn:hover {
  background: rgba(91, 140, 255, 0.1);
  color: var(--text-primary);
}
</style>