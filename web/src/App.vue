<template>
  <!-- 登录页：全屏独占，无侧边栏 -->
  <router-view v-if="isLogin" />

  <!-- 登录后布局：桌面左侧边栏固定；移动端抽屉式 + 顶部栏 -->
  <div v-else class="flex h-screen overflow-hidden bg-[#0c0f16] text-slate-100">
    <!-- 移动端遮罩 -->
    <div
      v-if="mobileOpen"
      class="fixed inset-0 z-30 bg-black/60 lg:hidden"
      @click="mobileOpen = false"
    ></div>

    <!-- 侧边栏 -->
    <aside
      class="fixed inset-y-0 left-0 z-40 flex h-full shrink-0 flex-col border-r border-white/[0.07] bg-[#12161f] transition-[width,transform] duration-200 lg:static lg:translate-x-0"
      :class="[
        collapsed ? 'w-[68px]' : 'w-60',
        mobileOpen ? 'translate-x-0' : '-translate-x-full',
      ]"
    >
      <!-- 品牌区 -->
      <div class="flex items-center px-4 py-6" :class="collapsed ? 'justify-center' : 'gap-3'">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-white/[0.07] bg-[#1b2433]">
          <svg class="h-5 w-5 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
            <rect x="4" y="4" width="16" height="6" rx="2" />
            <rect x="4" y="14" width="16" height="6" rx="2" />
            <path d="M8 7h.01M8 17h.01" stroke-width="3" stroke-linecap="round" />
          </svg>
        </div>
        <div v-if="!collapsed" class="min-w-0">
          <p class="text-[15px] font-semibold tracking-tight text-white">gost-panel</p>
          <p class="mt-0.5 text-xs text-slate-500">转发管理面板</p>
        </div>
      </div>

      <!-- 导航 -->
      <nav class="mt-2 flex-1 space-y-1 px-3" :class="collapsed && 'px-3.5'">
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          :title="collapsed ? item.label : undefined"
          class="flex items-center rounded-xl text-sm transition-colors duration-150"
          :class="[
            collapsed ? 'justify-center px-0 py-2.5' : 'gap-3 px-3.5 py-2.5',
            isActive(item.to)
              ? 'bg-blue-600/[0.14] font-medium text-blue-300'
              : 'text-slate-400 hover:bg-white/[0.04] hover:text-slate-200',
          ]"
          @click="mobileOpen = false"
        >
          <svg class="h-[18px] w-[18px] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
            <path :d="item.icon" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span v-if="!collapsed">{{ item.label }}</span>
        </router-link>
      </nav>

      <!-- 底部：折叠开关 + 退出 -->
      <div class="space-y-1 border-t border-white/[0.07] p-3" :class="collapsed && 'px-3.5'">
        <button
          :title="collapsed ? '展开侧边栏' : '收起侧边栏'"
          class="flex w-full items-center rounded-xl px-3.5 py-2.5 text-sm text-slate-500 transition-colors duration-150 hover:bg-white/[0.04] hover:text-slate-200"
          :class="collapsed && 'justify-center px-0'"
          @click="collapsed = !collapsed"
        >
          <svg class="h-[18px] w-[18px] shrink-0 transition-transform duration-200" :class="collapsed && 'rotate-180'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
            <path d="M11 19l-7-7 7-7m8 14-7-7 7-7" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span v-if="!collapsed" class="ml-3">收起</span>
        </button>

        <button
          title="修改密码"
          class="flex w-full items-center rounded-xl px-3.5 py-2.5 text-sm text-slate-400 transition-colors duration-150 hover:bg-white/[0.04] hover:text-slate-200"
          :class="collapsed && 'justify-center px-0'"
          @click="showChangePass = true"
        >
          <svg class="h-[18px] w-[18px] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
            <path d="M12 15v2m-6 4h12a2 2 0 0 0 2-2v-6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2Zm10-10V7a4 4 0 0 0-8 0v2h8Z" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span v-if="!collapsed" class="ml-3">修改密码</span>
        </button>

        <button
          title="退出登录"
          class="flex w-full items-center rounded-xl px-3.5 py-2.5 text-sm text-slate-400 transition-colors duration-150 hover:bg-white/[0.04] hover:text-slate-200"
          :class="collapsed && 'justify-center px-0'"
          @click="logout"
        >
          <svg class="h-[18px] w-[18px] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
            <path d="M15 17l5-5-5-5M20 12H9m3 8H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h6" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span v-if="!collapsed" class="ml-3">退出登录</span>
        </button>
      </div>
    </aside>

    <!-- 右侧：顶部栏 + 内容区 -->
    <div class="flex min-w-0 flex-1 flex-col">
      <!-- 移动端顶部栏 -->
      <header class="flex h-14 shrink-0 items-center gap-3 border-b border-white/[0.07] bg-[#12161f]/90 px-4 backdrop-blur lg:hidden">
        <button
          type="button"
          aria-label="打开菜单"
          class="flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-white/[0.06] hover:text-slate-200"
          @click="mobileOpen = true"
        >
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <path d="M4 6h16M4 12h16M4 18h16" stroke-linecap="round" />
          </svg>
        </button>
        <span class="text-[15px] font-semibold tracking-tight text-white">gost-panel</span>
      </header>

      <!-- 内容区 -->
      <main class="min-w-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6 sm:py-6 lg:px-8 lg:py-7">
        <router-view />
      </main>
    </div>

    <!-- 修改密码弹窗 -->
    <Modal v-model="showChangePass" title="修改面板密码" :busy="changingPass">
      <div class="flex flex-col gap-3.5">
        <p v-if="passMsg" class="rounded-lg border px-3 py-2 text-xs" :class="passErr ? 'border-red-400/20 bg-red-500/[0.08] text-red-300' : 'border-emerald-400/20 bg-emerald-500/[0.08] text-emerald-300'">{{ passMsg }}</p>
        <label class="label">当前密码
          <input v-model="passForm.old" type="password" class="field" placeholder="输入当前密码" />
        </label>
        <label class="label">新密码
          <input v-model="passForm.new" type="password" class="field" placeholder="至少 6 位" />
        </label>
        <label class="label">确认新密码
          <input v-model="passForm.confirm" type="password" class="field" placeholder="再次输入新密码" />
        </label>
      </div>
      <template #footer>
        <button type="button" class="btn" @click="showChangePass = false">取消</button>
        <button type="button" class="btn btn-primary" :disabled="changingPass" @click="doChangePass">{{ changingPass ? '提交中...' : '确认修改' }}</button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Modal from '@/components/Modal.vue'
