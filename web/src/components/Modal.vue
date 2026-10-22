<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      :aria-label="title"
      :aria-busy="busy"
      class="fixed inset-0 m-auto max-h-[calc(100dvh_-_2rem)] w-[calc(100%_-_2rem)] max-w-md overflow-y-auto rounded-xl border border-white/[0.1] bg-[#161b25] p-0 text-slate-100 shadow-[0_24px_80px_rgba(0,0,0,0.5)] backdrop:bg-black/70 backdrop:backdrop-blur-sm"
      @cancel.prevent="requestClose"
      @close="onClose"
      @click="onBackdropClick"
    >
      <header class="flex items-center gap-3 px-6 pt-6">
        <slot name="icon" />
        <h2 class="min-w-0 flex-1 text-base font-semibold tracking-tight text-white">{{ title }}</h2>
        <button
          type="button"
          aria-label="关闭弹窗"
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-white/[0.06] hover:text-slate-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400 disabled:cursor-not-allowed disabled:opacity-40"
          :disabled="busy"
          @click="requestClose"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <path d="m6 6 12 12M6 18 18 6" stroke-linecap="round" />
          </svg>
        </button>
      </header>
      <div class="px-6 pb-6 pt-4 text-sm leading-6 text-slate-400"><slot /></div>
      <footer v-if="$slots.footer" class="flex justify-end gap-2.5 border-t border-white/[0.07] bg-white/[0.02] px-6 py-4">
        <slot name="footer" />
      </footer>
    </dialog>
  </Teleport>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, required: true },
  busy: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])
const dialog = ref(null)
let previousOverflow

function restoreScroll() {
  if (previousOverflow !== undefined) {
    document.body.style.overflow = previousOverflow
    previousOverflow = undefined
  }
}

function syncOpen() {
  if (!dialog.value) return
  if (props.modelValue && !dialog.value.open) {
    previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    // 原生 dialog 提供背景隔离、键盘焦点约束和关闭后的焦点恢复。
    dialog.value.showModal()
  } else if (!props.modelValue && dialog.value.open) {
    dialog.value.close()
    restoreScroll()
  }
}

function requestClose() {
  if (!props.busy) emit('update:modelValue', false)
}

function onClose() {
  restoreScroll()
  if (props.modelValue) emit('update:modelValue', false)
}

function onBackdropClick(event) {
  if (event.target !== dialog.value) return
  const rect = dialog.value.getBoundingClientRect()
  if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) {
    requestClose()
  }
}

watch(() => props.modelValue, syncOpen, { flush: 'post' })
onMounted(syncOpen)
onBeforeUnmount(() => {
  dialog.value?.close()
  restoreScroll()
})
</script>
