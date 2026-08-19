<template>
  <div>
        <div class="header">
      <h1>转发规则</h1>
      <div class="actions">
        <button class="btn apply" :disabled="applying" @click="applyRules">
          {{ applying ? '同步中...' : '🔄 同步到 gost' }}
        </button>
        <button class="btn primary" @click="openForm()">＋ 新建规则</button>
      </div>
    </div>
    <p v-if="msg" class="msg" :class="{ err: msgErr }">{{ msg }}</p>

    <table>
      <thead>
        <tr>
          <th>ID</th><th>名称</th><th>类型</th><th>监听端口</th>
          <th>目标</th><th>状态</th><th>备注</th><th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!rules.length">
          <td colspan="8" class="empty">暂无规则，点击「新建规则」添加</td>
        </tr>
        <tr v-for="r in rules" :key="r.id">
          <td>{{ r.id }}</td>
          <td>{{ r.name }}</td>
          <td>
            <span class="tag" :class="r.type">{{ r.type === 'tunnel' ? '隧道 ' + r.tunnel_proto.toUpperCase() : '转发 ' + r.proto.toUpperCase() }}</span>
          </td>
          <td>{{ r.listen_port }}</td>
          <td class="mono">{{ r.target_host }}:{{ r.target_port }}</td>
          <td>
            <span class="dot" :class="r.enabled ? 'on' : 'off'"></span>
            {{ r.enabled ? '启用' : '停用' }}
          </td>
          <td class="remark">{{ r.remark }}</td>
          <td class="ops">
            <button v-if="r.type === 'tunnel'" class="btn sm" @click="showCmd(r)">命令</button>
            <button class="btn sm" @click="toggleRule(r)">{{ r.enabled ? '停用' : '启用' }}</button>
            <button class="btn sm" @click="openForm(r)">编辑</button>
            <button class="btn sm danger" @click="delRule(r)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-if="showForm" class="modal">
      <div class="card">
        <h3>{{ form.id ? '编辑规则' : '新建规则' }}</h3>
        <label>名称 <input v-model="form.name" placeholder="例如：网站转发" /></label>
        <label>规则类型
          <select v-model="form.type">
            <option value="forward">端口转发</option>
            <option value="tunnel">传输隧道（WS/WSS/gRPC/QUIC）</option>
          </select>
        </label>
        <template v-if="form.type === 'forward'">
          <label>协议
            <select v-model="form.proto">
              <option value="tcp">TCP</option>
              <option value="udp">UDP</option>
            </select>
          </label>
        </template>
        <template v-else>
          <label>隧道协议
            <select v-model="form.tunnel_proto">
              <option value="ws">WS (WebSocket)</option>
              <option value="wss">WSS (WebSocket+TLS)</option>
              <option value="grpc">gRPC</option>
              <option value="quic">QUIC</option>
              <option value="http2">HTTP/2</option>
            </select>
          </label>
          <label v-if="form.tunnel_proto === 'ws' || form.tunnel_proto === 'wss'">路径（可选）
            <input v-model="form.tunnel_path" placeholder="/ws" />
          </label>
          <label>认证用户名（可选，留空则无需认证）
            <input v-model="form.tunnel_user" placeholder="user" />
          </label>
          <label v-if="form.tunnel_user">认证密码
            <input v-model="form.tunnel_pass" type="password" placeholder="password" />
          </label>
        </template>
        <label>监听端口 <input v-model.number="form.listen_port" type="number" min="1" max="65535" /></label>
        <label>目标主机 <input v-model="form.target_host" placeholder="1.2.3.4 或域名" /></label>
        <label>目标端口 <input v-model.number="form.target_port" type="number" min="1" max="65535" /></label>
        <label>备注 <input v-model="form.remark" placeholder="可选" /></label>
        <div class="modal-ops">
          <button class="btn" @click="showForm = false">取消</button>
          <button class="btn primary" :disabled="saving" @click="save">{{ saving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <div v-if="cmdRule" class="modal">
      <div class="card">
        <h3>客户端连接命令</h3>
        <p class="hint">在你自己的电脑上运行（需安装 gost），把 SERVER_IP 换成服务器 IP，1080 是本地代理端口可改：</p>
        <pre class="cmd">{{ cmdRule.client_cmd }}</pre>
        <div class="modal-ops">
          <button class="btn" @click="copyCmd">复制</button>
          <button class="btn primary" @click="cmdRule = null">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const rules = ref([])
const showForm = ref(false)
const saving = ref(false)
const applying = ref(false)
const msg = ref('')
const msgErr = ref(false)
const cmdRule = ref(null)
const form = ref(emptyForm())

function emptyForm() {
  return { id: null, name: '', type: 'forward', proto: 'tcp', listen_port: null, target_host: '', target_port: null,
           tunnel_proto: 'ws', tunnel_path: '', tunnel_user: '', tunnel_pass: '', enabled: true, remark: '' }
}

function authHeaders() {
  return { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + localStorage.getItem('token') }
}

async function api(path, opts = {}) {
  const r = await fetch(path, { ...opts, headers: { ...authHeaders(), ...(opts.headers || {}) } })
  const d = await r.json().catch(() => ({}))
  if (!r.ok) throw new Error(d.detail || `HTTP ${r.status}`)
  return d
}

async function load() {
  try { rules.value = await api('/api/rules') } catch (e) { flash(e.message, true) }
}

function openForm(r) {
  form.value = r ? { ...r, enabled: !!r.enabled } : emptyForm()
  showForm.value = true
}

function showCmd(r) {
  cmdRule.value = r
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
      await api('/api/rules/' + form.value.id, { method: 'PUT', body: JSON.stringify(body) })
    } else {
      await api('/api/rules', { method: 'POST', body: JSON.stringify(body) })
    }
    showForm.value = false
    await load()
    flash('已保存并同步到 gost')
  } catch (e) { flash(e.message, true) } finally { saving.value = false }
}

