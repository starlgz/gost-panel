<template>
  <Modal :model-value="modelValue" :title="title" :busy="busy" @update:model-value="$emit('update:modelValue', $event)">
    <template #icon>
      <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border" :class="danger ? 'border-red-400/15 bg-red-500/10 text-red-400' : 'border-blue-400/15 bg-blue-500/10 text-blue-400'">
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7" aria-hidden="true">
          <path d="m10.3 4-8 14a2 2 0 0 0 1.7 3h16a2 2 0 0 0 1.7-3l-8-14a2 2 0 0 0-3.4 0ZM12 9v4m0 4h.01" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </span>
    </template>
    <p>{{ description }}</p>
    <slot />
    <p v-if="error" role="alert" class="mt-4 rounded-lg border border-red-400/20 bg-red-500/[0.08] px-3 py-2 text-xs text-red-300">{{ error }}</p>
    <template #footer>
      <button type="button" autofocus class="rounded-lg border border-white/[0.1] px-4 py-2 text-xs font-medium text-slate-300 transition-colors hover:bg-white/[0.06] focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400 disabled:cursor-not-allowed disabled:opacity-50" :disabled="busy" @click="$emit('update:modelValue', false)">
        {{ cancelText }}
      </button>
      <button type="button" class="inline-flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-semibold text-white transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-60" :class="danger ? 'bg-red-600 hover:bg-red-500 focus-visible:outline-red-400' : 'bg-blue-600 hover:bg-blue-500 focus-visible:outline-blue-400'" :disabled="busy" @click="$emit('confirm')">
        <svg v-if="busy" class="h-3.5 w-3.5 animate-spin" fill="none" viewBox="0 0 24 24" aria-hidden="true"><circle class="opacity-25" cx="12" cy="12" r="9" stroke="currentColor" stroke-width="3" /><path class="opacity-90" d="M12 3a9 9 0 0 1 9 9" stroke="currentColor" stroke-width="3" stroke-linecap="round" /></svg>
        {{ busy ? '处理中…' : confirmText }}
      </button>
    </template>
  </Modal>
</template>

<script setup>
import Modal from '@/components/Modal.vue'

defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, required: true },
  description: { type: String, required: true },
  confirmText: { type: String, default: '确认' },
  cancelText: { type: String, default: '取消' },
  danger: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  error: { type: String, default: '' },
})
defineEmits(['update:modelValue', 'confirm'])
</script>
