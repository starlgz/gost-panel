import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from '@/App.vue'
import Login from '@/views/Login.vue'
import Rules from '@/views/Rules.vue'
import Status from '@/views/Status.vue'
import Audit from '@/views/Audit.vue'
import Overview from '@/views/Overview.vue'
import '@/style.css'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/login', component: Login },
    { path: '/', component: Overview, meta: { auth: true } },
    { path: '/rules', component: Rules, meta: { auth: true } },
    { path: '/status', component: Status, meta: { auth: true } },
    { path: '/audit', component: Audit, meta: { auth: true } },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.meta.auth && !token) return '/login'
  if (to.path === '/login' && token) return '/'
})

createApp(App).use(router).mount('#app')