import { authApi } from '@/api/auth'
import { handleError } from '@/api'

const route = useRoute()
const router = useRouter()

const isLogin = computed(() => route.path === '/login')

// 移动端抽屉开关；桌面端折叠状态持久化
const mobileOpen = ref(false)
const collapsed = ref(localStorage.getItem('sidebar_collapsed') === '1')
watch(collapsed, (v) => localStorage.setItem('sidebar_collapsed', v ? '1' : '0'))

const navItems = [
  { to: '/', label: '数据面板', icon: 'M4 19V5m0 14h16M8 16v-4m4 4V8m4 8v-7m4 7V4' },
  { to: '/rules', label: '转发规则', icon: 'M4 6h16M4 12h16M4 18h16' },
  { to: '/status', label: '运行状态', icon: 'M3 12h4l3-8 4 16 3-8h4' },
  { to: '/audit', label: '审计日志', icon: 'M5 5h14v16l-3-2-2 2-2-2-2 2-3-2V5zM9 10h6M9 14h4' },
]

// / 是数据面板首页，用精确匹配；其余前缀匹配即可
function isActive(to) {
  return to === '/' ? route.path === '/' : route.path.startsWith(to)
}

function logout() {
  localStorage.removeItem('token')
  router.push('/login')
}

// 修改密码
const showChangePass = ref(false)
const changingPass = ref(false)
const passForm = ref({ old: '', new: '', confirm: '' })
const passMsg = ref('')
const passErr = ref(false)

async function doChangePass() {
  passMsg.value = ''
  if (passForm.value.new.length < 6) {
    passMsg.value = '新密码至少 6 位'
    passErr.value = true
    return
  }
  if (passForm.value.new !== passForm.value.confirm) {
    passMsg.value = '两次输入的新密码不一致'
    passErr.value = true
    return
  }
  changingPass.value = true
  try {
    const { data } = await authApi.changePassword(passForm.value.old, passForm.value.new)
    passMsg.value = data.message || '密码已修改，请重新登录'
    passErr.value = false
    setTimeout(() => {
      showChangePass.value = false
      logout()
    }, 1500)
  } catch (e) {
    passMsg.value = handleError(e)
    passErr.value = true
  } finally {
    changingPass.value = false
  }
}
</script>
