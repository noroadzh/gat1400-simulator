import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory('/'),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { title: 'Dashboard', icon: 'DataLine' }
    },
    {
      path: '/nodes',
      name: 'nodes',
      component: () => import('@/views/NodesView.vue'),
      meta: { title: 'Nodes', icon: 'Connection' }
    },
    {
      path: '/scenarios',
      name: 'scenarios',
      component: () => import('@/views/ScenariosView.vue'),
      meta: { title: 'Scenarios', icon: 'VideoCamera' }
    },
    {
      path: '/resources',
      name: 'resources',
      component: () => import('@/views/ResourcesView.vue'),
      meta: { title: 'Resources', icon: 'Files' }
    },
    {
      path: '/subscriptions',
      name: 'subscriptions',
      component: () => import('@/views/SubscriptionsView.vue'),
      meta: { title: 'Subscriptions', icon: 'Bell' }
    },
    {
      path: '/captures',
      name: 'captures',
      component: () => import('@/views/CapturesView.vue'),
      meta: { title: 'Captures', icon: 'FolderOpened' }
    },
    {
      path: '/config',
      name: 'config',
      component: () => import('@/views/ConfigView.vue'),
      meta: { title: 'Config', icon: 'Setting' }
    }
  ]
})

router.afterEach((to) => {
  if (to.meta?.title) {
    document.title = `${to.meta.title} · GA/T 1400 Simulator`
  }
})

export default router