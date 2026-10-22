<template>
  <div class="mx-auto max-w-[1440px] space-y-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="mb-2 text-xs font-medium uppercase tracking-[0.18em] text-blue-400/80">GOST PANEL</p>
        <h1 class="text-2xl font-semibold tracking-tight text-white">数据面板</h1>
        <p class="mt-1.5 text-sm text-slate-500">实时查看转发服务、协议分布与最近操作</p>
      </div>
      <div class="flex items-center gap-2.5">
        <span class="text-xs text-slate-500">更新于 {{ updatedAt || '等待数据' }}</span>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2 text-xs font-medium text-slate-300 transition-colors hover:border-white/[0.16] hover:bg-white/[0.07] hover:text-white disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="loading"
          title="刷新数据"
          @click="refresh"
        >
          <svg class="h-3.5 w-3.5" :class="loading && 'animate-spin'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <path d="M20 11a8.1 8.1 0 0 0-14.9-4L3 9m0 0V4m0 5h5M4 13a8.1 8.1 0 0 0 14.9 4L21 15m0 0v5m0-5h-5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          {{ loading ? '刷新中' : '刷新' }}
        </button>
        <router-link to="/rules" class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-3 py-2 text-xs font-semibold text-white transition-colors hover:bg-blue-500">
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <path d="M12 5v14m-7-7h14" stroke-linecap="round" />
          </svg>
          新建规则
        </router-link>
      </div>
    </header>

    <p v-if="error" role="alert" class="rounded-lg border border-red-400/20 bg-red-500/[0.08] px-4 py-3 text-sm text-red-300">{{ error }}</p>

    <section class="grid gap-3.5 sm:grid-cols-2 xl:grid-cols-4" aria-label="核心指标">
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
          <strong class="text-2xl font-semibold tracking-tight text-white">{{ metric.value }}</strong>
          <span class="text-xs" :class="metric.detailClass">{{ metric.detail }}</span>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[minmax(0,1.3fr)_minmax(320px,0.7fr)]">
      <div class="rounded-xl border border-white/[0.07] bg-[#161b25] p-5 sm:p-6">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="text-sm font-semibold text-white">协议分布</h2>
            <p class="mt-1 text-xs text-slate-500">按当前规则类型统计</p>
          </div>
          <span class="rounded-md bg-white/[0.04] px-2 py-1 text-xs text-slate-500">{{ total }} 条规则</span>
        </div>

        <div v-if="protoRows.length" class="mt-6 space-y-4">
          <div v-for="row in protoRows" :key="row.label">
            <div class="mb-2 flex items-center justify-between gap-3 text-xs">
              <div class="flex min-w-0 items-center gap-2.5">
                <span class="h-2 w-2 shrink-0 rounded-full" :class="row.dotClass"></span>
                <span class="font-medium text-slate-300">{{ row.label }}</span>
                <span class="truncate text-slate-600">{{ row.description }}</span>
              </div>
              <span class="shrink-0 font-semibold text-slate-300">{{ row.count }} <span class="font-normal text-slate-600">({{ row.percent }}%)</span></span>
            </div>
            <div class="h-2 overflow-hidden rounded-full bg-white/[0.06]">
              <div class="h-full rounded-full transition-[width] duration-500" :class="row.barClass" :style="{ width: `${row.percent}%` }"></div>
            </div>
          </div>
        </div>
        <div v-else class="mt-10 rounded-lg border border-dashed border-white/[0.1] py-10 text-center text-sm text-slate-500">暂无规则数据</div>
      </div>

      <div class="rounded-xl border border-white/[0.07] bg-[#161b25] p-5 sm:p-6">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="text-sm font-semibold text-white">服务健康度</h2>
            <p class="mt-1 text-xs text-slate-500">面板与 GOST 当前连接状态</p>
          </div>
          <span class="h-2.5 w-2.5 rounded-full" :class="healthDotClass" :title="healthLabel"></span>
        </div>

        <div class="mt-6 space-y-3">
          <div v-for="item in healthItems" :key="item.label" class="flex items-center justify-between rounded-lg border border-white/[0.06] bg-white/[0.02] px-3.5 py-3">
            <div class="flex items-center gap-3">
              <span class="h-2 w-2 rounded-full" :class="item.ok ? 'bg-emerald-400' : 'bg-red-400'"></span>
              <span class="text-sm text-slate-300">{{ item.label }}</span>
            </div>
            <span class="text-xs font-medium" :class="item.ok ? 'text-emerald-300' : 'text-red-300'">{{ item.value }}</span>
          </div>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[minmax(0,1.25fr)_minmax(320px,0.75fr)]">
      <div class="rounded-xl border border-white/[0.07] bg-[#161b25] p-5 sm:p-6">
        <div class="flex items-center justify-between gap-4">
          <div>
            <h2 class="text-sm font-semibold text-white">最近规则</h2>
            <p class="mt-1 text-xs text-slate-500">按创建时间倒序显示</p>
          </div>
          <router-link to="/rules" class="text-xs font-medium text-blue-400 transition-colors hover:text-blue-300">查看全部</router-link>
        </div>
        <!-- 移动端卡片列表 -->
        <div v-if="latestRules.length" class="mt-4 space-y-2 md:hidden">
          <div v-for="rule in latestRules" :key="rule.id" class="rounded-lg border border-white/[0.06] bg-white/[0.02] px-3.5 py-3">
            <div class="flex items-center justify-between gap-2">
              <span class="truncate text-xs font-medium text-slate-200">{{ rule.name }}</span>
              <span class="shrink-0 rounded bg-blue-500/10 px-1.5 py-0.5 text-[10px] text-blue-300">{{ ruleType(rule) }}</span>
            </div>
            <div class="mt-2 flex items-center justify-between gap-2 font-mono text-[11px] text-slate-500">
              <span>:{{ rule.listen_port }} → {{ rule.target_host }}:{{ rule.target_port }}</span>
              <span class="inline-flex items-center gap-1" :class="rule.enabled ? 'text-emerald-300' : 'text-slate-500'">
                <span class="h-1 w-1 rounded-full" :class="rule.enabled ? 'bg-emerald-400' : 'bg-slate-600'"></span>
                {{ rule.enabled ? '启用' : '停用' }}
              </span>
            </div>
          </div>
        </div>

        <!-- 桌面端表格 -->
        <div class="mt-4 hidden overflow-x-auto md:block">
          <table class="w-full min-w-[560px] text-left text-xs">
            <thead class="text-slate-600">
              <tr>
                <th class="pb-3 pr-4 font-medium">名称</th>
                <th class="pb-3 pr-4 font-medium">类型</th>
                <th class="pb-3 pr-4 font-medium">监听</th>
                <th class="pb-3 pr-4 font-medium">目标</th>
                <th class="pb-3 font-medium">状态</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-white/[0.06]">
              <tr v-for="rule in latestRules" :key="rule.id" class="text-slate-300">
                <td class="max-w-[180px] truncate py-3 pr-4 font-medium text-slate-200">{{ rule.name }}</td>
                <td class="py-3 pr-4"><span class="rounded bg-blue-500/10 px-2 py-1 text-[11px] text-blue-300">{{ ruleType(rule) }}</span></td>
                <td class="py-3 pr-4 font-mono text-slate-400">:{{ rule.listen_port }}</td>
                <td class="max-w-[180px] truncate py-3 pr-4 font-mono text-slate-500">{{ rule.target_host }}:{{ rule.target_port }}</td>
                <td class="py-3"><span class="inline-flex items-center gap-1.5" :class="rule.enabled ? 'text-emerald-300' : 'text-slate-500'"><span class="h-1.5 w-1.5 rounded-full" :class="rule.enabled ? 'bg-emerald-400' : 'bg-slate-600'"></span>{{ rule.enabled ? '启用' : '停用' }}</span></td>
              </tr>
              <tr v-if="!latestRules.length">
                <td colspan="5" class="py-10 text-center text-slate-500">暂无规则数据</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="!latestRules.length" class="py-10 text-center text-sm text-slate-500">暂无规则数据</p>
      </div>

      <div class="rounded-xl border border-white/[0.07] bg-[#161b25] p-5 sm:p-6">
        <div class="flex items-center justify-between gap-4">
          <div>
            <h2 class="text-sm font-semibold text-white">最近操作</h2>
            <p class="mt-1 text-xs text-slate-500">最新审计活动</p>
          </div>
          <router-link to="/audit" class="text-xs font-medium text-blue-400 transition-colors hover:text-blue-300">审计日志</router-link>
        </div>
        <div class="mt-4 space-y-1">
          <div v-for="item in recentAudit" :key="item.id" class="flex gap-3 rounded-lg px-2 py-3 transition-colors hover:bg-white/[0.03]">
            <span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-blue-400"></span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span class="text-xs font-medium text-slate-300">{{ actionLabel(item.action) }}</span>
                <time class="font-mono text-[10px] text-slate-600">{{ formatTime(item.ts) }}</time>
              </div>
              <p class="mt-1 truncate text-xs text-slate-500">{{ item.detail || '无附加信息' }}</p>
            </div>
          </div>
          <p v-if="!recentAudit.length" class="py-10 text-center text-sm text-slate-500">暂无操作记录</p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { statusApi } from '@/api/status'
