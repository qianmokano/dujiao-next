<template>
  <Teleport to="body">
    <div class="fixed z-[100] pointer-events-none" :class="[positionClass, appearance === 'atlas' ? 'w-max max-w-[calc(100vw-32px)]' : '']">
      <TransitionGroup
        enter-active-class="transition duration-300 ease-out"
        enter-from-class="opacity-0 translate-y-2 scale-95"
        enter-to-class="opacity-100 translate-y-0 scale-100"
        leave-active-class="transition duration-200 ease-in"
        leave-from-class="opacity-100 translate-y-0 scale-100"
        leave-to-class="opacity-0 translate-y-2 scale-95"
      >
        <div
          v-for="item in toasts"
          :key="item.id"
          :role="appearance === 'atlas' ? feedbackRole(item.type) : 'alert'"
          class="pointer-events-auto mb-2"
          :class="appearance === 'atlas' ? 'atlas-feedback atlas-toast max-w-full' : ['flex items-center gap-2 rounded-xl border px-4 py-3 text-sm font-medium shadow-lg backdrop-blur-xl', typeClass(item.type)]"
        >
          <component :is="item.type === 'success' ? Check : (item.type === 'error' ? (appearance === 'atlas' ? CircleX : X) : Info)" class="h-4 w-4 shrink-0" :class="appearance === 'atlas' ? ['atlas-feedback-icon', feedbackIconClass(item.type)] : ''" aria-hidden="true" />
          <span class="flex-1" :class="appearance === 'atlas' ? 'atlas-feedback-message' : ''">{{ item.message }}</span>
          <button
            v-if="item.action"
            class="ml-2 shrink-0 rounded-lg px-2.5 py-1 text-xs font-bold underline underline-offset-2 transition-colors hover:opacity-80"
            @click="handleAction(item)"
          >
            {{ item.action.label }}
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { Check, CircleX, X, Info } from 'lucide-vue-next'
import { useToast, type ToastItem } from '../composables/useToast'
import { feedbackIconClass, feedbackRole } from '../utils/feedback'

withDefaults(defineProps<{ appearance?: 'atlas' | 'default' }>(), { appearance: 'default' })

const { toasts, removeToast } = useToast()

const positionClass = 'bottom-6 left-1/2 -translate-x-1/2 md:bottom-auto md:top-6 flex flex-col items-center'

const typeClass = (type: string) => {
  switch (type) {
    case 'success':
      return 'border-success/40 bg-success/10 text-success'
    case 'error':
      return 'border-destructive/40 bg-destructive/10 text-destructive'
    default:
      return 'border-border bg-card text-card-foreground'
  }
}

const handleAction = (item: ToastItem) => {
  item.action?.onClick()
  removeToast(item.id)
}
</script>
