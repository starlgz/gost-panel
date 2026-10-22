<template>
  <div class="mx-auto max-w-[1440px] space-y-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="mb-2 text-xs font-medium uppercase tracking-[0.18em] text-blue-400/80">AUDIT LOG</p>
        <h1 class="text-2xl font-semibold tracking-tight text-white">审计日志</h1>
        <p class="mt-1.5 text-sm text-slate-500">记录面板登录与规则变更操作，共 {{ total }} 条</p>
      </div>
      <div class="flex items-center gap-2.5">
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2 text-xs font-medium text-slate-300 transition-colors hover:border-white/[0.16] hover:bg-white/[0.07] hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="loading || clearing"
          @click="load"
        >
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <path d="M20 11a8.1 8.1 0 0 0-14.9-4L3 9m0 0V4m0 5h5M4 13a8.1 8.1 0 0 0 14.9 4L21 15m0 0v5m0-5h-5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          刷新
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-red-400/20 bg-red-500/[0.06] px-3 py-2 text-xs font-medium text-red-300 transition-colors hover:border-red-400/40 hover:bg-red-500/[0.12] disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="loading || clearing || !total"
          title="清空全部审计记录"
          @click="openClearConfirm"
        >
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
            <path d="M4 7h16M10 11v6m4-6v6M6 7l1 13h10l1-13M9 7V4h6v3" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          {{ clearing ? '清空中...' : '清空日志' }}
        </button>
      </div>
    </header>

    <p v-if="error" role="alert" class="rounded-lg border border-red-400/20 bg-red-500/[0.08] px-4 py-3 text-sm text-red-300">{{ error }}</p>

    <section class="rounded-xl border border-white/[0.07] bg-[#161b25] p-5 sm:p-6" aria-label="审计日志列表">
      <div class="overflow-x-auto">
        <table class="w-full min-w-[620px] text-left text-xs">
          <thead class="text-slate-600">
            <tr>
              <th class="pb-3 pr-4 font-medium" style="width: 170px;">时间</th>
              <th class="pb-3 pr-4 font-medium" style="width: 130px;">操作</th>
              <th class="pb-3 font-medium">详情</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-white/[0.06]">
            <tr v-for="l in logs" :key="l.id" class="text-slate-300 transition-colors hover:bg-white/[0.02]">
              <td class="py-3.5 pr-4 font-mono text-slate-400">{{ formatTime(l.ts) }}</td>
              <td class="py-3.5 pr-4">
                <span class="rounded px-2 py-1 text-[11px]" :class="actionClass(l.action)">{{ actionLabel(l.action) }}</span>
              </td>
              <td class="py-3.5 text-slate-500">{{ l.detail || '-' }}</td>
            </tr>
            <tr v-if="!logs.length">
              <td colspan="3" class="py-10 text-center text-slate-500">{{ loading ? '加载中…' : '暂无日志' }}</td>
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
            :disabled="loading || clearing || page <= 1"
            @click="gotoPage(page - 1)"
          >
            上一页
          </button>
          <button
            v-for="p in pageList"
            :key="p"
            type="button"
            :disabled="loading || clearing"
            class="h-7 w-7 rounded-md text-xs transition-colors"
            :class="p === page ? 'bg-blue-600 font-semibold text-white' : 'border border-white/[0.08] text-slate-300 hover:bg-white/[0.06]'"
            @click="gotoPage(p)"
          >
            {{ p }}
          </button>
          <button
            type="button"
            class="rounded-md border border-white/[0.08] px-2.5 py-1.5 text-xs text-slate-300 transition-colors hover:bg-white/[0.06] disabled:cursor-not-allowed disabled:opacity-40"
            :disabled="loading || clearing || page >= totalPages"
            @click="gotoPage(page + 1)"
          >
            下一页
          </button>
        </div>
      </div>
    </section>
    <ConfirmDialog
      v-model="showClearConfirm"
      title="清空审计日志"
      :description="`确认清空全部 ${total} 条审计记录？此操作无法恢复。`"
      confirm-text="确认清空"
      danger
      :busy="clearing"
      :error="clearError"
      @confirm="clearAll"
    >
      <p class="mt-2 text-xs text-slate-500">清空后仅保留本次清空操作的记录，转发规则不受影响。</p>
    </ConfirmDialog>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { auditApi } from '@/api/audit'
import { handleError } from '@/api'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

const logs = ref([])
const error = ref('')
const loading = ref(false)
const clearing = ref(false)
const showClearConfirm = ref(false)
const clearError = ref('')
let loadVersion = 0
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)
const PAGE_SIZE = 20

function gotoPage(p) {
  if (loading.value || clearing.value || p < 1 || p > totalPages.value || p === page.value) return
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

const LABELS = {
  login: '登录面板',
  'rule.create': '创建规则',
  'rule.update': '更新规则',
  'rule.delete': '删除规则',
  'rule.apply': '同步规则',
  'audit.clear': '清空日志',
}

// 登录蓝、删除红、其余绿
function actionClass(action) {
  if (String(action).includes('login')) return 'bg-blue-500/10 text-blue-300'
  if (String(action).includes('delete') || String(action).includes('del')) return 'bg-red-500/[0.08] text-red-300'
  return 'bg-emerald-500/10 text-emerald-300'
}

function actionLabel(action) {
  return LABELS[action] || action
}

function formatTime(value) {
  if (!value) return '-'
  return String(value).replace('T', ' ').slice(0, 19)
}

async function load() {
  const version = ++loadVersion
  loading.value = true
  try {
    const { data } = await auditApi.list({
      page: page.value,
      page_size: PAGE_SIZE,
    })
    if (version !== loadVersion) return
    logs.value = data.items || []
    total.value = data.total || 0
    page.value = data.page
    totalPages.value = Math.max(1, Math.ceil(total.value / data.page_size))
    if (page.value > totalPages.value) {
      page.value = totalPages.value
      return await load()
    }
    error.value = ''
  } catch (e) {
    if (version === loadVersion) error.value = handleError(e)
  } finally {
    if (version === loadVersion) loading.value = false
  }
}

function openClearConfirm() {
  clearError.value = ''
  showClearConfirm.value = true
}

async function clearAll() {
  if (clearing.value) return
  clearing.value = true
  clearError.value = ''
  try {
    await auditApi.clear()
    showClearConfirm.value = false
    logs.value = []
    page.value = 1
    await load()
  } catch (e) {
    clearError.value = handleError(e)
  } finally {
    clearing.value = false
  }
}

onMounted(load)
onBeforeUnmount(() => { loadVersion++ })
</script>
