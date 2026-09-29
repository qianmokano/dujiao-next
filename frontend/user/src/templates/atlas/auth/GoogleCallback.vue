<template>
  <div class="mx-auto flex min-h-[60vh] w-full max-w-[400px] flex-col justify-center px-5 py-16 sm:px-6">
    <div class="rounded-[10px] border bg-card p-8">
      <div v-if="loading" class="grid justify-items-center gap-4 text-center">
        <span
          class="h-8 w-8 animate-spin rounded-full border-[3px] border-border border-t-primary motion-reduce:animate-none"
          aria-hidden="true"
        ></span>
        <p class="text-[14px] text-muted-foreground">{{ t('auth.googleCallback.processing') }}</p>
      </div>
      <div v-else-if="errMsg" class="grid justify-items-center gap-4 text-center">
        <AlertCircle class="h-8 w-8 text-destructive" :stroke-width="1.5" />
        <p class="text-[14.5px] font-medium text-foreground">{{ errMsg }}</p>
        <RouterLink :to="retryPath" class="mt-1 inline-flex h-10 items-center rounded-md border px-4 text-[13.5px] text-foreground transition-colors hover:border-hairline-strong">{{ t('auth.googleCallback.back') }}</RouterLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AlertCircle } from 'lucide-vue-next'
import { useGoogleRedirectCallback } from '../../../composables/useGoogleRedirectCallback'

const { t } = useI18n()
const { loading, errMsg, retryPath } = useGoogleRedirectCallback()
</script>
