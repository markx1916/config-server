import { createApp } from 'vue'
import { createPinia } from 'pinia'

import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css' // Import Element Plus CSS
// If you want to use SCSS variables for Element Plus, you might need a different import strategy
// e.g., import 'element-plus/theme-chalk/src/index.scss' (requires sass-loader)

import App from './App.vue'
import router from './router' // Will be created in src/router/index.ts

// Import global SCSS styles if any - create this file later if needed
// import './assets/styles/main.scss'


const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ElementPlus)

app.mount('#app')
