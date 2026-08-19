<template>
  <div>
    <h1>运行状态</h1>
    <div class="cards">
      <div class="card">
        <div class="label">gost 引擎</div>
        <div class="value" :class="st.gost?.active ? 'ok' : 'bad'">
          {{ st.gost?.active ? '● 运行中' : '○ 已停止' }}
        </div>
      </div>
      <div class="card">
        <div class="label">gost API</div>
        <div class="value" :class="st.gost?.api === 'up' ? 'ok' : 'bad'">{{ st.gost?.api }}</div>
      </div>
      <div class="card">
        <div class="label">活动规则</div>
        <div class="value">{{ st.gost?.service_count }}</div>
      </div>
      <div class="card">
        <div class="label">面板版本</div>
        <div class="value">{{ st.panel?.version }}</div>
      </div>
    </div>
    <p class="time">数据时间：{{ st.panel?.time }}</p>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const st = ref({})

onMounted(async () => {
  const r = await fetch('/api/status', {
    headers: { 'Authorization': 'Bearer ' + localStorage.getItem('token') },
  })
  if (r.ok) st.value = await r.json()
})
</script>

<style scoped>
.cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 14px; }
.card { background: #1c2029; padding: 18px; border-radius: 10px; }
.label { color: #8a94a3; font-size: 13px; margin-bottom: 8px; }
.value { font-size: 22px; font-weight: 700; }
.ok { color: #3fb950; } .bad { color: #f85149; }
.time { color: #8a94a3; font-size: 12px; margin-top: 16px; }
</style>
