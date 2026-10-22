<template>
  <div class="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#0c0f16] px-5 py-10 text-slate-100">
    <!-- 极淡的背景光晕 -->
    <div class="pointer-events-none absolute top-1/2 left-1/2 h-[520px] w-[520px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-blue-600/[0.055] blur-[100px]"></div>

    <!-- 登录区域 -->
    <div class="relative z-10 w-full max-w-[400px]">
      <!-- 标题 -->
      <div class="mb-9 text-center">
        <div class="mx-auto mb-6 flex h-14 w-14 items-center justify-center rounded-2xl border border-white/[0.07] bg-[#1b2433] shadow-lg shadow-black/20">
          <svg class="h-7 w-7 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
            <rect x="4" y="4" width="16" height="6" rx="2" />
            <rect x="4" y="14" width="16" height="6" rx="2" />
            <path d="M8 7h.01M8 17h.01" stroke-width="3" stroke-linecap="round" />
          </svg>
        </div>

        <h1 class="text-[27px] font-semibold tracking-tight text-white">欢迎回来</h1>
        <p class="mt-3 text-sm text-slate-500">请输入您的账号信息以继续</p>
      </div>

      <!-- 登录卡片 -->
      <div class="rounded-[22px] border border-white/[0.075] bg-[#161b25] p-7 shadow-[0_25px_80px_rgba(0,0,0,0.24)] sm:p-9">
        <form class="space-y-5" @submit.prevent="doLogin">
          <!-- 用户名 -->
          <div>
            <label for="username" class="mb-2.5 block text-[13px] font-medium text-slate-300"> 用户名 </label>

            <div class="group relative">
              <span class="pointer-events-none absolute inset-y-0 left-4 flex items-center text-slate-500 transition-colors group-focus-within:text-blue-400">
                <svg class="h-[18px] w-[18px]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
                  <circle cx="12" cy="8" r="4" />
                  <path d="M4 21a8 8 0 0 1 16 0" />
                </svg>
              </span>

              <input
                id="username"
                v-model="username"
                name="username"
                type="text"
                autocomplete="username"
                required
                placeholder="请输入用户名"
                class="h-[49px] w-full rounded-xl border border-white/[0.08] bg-[#0e131c] pr-4 pl-12 text-sm text-white transition-all duration-200 outline-none placeholder:text-slate-600 hover:border-white/[0.16] focus:border-blue-500/70 focus:ring-[3px] focus:ring-blue-500/10"
              />
            </div>
          </div>

          <!-- 密码 -->
          <div>
            <label for="password" class="mb-2.5 block text-[13px] font-medium text-slate-300"> 密码 </label>

            <div class="group relative">
              <span class="pointer-events-none absolute inset-y-0 left-4 flex items-center text-slate-500 transition-colors group-focus-within:text-blue-400">
                <svg class="h-[18px] w-[18px]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
                  <rect x="4" y="10" width="16" height="11" rx="2" />
                  <path d="M8 10V7a4 4 0 0 1 8 0v3" />
                </svg>
              </span>

              <input
                id="password"
                v-model="password"
                name="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                required
                placeholder="请输入密码"
                class="h-[49px] w-full rounded-xl border border-white/[0.08] bg-[#0e131c] pr-12 pl-12 text-sm text-white transition-all duration-200 outline-none placeholder:text-slate-600 hover:border-white/[0.16] focus:border-blue-500/70 focus:ring-[3px] focus:ring-blue-500/10"
              />

              <!-- 显示密码 -->
              <button
                type="button"
                :aria-label="showPassword ? '隐藏密码' : '显示密码'"
                class="absolute inset-y-0 right-4 flex items-center text-slate-500 transition-colors hover:text-slate-300"
                @click="showPassword = !showPassword"
              >
                <svg class="h-[18px] w-[18px]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
                  <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
              </button>
            </div>
          </div>

          <!-- 错误提示 -->
          <p v-if="error" role="alert" class="text-center text-[13px] text-red-400">{{ error }}</p>

          <!-- 登录按钮 -->
          <button
            type="submit"
            :disabled="loading"
            class="group mt-2 flex h-[49px] w-full items-center justify-center gap-2.5 rounded-xl bg-[#316bec] text-sm font-semibold text-white shadow-lg shadow-blue-600/10 transition-all duration-200 hover:bg-[#407bfa] hover:shadow-blue-600/20 active:scale-[0.985] disabled:cursor-not-allowed disabled:opacity-60"
          >
            {{ loading ? '登录中...' : '登录' }}
            <svg v-if="!loading" class="h-4 w-4 transition-transform group-hover:translate-x-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path d="M5 12h14m-6-6 6 6-6 6" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </button>
        </form>
      </div>

      <!-- 简洁底部 -->
      <div class="mt-7 flex items-center justify-center gap-2 text-xs text-slate-600">
        <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
          <rect x="4" y="10" width="16" height="11" rx="2" />
          <path d="M8 10V7a4 4 0 0 1 8 0v3" />
        </svg>
        安全管理控制台
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '@/api/auth'
import { handleError } from '@/api'

const router = useRouter()
const username = ref('')
const password = ref('')
const showPassword = ref(false)
const loading = ref(false)
const error = ref('')

async function doLogin() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await authApi.login(username.value, password.value)
    localStorage.setItem('token', data.token)
    router.push('/')
  } catch (e) {
    error.value = handleError(e)
  } finally {
    loading.value = false
  }
}
</script>