import { rulesApi } from '@/api/rules'
import { auditApi } from '@/api/audit'
import { handleError } from '@/api'

const st = ref({})
const rules = ref([])
const recentAudit = ref([])
const loading = ref(false)
const error = ref('')
const updatedAt = ref('')

const total = computed(() => rules.value.length)
const enabledCount = computed(() => rules.value.filter((rule) => Number(rule.enabled) === 1).length)
const disabledCount = computed(() => total.value - enabledCount.value)
const forwardCount = computed(() => rules.value.filter((rule) => rule.type !== 'tunnel').length)
const tunnelCount = computed(() => rules.value.filter((rule) => rule.type === 'tunnel').length)
const enabledRate = computed(() => total.value ? `${Math.round((enabledCount.value / total.value) * 100)}%` : '0%')

const metrics = computed(() => [
  {
    label: '规则总数', value: total.value, detail: `${enabledCount.value} 条启用`, detailClass: 'text-emerald-300',
    icon: 'M4 6h16M4 12h16M4 18h16', iconBg: 'bg-blue-500/10', iconColor: 'text-blue-300',
  },
  {
    label: '启用率', value: enabledRate.value, detail: `${disabledCount.value} 条停用`, detailClass: 'text-slate-500',
    icon: 'M5 12h4l3-8 4 16 3-8h4', iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-300',
  },
  {
    label: '端口转发', value: forwardCount.value, detail: 'TCP / UDP', detailClass: 'text-slate-500',
    icon: 'M8 7h8m-8 5h8m-8 5h5M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z', iconBg: 'bg-violet-500/10', iconColor: 'text-violet-300',
  },
  {
    label: '传输隧道', value: tunnelCount.value, detail: 'WS / gRPC / QUIC', detailClass: 'text-slate-500',
    icon: 'M13 10V3L4 14h7v7l9-11h-7Z', iconBg: 'bg-amber-500/10', iconColor: 'text-amber-300',
  },
])

