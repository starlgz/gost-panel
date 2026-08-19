import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Login from './views/Login.vue'
import Rules from './views/Rules.vue'
import Status from './views/Status.vue'
import Audit from './views/Audit.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    { path: '/', component: Rules, meta: { auth: true } },
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
