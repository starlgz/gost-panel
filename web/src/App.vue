<template>
  <div class="app">
    <nav v-if="isAuthed">
      <div class="brand">🖤 gost-panel</div>
      <div class="links">
        <router-link to="/" exact-active-class="active">转发规则</router-link>
        <router-link to="/status" active-class="active">状态</router-link>
        <router-link to="/audit" active-class="active">审计日志</router-link>
        <a href="#" @click.prevent="logout">退出</a>
      </div>
    </nav>
    <main>
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const isAuthed = computed(() => !!localStorage.getItem('token'))

function logout() {
  localStorage.removeItem('token')
  router.push('/login')
}
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
  background: #0f1115;
  color: #e6e8eb;
  min-height: 100vh;
}
.app { max-width: 1100px; margin: 0 auto; padding: 20px; }
nav {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 20px; background: #171a21; border-radius: 10px; margin-bottom: 20px;
}
.brand { font-weight: 700; font-size: 17px; letter-spacing: 0.5px; }
.links { display: flex; gap: 18px; }
.links a {
  color: #9aa4b2; text-decoration: none; font-size: 14px; padding: 6px 12px; border-radius: 6px;
}
.links a:hover { color: #fff; background: #232833; }
.links a.active { color: #fff; background: #2f6feb; }
main { background: #171a21; border-radius: 10px; padding: 24px; }
h1 { font-size: 18px; margin-bottom: 18px; }
</style>
