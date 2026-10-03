<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { SSOCaptchaChallenge } from '../../api/auth'
import TurnstileCaptcha from './TurnstileCaptcha.vue'

defineProps<{ challenge: SSOCaptchaChallenge | null; modelValue: string }>()
const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'refresh'): void
}>()
const { t } = useI18n()
</script>

<template>
  <div v-if="challenge?.required" class="space-y-2">
    <label class="text-sm font-medium">{{ t('auth.common.passportCaptchaLabel') }}</label>
    <div v-if="challenge.type === 'image'" class="flex flex-wrap items-center gap-3">
      <button type="button" :aria-label="t('auth.common.refreshCaptcha')" @click="emit('refresh')">
        <img :src="challenge.image_base64" :alt="t('auth.common.passportCaptchaLabel')" class="h-12 rounded border" />
      </button>
      <input :value="modelValue" autocomplete="off" class="h-11 min-w-0 flex-1 rounded-md border border-input bg-transparent px-3 text-sm" :aria-label="t('auth.common.passportCaptchaLabel')" :placeholder="t('auth.common.captchaPlaceholder')" @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)" />
    </div>
    <TurnstileCaptcha v-else-if="challenge.type === 'turnstile'" :key="challenge.challenge" :site-key="challenge.site_key || ''" :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)" />
  </div>
</template>
