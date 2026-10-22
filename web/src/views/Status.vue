<template>
  <div class="mx-auto max-w-[1440px] space-y-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="mb-2 text-xs font-medium uppercase tracking-[0.18em] text-blue-400/80">RUNTIME STATUS</p>
        <h1 class="text-2xl font-semibold tracking-tight text-white">运行状态</h1>
        <p class="mt-1.5 text-sm text-slate-500">gost 引擎与面板的实时健康状态，每 10 秒自动刷新</p>
      </div>
      <div class="flex items-center gap-2.5">
        <span class="text-xs text-slate-500">数据时间 {{ formatTime(st.panel?.time) || '等待数据' }}</span>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2 text-xs font-medium text-slate-300 transition-colors hover:border-white/[0.16] hover:bg-white/[0.07] hover:text-white"
          @click="refresh"
        >
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <path d="M20 11a8.1 8.1 0 0 0-14.9-4L3 9m0 0V4m0 5h5M4 13a8.1 8.1 0 0 0 14.9 4L21 15m0 0v5m0-5h-5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          刷新
        </button>
      </div>
    </header>

    <p v-if="error" role="alert" class="rounded-lg border border-red-400/20 bg-red-500/[0.08] px-4 py-3 text-sm text-red-300">{{ error }}</p>

    <!-- 健康指标 -->
    <section class="grid gap-3.5 sm:grid-cols-2 xl:grid-cols-4" aria-label="健康指标">
      <div v-for="metric in metrics" :key="metric.label" class="rounded-xl border border-white/[0.07] bg-[#161b25] p-5">
        <div class="flex items-start justify-between gap-3">
          <span class="text-sm text-slate-500">{{ metric.label }}</span>
          <span class="flex h-8 w-8 items-center justify-center rounded-lg" :class="metric.iconBg">
            <svg class="h-4 w-4" :class="metric.iconColor" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
              <path :d="metric.icon" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </span>
        </div>
        <div class="mt-4 flex items-end justify-between gap-3">
          <strong class="flex items-center gap-2.5 text-2xl font-semibold tracking-tight" :class="metric.valueClass">
            <span v-if="metric.pulse" class="relative inline-flex h-2.5 w-2.5">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-60" :class="metric.dotClass"></span>
              <span class="relative inline-flex h-2.5 w-2.5 rounded-full" :class="metric.dotClass"></span>
            </span>
            <span v-else-if="metric.dotClass" class="inline-block h-2.5 w-2.5 rounded-full" :class="metric.dotClass"></span>
            {{ metric.value }}
          </strong>
          <span class="text-xs" :class="metric.detailClass">{{ metric.detail }}</span>
        </div>
      </div>
    </section>

    <!-- 状态说明 -->
    <section class="grid gap-4 xl:grid-cols-[minmax(0,1.3fr)_minmax(320px,0.7fr)]">
      <div class="rounded-xl border border-white/[0.07] bg-[#161b25] p-5 sm:p-6">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="text-sm font-semibold text-white">指标说明</h2>
            <p class="mt-1 text-xs text-slate-500">各状态项的具体含义与影响</p>
          </div>
        </div>
        <div class="mt-6 space-y-4">
          <div v-for="item in explanations" :key="item.title" class="flex gap-3">
            <span class="mt-0.5 h-1.5 w-1.5 shrink-0 rounded-full" :class="item.dot"></span>
            <div>
              <p class="text-sm font-medium text-slate-200">{{ item.title }}</p>
              <p class="mt-1 text-xs leading-relaxed text-slate-500">{{ item.desc }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="rounded-xl border border-white/[0.07] bg-[#161b25] p-5 sm:p-6">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="text-sm font-semibold text-white">快速操作</h2>
            <p class="mt-1 text-xs text-slate-500">状态异常时的常用入口</p>
          </div>
        </div>
        <div class="mt-6 space-y-3">
          <router-link to="/rules" class="flex items-center justify-between rounded-lg border border-white/[0.06] bg-white/[0.02] px-3.5 py-3 transition-colors hover:bg-white/[0.05]">
            <span class="text-sm text-slate-300">转发规则</span>
            <span class="text-xs text-slate-500">检查并同步规则 →</span>
          </router-link>
          <router-link to="/audit" class="flex items-center justify-between rounded-lg border border-white/[0.06] bg-white/[0.02] px-3.5 py-3 transition-colors hover:bg-white/[0.05]">
            <span class="text-sm text-slate-300">审计日志</span>
            <span class="text-xs text-slate-500">查看近期变更 →</span>
          </router-link>
        </div>
        <div class="mt-5 flex items-center justify-between border-t border-white/[0.07] pt-4 text-xs">
          <span class="text-slate-500">面板版本</span>
          <span class="font-mono text-slate-300">{{ st.panel?.version || '-' }}</span>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { statusApi } from '@/api/status'
import { handleError } from '@/api'

const st = ref({})
const loaded = ref(false)
const error = ref('')

const metrics = computed(() => [
  {
    label: 'gost 引擎',
    value: st.value.gost?.active ? '运行中' : '已停止',
    valueClass: st.value.gost?.active ? 'text-emerald-300' : 'text-red-300',
    dotClass: st.value.gost?.active ? 'bg-emerald-400' : 'bg-red-400',
    pulse: true,
    detail: 'systemd 服务',
    detailClass: 'text-slate-500',
    icon: 'M13 10V3L4 14h7v7l9-11h-7Z', iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-300',
  },
  {
    label: 'gost API',
    value: st.value.gost?.api === 'up' ? '在线' : '离线',
    valueClass: st.value.gost?.api === 'up' ? 'text-emerald-300' : 'text-red-300',
    dotClass: st.value.gost?.api === 'up' ? 'bg-emerald-400' : 'bg-red-400',
    detail: '管理接口连通',
    detailClass: 'text-slate-500',
    icon: 'M3 12h4l3-8 4 16 3-8h4', iconBg: 'bg-blue-500/10', iconColor: 'text-blue-300',
  },
  {
    label: '活动规则',
    value: st.value.gost?.service_count ?? '-',
    valueClass: 'text-white',
    detail: '已加载 service',
    detailClass: 'text-slate-500',
    icon: 'M4 6h16M4 12h16M4 18h16', iconBg: 'bg-violet-500/10', iconColor: 'text-violet-300',
  },
  {
    label: '面板版本',
    value: st.value.panel?.version || '-',
    valueClass: 'text-white',
    detail: 'panel release',
    detailClass: 'text-slate-500',
    icon: 'M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42zM7.5 7.5h.01', iconBg: 'bg-amber-500/10', iconColor: 'text-amber-300',
  },
])

const explanations = [
  { title: 'gost 引擎', desc: 'systemd 服务是否处于 active 状态；停止时所有转发均不可用，需登录服务器排查。', dot: 'bg-emerald-400' },
  { title: 'gost API', desc: '面板与 gost 管理 API（默认 127.0.0.1:18080）的连通性；离线时无法同步规则。', dot: 'bg-blue-400' },
  { title: '活动规则', desc: 'gost 当前实际加载的 service 数量，与「转发规则」页同步后的数量应一致。', dot: 'bg-violet-400' },
]

function formatTime(value) {
  if (!value) return ''
  return String(value).replace('T', ' ').slice(0, 19)
}

async function refresh() {
  try {
    const { data } = await statusApi.get()
    st.value = data
    error.value = ''
    loaded.value = true
  } catch (e) {
    error.value = handleError(e)
  }
}

let timer
onMounted(() => {
  refresh()
  timer = setInterval(refresh, 10000)
})
onUnmounted(() => clearInterval(timer))
</script>
