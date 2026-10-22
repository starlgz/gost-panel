<template>
  <div ref="root" class="relative">
    <!-- 触发器 -->
    <button
      type="button"
      class="flex h-[42px] w-full items-center justify-between gap-2 rounded-lg border px-3.5 text-sm outline-none transition-all duration-200"
      :class="open
        ? 'border-blue-500/70 ring-[3px] ring-blue-500/10'
        : 'border-white/[0.08] hover:border-white/[0.16]'"
      @click="open = !open"
      @keydown.down.prevent="move(1)"
      @keydown.up.prevent="move(-1)"
      @keydown.enter.prevent="open ? pick(highlight) : (open = true)"
      @keydown.escape="open = false"
    >
      <span :class="selected ? 'text-white' : 'text-slate-500'">{{ display }}</span>
      <svg
        class="h-4 w-4 shrink-0 text-slate-500 transition-transform duration-200"
        :class="open && 'rotate-180'"
        fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8"
      >
        <path d="M6 9l6 6 6-6" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </button>

    <!-- 下拉面板 -->
    <transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="-translate-y-1 scale-[0.98] opacity-0"
      enter-to-class="translate-y-0 scale-100 opacity-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="translate-y-0 scale-100 opacity-100"
      leave-to-class="-translate-y-1 opacity-0"
    >
      <div
        v-if="open"
        class="absolute left-0 right-0 z-20 mt-1.5 origin-top overflow-hidden rounded-xl border border-white/[0.08] bg-[#1b2230] p-1.5 shadow-[0_16px_48px_rgba(0,0,0,0.45)]"
      >
        <button
          v-for="(opt, i) in options"
          :key="opt.value"
          type="button"
          class="flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2.5 text-left text-sm transition-colors duration-100"
          :class="i === highlight ? 'bg-white/[0.06] text-white' : 'text-slate-300'"
          @mouseenter="highlight = i"
          @click="pick(i)"
        >
          <span class="min-w-0">
            <span class="block truncate">{{ opt.label }}</span>
            <span v-if="opt.desc" class="mt-0.5 block truncate text-xs text-slate-500">{{ opt.desc }}</span>
          </span>
          <svg
            v-if="opt.value === modelValue"
            class="h-4 w-4 shrink-0 text-blue-400"
            fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
          >
            <path d="M5 13l4 4L19 7" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </button>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  options: { type: Array, required: true }, // [{ value, label, desc? }]
  placeholder: { type: String, default: '请选择' },
})
const emit = defineEmits(['update:modelValue'])

const open = ref(false)
const highlight = ref(0)
const root = ref(null)

const selected = computed(() => props.options.some((o) => o.value === props.modelValue))
const display = computed(() => {
  const hit = props.options.find((o) => o.value === props.modelValue)
  return hit ? hit.label : props.placeholder
})

function pick(i) {
  emit('update:modelValue', props.options[i].value)
  open.value = false
}

function move(dir) {
  if (!open.value) {
    open.value = true
    return
  }
  highlight.value = (highlight.value + dir + props.options.length) % props.options.length
}

function onClickOutside(e) {
  if (root.value && !root.value.contains(e.target)) open.value = false
}

function syncHighlight() {
  const idx = props.options.findIndex((o) => o.value === props.modelValue)
  highlight.value = idx >= 0 ? idx : 0
}

onMounted(() => {
  document.addEventListener('click', onClickOutside)
  syncHighlight()
})
onBeforeUnmount(() => document.removeEventListener('click', onClickOutside))
</script>
