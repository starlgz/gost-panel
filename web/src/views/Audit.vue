<template>
  <div>
    <h1>审计日志</h1>
    <table>
      <thead>
        <tr><th>时间</th><th>操作</th><th>详情</th></tr>
      </thead>
      <tbody>
        <tr v-if="!logs.length"><td colspan="3" class="empty">暂无日志</td></tr>
        <tr v-for="l in logs" :key="l.id">
          <td class="mono">{{ l.ts }}</td>
          <td><span class="tag">{{ l.action }}</span></td>
          <td>{{ l.detail }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const logs = ref([])

onMounted(async () => {
  const r = await fetch('/api/audit?limit=50', {
    headers: { 'Authorization': 'Bearer ' + localStorage.getItem('token') },
  })
  if (r.ok) logs.value = await r.json()
})
</script>

<style scoped>
table { width: 100%; border-collapse: collapse; font-size: 14px; }
th, td { text-align: left; padding: 9px 12px; border-bottom: 1px solid #232833; }
th { color: #8a94a3; font-weight: 500; font-size: 13px; }
.mono { font-family: ui-monospace, monospace; font-size: 12px; color: #8a94a3; }
.tag { padding: 2px 8px; border-radius: 5px; font-size: 12px; background: #1c3152; color: #6aa1ff; }
.empty { text-align: center; color: #8a94a3; padding: 30px 0; }
</style>
