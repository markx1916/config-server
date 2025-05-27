import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css' // Element Plus styles
import * as ElementPlusIconsVue from '@element-plus/icons-vue' // Import all icons
import App from './App.vue'
import router from './router' // Our router configuration
import pinia from './store' // Import Pinia instance

const app = createApp(App)

// Register all Element Plus icons globally
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(pinia) // Use Pinia
app.use(ElementPlus)
app.use(router)

app.mount('#app')
