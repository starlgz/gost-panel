<template>
  <div class="login-wrap">
    <form class="login-card" @submit.prevent="doLogin">
      <h2>🖤 gost-panel</h2>
      <p class="sub">GOST 转发管理面板</p>
      <input v-model="username" placeholder="用户名" autocomplete="username" />
      <input v-model="password" type="password" placeholder="密码" autocomplete="current-password" />
      <button :disabled="loading">{{ loading ? '登录中...' : '登录' }}</button>
      <p v-if="error" class="error">{{ error }}</p>
    </form>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function doLogin() {
  loading.value = true
  error.value = ''
  try {
    const r = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value, password: password.value }),
    })
    const d = await r.json()
    if (r.ok) {
      localStorage.setItem('token', d.token)
      router.push('/')
    } else {
      error.value = d.detail || '登录失败'
    }
  } catch {
    error.value = '无法连接服务器'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap { display: flex; justify-content: center; padding-top: 80px; }
.login-card {
  background: #1c2029; padding: 36px; border-radius: 12px; width: 320px;
  display: flex; flex-direction: column; gap: 12px;
}
h2 { text-align: center; margin: 0; }
.sub { text-align: center; color: #8a94a3; font-size: 13px; margin-bottom: 10px; }
input {
  padding: 10px 12px; border-radius: 8px; border: 1px solid #2c3340;
  background: #12151b; color: #e6e8eb; font-size: 14px;
}
button {
  padding: 11px; border: none; border-radius: 8px; background: #2f6feb;
  color: #fff; font-size: 15px; cursor: pointer; margin-top: 6px;
}
button:disabled { opacity: 0.6; }
.error { color: #f85149; font-size: 13px; text-align: center; }
</style>