const protoRows = computed(() => {
  const definitions = [
    ['TCP', '端口转发', (rule) => rule.type !== 'tunnel' && rule.proto === 'tcp', 'bg-blue-400', 'bg-blue-400'],
    ['UDP', '端口转发', (rule) => rule.type !== 'tunnel' && rule.proto === 'udp', 'bg-emerald-400', 'bg-emerald-400'],
    ['WS', '传输隧道', (rule) => rule.type === 'tunnel' && rule.tunnel_proto === 'ws', 'bg-violet-400', 'bg-violet-400'],
    ['WSS', '传输隧道', (rule) => rule.type === 'tunnel' && rule.tunnel_proto === 'wss', 'bg-fuchsia-400', 'bg-fuchsia-400'],
    ['gRPC', '传输隧道', (rule) => rule.type === 'tunnel' && rule.tunnel_proto === 'grpc', 'bg-amber-400', 'bg-amber-400'],
    ['QUIC / HTTP2', '传输隧道', (rule) => rule.type === 'tunnel' && ['quic', 'http2'].includes(rule.tunnel_proto), 'bg-cyan-400', 'bg-cyan-400'],
  ]
  return definitions
    .map(([label, description, match, dotClass, barClass]) => {
      const count = rules.value.filter(match).length
      return { label, description, count, dotClass, barClass, percent: total.value ? Math.round((count / total.value) * 100) : 0 }
    })
    .filter((row) => row.count > 0)
})

const healthItems = computed(() => [
  { label: 'gost 引擎', ok: Boolean(st.value.gost?.active), value: st.value.gost?.active ? '运行中' : '已停止' },
  { label: 'gost API', ok: st.value.gost?.api === 'up', value: st.value.gost?.api === 'up' ? '在线' : '离线' },
  { label: '已同步服务', ok: true, value: `${st.value.gost?.service_count ?? 0} 个` },
])

const healthDotClass = computed(() => st.value.gost?.active && st.value.gost?.api === 'up' ? 'bg-emerald-400' : 'bg-amber-400')
const healthLabel = computed(() => st.value.gost?.active && st.value.gost?.api === 'up' ? '服务正常' : '服务需要关注')
const latestRules = computed(() => rules.value.slice(0, 5))

async function refresh() {
  loading.value = true
  try {
    const [statusResponse, rulesResponse] = await Promise.all([statusApi.get(), rulesApi.list()])
    st.value = statusResponse.data || {}
    rules.value = Array.isArray(rulesResponse.data) ? rulesResponse.data : []
    updatedAt.value = formatTime(st.value.panel?.time) || formatTime(new Date().toISOString())
    error.value = ''
    try {
      const { data } = await auditApi.list({ limit: 5 })
      recentAudit.value = Array.isArray(data) ? data : []
    } catch {
      recentAudit.value = []
    }
  } catch (e) {
    error.value = handleError(e)
  } finally {
    loading.value = false
  }
}

function ruleType(rule) {
  return rule.type === 'tunnel' ? `隧道 ${String(rule.tunnel_proto || '').toUpperCase()}` : `转发 ${String(rule.proto || '').toUpperCase()}`
}

function actionLabel(action) {
  return { login: '登录面板', 'rule.create': '创建规则', 'rule.update': '更新规则', 'rule.delete': '删除规则', 'rule.apply': '同步规则', 'audit.clear': '清空日志' }[action] || action
}

function formatTime(value) {
  if (!value) return ''
  return String(value).replace('T', ' ').slice(0, 19)
}

let timer
onMounted(() => {
  refresh()
  timer = setInterval(refresh, 10000)
})
onUnmounted(() => clearInterval(timer))
</script>
