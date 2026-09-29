import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'
import './styles/global.css'

const app = createApp(App)
app.use(ElementPlus)
app.use(router)
// 全局注册全部 Element Plus 图标，让 Sidebar/Topbar/StatCard 等使用
// 字符串名称（<component :is="'DataLine'">）渲染时不会因为静态 import
// 缺失而出现空白图标。
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}
app.mount('#app')