async function delRule(r) {
  if (!confirm(`确认删除规则「${r.name}」？`)) return
  try {
    await api('/api/rules/' + r.id, { method: 'DELETE' })
    await load()
    flash('已删除')
  } catch (e) { flash(e.message, true) }
}

async function toggleRule(r) {
  try {
    await api('/api/rules/' + r.id, { method: 'PUT', body: JSON.stringify({ enabled: !r.enabled }) })
    await load()
  } catch (e) { flash(e.message, true) }
}

async function applyRules() {
  applying.value = true
  try {
    const d = await api('/api/rules/apply', { method: 'POST' })
    flash(d.message || '同步完成')
  } catch (e) { flash(e.message, true) } finally { applying.value = false }
}

function flash(text, err = false) {
  msg.value = text; msgErr.value = err
  setTimeout(() => { msg.value = ''; msgErr.value = false }, 4000)
}

onMounted(load)
</script>

<style scoped>
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.actions { display: flex; gap: 10px; }
.btn {
  padding: 8px 14px; border: none; border-radius: 7px; cursor: pointer;
  background: #232833; color: #e6e8eb; font-size: 13px;
}
.btn.primary { background: #2f6feb; color: #fff; }
.btn.apply { background: #1f8f5a; color: #fff; }
.btn.danger { background: #3a1f24; color: #f85149; }
.btn.sm { padding: 5px 10px; font-size: 12px; }
.btn:disabled { opacity: 0.6; }
table { width: 100%; border-collapse: collapse; font-size: 14px; }
th, td { text-align: left; padding: 10px 12px; border-bottom: 1px solid #232833; }
th { color: #8a94a3; font-weight: 500; font-size: 13px; }
.empty { text-align: center; color: #8a94a3; padding: 30px 0; }
.mono { font-family: ui-monospace, monospace; font-size: 13px; }
.remark { color: #8a94a3; font-size: 13px; max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tag { padding: 2px 8px; border-radius: 5px; font-size: 12px; font-weight: 600; }
.tag.tcp { background: #1c3152; color: #6aa1ff; }
.tag.udp { background: #2e4a2a; color: #7ed36a; }
.dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 5px; }
.dot.on { background: #3fb950; } .dot.off { background: #666; }
.ops { display: flex; gap: 6px; }
.msg { padding: 10px 14px; border-radius: 7px; background: #1c3152; color: #6aa1ff; margin-bottom: 14px; font-size: 13px; }
.msg.err { background: #3a1f24; color: #f85149; }
.modal {
  position: fixed; inset: 0; background: rgba(0,0,0,0.6);
  display: flex; justify-content: center; align-items: center; z-index: 100;
}
.card {
  background: #1c2029; padding: 26px; border-radius: 12px; width: 420px;
  display: flex; flex-direction: column; gap: 12px;
}
.card h3 { margin-bottom: 4px; }
label { display: flex; flex-direction: column; gap: 5px; font-size: 13px; color: #8a94a3; }
input, select {
  padding: 9px 11px; border-radius: 7px; border: 1px solid #2c3340;
  background: #12151b; color: #e6e8eb; font-size: 14px;
}
.modal-ops { display: flex; justify-content: flex-end; gap: 10px; margin-top: 10px; }
</style>
