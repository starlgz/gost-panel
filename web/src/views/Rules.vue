<template>
  <div class="mx-auto max-w-[1440px] space-y-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="mb-2 text-xs font-medium uppercase tracking-[0.18em] text-blue-400/80">FORWARD RULES</p>
        <h1 class="text-2xl font-semibold tracking-tight text-white">转发规则</h1>
        <p class="mt-1.5 text-sm text-slate-500">管理端口转发与传输隧道，保存后自动同步到 gost</p>
      </div>
      <div class="flex items-center gap-2.5">
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2 text-xs font-medium text-slate-300 transition-colors hover:border-white/[0.16] hover:bg-white/[0.07] hover:text-white disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="applying"
           title="把当前数据库配置同步到gost"
          @click="applyRules"
        >
          <svg class="h-3.5 w-3.5" :class="applying && 'animate-spin'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <path d="M20 11a8.1 8.1 0 0 0-14.9-4L3 9m0 0V4m0 5h5M4 13a8.1 8.1 0 0 0 14.9 4L21 15m0 0v5m0-5h-5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          {{ applying ? '同步中' : '同步到 gost' }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2 text-xs font-medium text-slate-300 transition-colors hover:border-white/[0.16] hover:bg-white/[0.07] hover:text-white disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="pulling"
          title="从 gost 拉取当前运行的服务，和数据库规则比对"
          @click="pullGostServices"
        >
          <svg class="h-3.5 w-3.5" :class="pulling && 'animate-spin'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5-5 5 5M12 5v12" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          {{ pulling ? '拉取中' : '拉取比对' }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-3 py-2 text-xs font-semibold text-white transition-colors hover:bg-blue-500"
          @click="openForm()"
        >
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <path d="M12 5v14m-7-7h14" stroke-linecap="round" />
          </svg>
          新建规则
        </button>
      </div>
    </header>

    <p v-if="msg" role="alert" class="rounded-lg border px-4 py-3 text-sm" :class="msgErr ? 'border-red-400/20 bg-red-500/[0.08] text-red-300' : 'border-blue-400/20 bg-blue-500/[0.08] text-blue-300'">{{ msg }}</p>

    <!-- 统计 -->
    <section class="grid gap-3.5 sm:grid-cols-2 xl:grid-cols-4" aria-label="规则统计">
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

    <!-- 规则列表 -->
    <section class="rounded-xl border border-white/[0.07] bg-[#161b25] p-5 sm:p-6" aria-label="规则列表">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-white">全部规则</h2>
          <p class="mt-1 text-xs text-slate-500">
            {{ search ? `匹配 ${total} / ${totalAll} 条` : `共 ${totalAll} 条` }}
          </p>
        </div>
        <!-- 搜索框 -->
        <div class="relative w-full sm:w-64">
          <svg class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <circle cx="11" cy="11" r="7" />
            <path d="m21 21-4.3-4.3" stroke-linecap="round" />
          </svg>
          <input
            v-model="search"
            type="text"
            placeholder="搜索名称、端口、目标、备注…"
            class="h-9 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] pl-9 pr-8 text-xs text-white outline-none transition-colors placeholder:text-slate-600 hover:border-white/[0.16] focus:border-blue-500/70 focus:ring-[3px] focus:ring-blue-500/10"
          />
          <button
            v-if="search"
            type="button"
            aria-label="清空搜索"
            class="absolute top-1/2 right-2.5 -translate-y-1/2 text-slate-500 transition-colors hover:text-slate-300"
            @click="search = ''"
          >
            <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path d="M18 6 6 18M6 6l12 12" stroke-linecap="round" />
            </svg>
          </button>
        </div>
      </div>
      <div class="mt-4 overflow-x-auto">
        <table class="w-full min-w-[720px] text-left text-xs">
          <thead class="text-slate-600">
            <tr>
              <th class="pb-3 pr-4 font-medium">名称</th>
              <th class="pb-3 pr-4 font-medium">类型</th>
              <th class="pb-3 pr-4 font-medium">监听</th>
              <th class="pb-3 pr-4 font-medium">目标</th>
              <th class="pb-3 pr-4 font-medium">状态</th>
              <th class="pb-3 pr-4 font-medium">备注</th>
              <th class="pb-3 text-right font-medium">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-white/[0.06]">
            <tr v-for="r in rules" :key="r.id" class="text-slate-300 transition-colors hover:bg-white/[0.02]">
              <td class="max-w-[180px] truncate py-3.5 pr-4 font-medium text-slate-200">{{ r.name }}</td>
              <td class="py-3.5 pr-4"><span class="rounded bg-blue-500/10 px-2 py-1 text-[11px] text-blue-300">{{ ruleType(r) }}</span></td>
              <td class="py-3.5 pr-4 font-mono text-slate-400">:{{ r.listen_port }}</td>
              <td class="max-w-[200px] truncate py-3.5 pr-4 font-mono text-slate-500">{{ r.target_host }}:{{ r.target_port }}</td>
              <td class="py-3.5 pr-4">
                <span class="inline-flex items-center gap-1.5" :class="r.enabled ? 'text-emerald-300' : 'text-slate-500'">
                  <span class="h-1.5 w-1.5 rounded-full" :class="r.enabled ? 'bg-emerald-400' : 'bg-slate-600'"></span>
                  {{ r.enabled ? '启用' : '停用' }}
                </span>
              </td>
              <td class="max-w-[180px] truncate py-3.5 pr-4 text-slate-500">{{ r.remark || '-' }}</td>
              <td class="py-3.5">
                <div class="flex justify-end gap-1.5">
                  <button v-if="r.type === 'tunnel'" class="rounded-md border border-white/[0.08] px-2 py-1 text-[11px] text-slate-300 transition-colors hover:bg-white/[0.06]" @click="cmdRule = r">命令</button>
                  <button class="rounded-md border border-white/[0.08] px-2 py-1 text-[11px] text-slate-300 transition-colors hover:bg-white/[0.06]" @click="toggleRule(r)">{{ r.enabled ? '停用' : '启用' }}</button>
                  <button class="rounded-md border border-white/[0.08] px-2 py-1 text-[11px] text-slate-300 transition-colors hover:bg-white/[0.06]" @click="openForm(r)">编辑</button>
                  <button class="rounded-md border border-red-400/20 px-2 py-1 text-[11px] text-red-300 transition-colors hover:bg-red-500/[0.08]" @click="requestDelete(r)">删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="!rules.length">
              <td colspan="7" class="py-10 text-center text-slate-500">
                {{ loading ? '加载中…' : search ? '没有匹配的规则，换个关键词试试' : '暂无规则，点击右上角「新建规则」添加' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div v-if="totalPages > 1" class="mt-4 flex items-center justify-between border-t border-white/[0.07] pt-4">
        <span class="text-xs text-slate-500">第 {{ page }} / {{ totalPages }} 页，共 {{ total }} 条</span>
        <div class="flex items-center gap-1.5">
          <button
            type="button"
            class="rounded-md border border-white/[0.08] px-2.5 py-1.5 text-xs text-slate-300 transition-colors hover:bg-white/[0.06] disabled:cursor-not-allowed disabled:opacity-40"
            :disabled="loading || page <= 1"
            @click="gotoPage(page - 1)"
          >
            上一页
          </button>
          <button
            v-for="p in pageList"
            :key="p"
            type="button"
            :disabled="loading"
            class="h-7 w-7 rounded-md text-xs transition-colors"
            :class="p === page ? 'bg-blue-600 font-semibold text-white' : 'border border-white/[0.08] text-slate-300 hover:bg-white/[0.06]'"
            @click="gotoPage(p)"
          >
            {{ p }}
          </button>
          <button
            type="button"
            class="rounded-md border border-white/[0.08] px-2.5 py-1.5 text-xs text-slate-300 transition-colors hover:bg-white/[0.06] disabled:cursor-not-allowed disabled:opacity-40"
            :disabled="loading || page >= totalPages"
            @click="gotoPage(page + 1)"
          >
            下一页
          </button>
        </div>
      </div>
    </section>

    <!-- 新建/编辑弹窗 -->
    <Modal v-model="showForm" :title="form.id ? '编辑规则' : '新建规则'" :busy="saving">
      <div class="flex flex-col gap-3.5">
        <label class="label">名称 <input v-model="form.name" class="field" placeholder="例如：网站转发" /></label>
        <label class="label">规则类型
          <Select v-model="form.type" :options="typeOptions" />
        </label>
        <template v-if="form.type === 'forward'">
          <label class="label">协议
            <Select v-model="form.proto" :options="protoOptions" />
          </label>
        </template>
        <template v-else>
          <label class="label">隧道协议
            <Select v-model="form.tunnel_proto" :options="tunnelOptions" />
          </label>
          <label v-if="form.tunnel_proto === 'ws' || form.tunnel_proto === 'wss'" class="label">
            路径（可选）<input v-model="form.tunnel_path" class="field" placeholder="/ws" />
          </label>
          <label class="label">认证用户名（可选，留空则无需认证）
            <input v-model="form.tunnel_user" class="field" placeholder="user" />
          </label>
          <label v-if="form.tunnel_user" class="label">
            认证密码<input v-model="form.tunnel_pass" class="field" type="password" placeholder="password" />
          </label>
        </template>
        <div class="grid grid-cols-2 gap-3">
          <label class="label">监听端口
            <input v-model.number="form.listen_port" class="field" type="number" min="1" max="65535" />
          </label>
          <label class="label">目标端口
            <input v-model.number="form.target_port" class="field" type="number" min="1" max="65535" />
          </label>
        </div>
        <label class="label">目标主机 <input v-model="form.target_host" class="field" placeholder="1.2.3.4 或域名" /></label>
        <label class="label">备注 <input v-model="form.remark" class="field" placeholder="可选" /></label>
      </div>
      <template #footer>
        <button type="button" class="btn" @click="showForm = false">取消</button>
        <button type="button" class="btn btn-primary" :disabled="saving" @click="save">{{ saving ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>

    <!-- 客户端命令弹窗 -->
    <Modal :model-value="!!cmdRule" title="客户端连接命令" @update:model-value="cmdRule = null">
      <p class="text-[13px] text-slate-500">
        在你自己的电脑上运行（需安装 gost），把 SERVER_IP 换成服务器 IP，1080 是本地代理端口可改：
      </p>
      <pre class="mt-3 rounded-lg bg-[#0e131c] p-3.5 font-mono text-[13px] whitespace-pre-wrap break-all text-[#6aa1ff]">{{ cmdRule?.client_cmd }}</pre>
      <template #footer>
        <button type="button" class="btn" @click="copyCmd">复制</button>
        <button type="button" class="btn btn-primary" @click="cmdRule = null">关闭</button>
      </template>
    </Modal>

    <!-- gost 服务比对弹窗 -->
    <Modal v-model="showPullResult" title="gost 运行状态比对">
      <div class="space-y-4">
        <p class="text-xs text-slate-500">
          以下是 gost 当前实际运行的服务。面板规则保存后会自动同步，如果发现不一致，点击「同步到 gost」即可修复。
        </p>
        <div v-if="pullResult.length" class="space-y-1.5">
          <div v-for="svc in pullResult" :key="svc.name" class="flex items-center justify-between rounded-lg border border-white/[0.06] bg-white/[0.02] px-3.5 py-2.5">
            <div class="flex items-center gap-2.5">
              <span class="h-1.5 w-1.5 rounded-full bg-emerald-400"></span>
              <span class="font-mono text-xs text-slate-300">{{ svc.name }}</span>
              <span class="font-mono text-xs text-slate-500">{{ svc.addr }}</span>
            </div>
            <span class="rounded bg-emerald-500/10 px-2 py-0.5 text-[11px] text-emerald-300">运行中</span>
          </div>
        </div>
        <p v-else class="py-6 text-center text-xs text-slate-500">gost 当前没有运行任何服务</p>
      </div>
      <template #footer>
        <button type="button" class="btn" @click="showPullResult = false">关闭</button>
        <button type="button" class="btn" :disabled="importing" @click="importGostServices">
          {{ importing ? '导入中...' : '导入到面板' }}
        </button>
        <button type="button" class="btn btn-primary" @click="showPullResult = false; applyRules()">同步到 gost</button>
      </template>
    </Modal>

    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="删除转发规则"
      :description="`确认删除规则「${deleteTarget?.name || ''}」？此操作无法恢复。`"
      confirm-text="确认删除"
      danger
      :busy="deleting"
      :error="deleteError"
      @confirm="delRule"
    />
  </div>
</template>

<script setup>
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { rulesApi } from '@/api/rules'
import { gostApi } from '@/api/gost'
import { handleError } from '@/api'
import Select from '@/components/Select.vue'
import Modal from '@/components/Modal.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

const rules = ref([])
const showForm = ref(false)
const saving = ref(false)
const applying = ref(false)
const msg = ref('')
const msgErr = ref(false)
const cmdRule = ref(null)
const form = ref(emptyForm())
const loading = ref(false)
const showDeleteConfirm = ref(false)
const deleteTarget = ref(null)
const deleting = ref(false)
const deleteError = ref('')
const pulling = ref(false)
const importing = ref(false)
const showPullResult = ref(false)
const pullResult = ref([])

// 自定义下拉的选项
const typeOptions = [
  { value: 'forward', label: '端口转发', desc: 'TCP / UDP 直转' },
  { value: 'tunnel', label: '传输隧道', desc: 'WS / WSS / gRPC / QUIC' },
]
const protoOptions = [
  { value: 'tcp', label: 'TCP', desc: '可靠传输，适用大多数场景' },
  { value: 'udp', label: 'UDP', desc: '低延迟，适用游戏、语音' },
]
const tunnelOptions = [
  { value: 'ws', label: 'WS', desc: 'WebSocket，可套 CDN' },
  { value: 'wss', label: 'WSS', desc: 'WebSocket + TLS 加密' },
  { value: 'grpc', label: 'gRPC', desc: '多路复用，抗封锁' },
  { value: 'quic', label: 'QUIC', desc: '基于 UDP 的加密传输' },
  { value: 'http2', label: 'HTTP/2', desc: 'h2 长连接传输' },
]

const metrics = computed(() => {
  // 全局统计由后端聚合，切页和搜索不改变统计卡片。
  const s = stats.value
  const enabled = s.enabled
  const tcp = s.tcp
  const udp = s.udp
  return [
    {
      label: '规则总数', value: totalAll.value, detail: `${enabled} 条启用`, detailClass: 'text-emerald-300',
      icon: 'M4 6h16M4 12h16M4 18h16', iconBg: 'bg-blue-500/10', iconColor: 'text-blue-300',
    },
    {
      label: 'TCP 转发', value: tcp, detail: '端口转发', detailClass: 'text-slate-500',
      icon: 'M8 7h8m-8 5h8m-8 5h5M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z', iconBg: 'bg-violet-500/10', iconColor: 'text-violet-300',
    },
    {
      label: 'UDP 转发', value: udp, detail: '端口转发', detailClass: 'text-slate-500',
      icon: 'M8 7h8m-8 5h8m-8 5h5M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z', iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-300',
    },
    {
      label: '传输隧道', value: s.tunnel, detail: 'WS / gRPC / QUIC', detailClass: 'text-slate-500',
      icon: 'M13 10V3L4 14h7v7l9-11h-7Z', iconBg: 'bg-amber-500/10', iconColor: 'text-amber-300',
    },
  ]
})

// 统计接口仅返回聚合值，避免列表页额外拉取全部规则。
const stats = ref({ enabled: 0, tcp: 0, udp: 0, tunnel: 0 })

async function loadStats() {
  try {
    const { data } = await rulesApi.stats()
    stats.value = data
    totalAll.value = data.total
  } catch (e) { flash(handleError(e), true) }
}

function ruleType(r) {
  return r.type === 'tunnel' ? `隧道 ${String(r.tunnel_proto || '').toUpperCase()}` : `转发 ${String(r.proto || '').toUpperCase()}`
}

// 搜索 + 后端分页
const search = ref('')
const page = ref(1)
const totalPages = ref(1)
const total = ref(0) // 当前过滤条件的总条数
const totalAll = ref(0) // 无过滤时的规则总数（统计卡用）
const PAGE_SIZE = 8
let searchTimer = null
let loadVersion = 0

// 防抖 300ms，输入停顿后才请求
watch(search, () => {
  clearTimeout(searchTimer)
  // 输入一变就使旧请求失效，覆盖防抖等待期间的请求竞态。
  loadVersion++
  loading.value = true
  rules.value = []
  page.value = 1
  searchTimer = setTimeout(() => {
    load()
  }, 300)
}, { flush: 'sync' })

function gotoPage(p) {
  if (loading.value || p < 1 || p > totalPages.value || p === page.value) return
  page.value = p
  load()
}

// 最多显示 5 个页码，当前页居中
const pageList = computed(() => {
  const span = 2
  let start = Math.max(1, page.value - span)
  const end = Math.min(totalPages.value, start + span * 2)
  start = Math.max(1, end - span * 2)
  const list = []
  for (let p = start; p <= end; p++) list.push(p)
  return list
})

async function load() {
  clearTimeout(searchTimer)
  const version = ++loadVersion
  loading.value = true
  try {
    const { data } = await rulesApi.list({ page: page.value, page_size: PAGE_SIZE, keyword: search.value.trim() || undefined })
    if (version !== loadVersion) return
    rules.value = data.items || []
    total.value = data.total || 0
    page.value = data.page
    totalPages.value = Math.max(1, Math.ceil(total.value / data.page_size))
    if (page.value > totalPages.value) {
      page.value = totalPages.value
      return await load()
    }
  } catch (e) {
    if (version === loadVersion) flash(handleError(e), true)
  } finally {
    if (version === loadVersion) loading.value = false
  }
}

function emptyForm() {
  return { id: null, name: '', type: 'forward', proto: 'tcp', listen_port: null, target_host: '', target_port: null,
           tunnel_proto: 'ws', tunnel_path: '', tunnel_user: '', tunnel_pass: '', enabled: true, remark: '' }
}

function openForm(r) {
  form.value = r ? { ...r, enabled: !!r.enabled } : emptyForm()
  showForm.value = true
}

async function copyCmd() {
  try {
    await navigator.clipboard.writeText(cmdRule.value.client_cmd)
    flash('命令已复制')
  } catch {
    flash('复制失败，请手动选择复制', true)
  }
}

async function save() {
  saving.value = true
  try {
    const body = { ...form.value, enabled: true }
    if (form.value.id) {
      await rulesApi.update(form.value.id, body)
    } else {
      await rulesApi.create(body)
    }
    showForm.value = false
    await Promise.all([load(), loadStats()])
    flash('已保存并同步到 gost')
  } catch (e) { flash(handleError(e), true) } finally { saving.value = false }
}

function requestDelete(rule) {
  deleteTarget.value = rule
  deleteError.value = ''
  showDeleteConfirm.value = true
}

async function delRule() {
  if (deleting.value || !deleteTarget.value) return
  deleting.value = true
  deleteError.value = ''
  try {
    await rulesApi.remove(deleteTarget.value.id)
    showDeleteConfirm.value = false
    await Promise.all([load(), loadStats()])
    flash('已删除')
  } catch (e) {
    deleteError.value = handleError(e)
  } finally {
    deleting.value = false
  }
}

async function toggleRule(r) {
  try {
    await rulesApi.toggleEnabled(r.id, !r.enabled)
    await Promise.all([load(), loadStats()])
  } catch (e) { flash(handleError(e), true) }
}

async function applyRules() {
  applying.value = true
  try {
    const { data } = await rulesApi.apply()
    flash(data.message || '同步完成')
  } catch (e) { flash(handleError(e), true) } finally { applying.value = false }
}

async function pullGostServices() {
  pulling.value = true
  try {
    const { data } = await gostApi.services()
    pullResult.value = Array.isArray(data) ? data : []
    showPullResult.value = true
  } catch (e) { flash(handleError(e), true) } finally { pulling.value = false }
}

async function importGostServices() {
  importing.value = true
  try {
    const { data } = await gostApi.importServices()
    showPullResult.value = false
    flash(data.message || '导入完成')
    await Promise.all([load(), loadStats()])
  } catch (e) { flash(handleError(e), true) } finally { importing.value = false }
}

function flash(text, err = false) {
  msg.value = text; msgErr.value = err
  setTimeout(() => { msg.value = ''; msgErr.value = false }, 4000)
}

onMounted(() => {
  load()
  loadStats()
})
onBeforeUnmount(() => {
  clearTimeout(searchTimer)
  loadVersion++
})
</script>
