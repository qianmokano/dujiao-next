<template>
  <div class="space-y-3 rounded-xl border bg-muted/30 p-5 text-sm">
    <p class="text-muted-foreground">{{ t('personalCenter.security.unifiedIdentityHint') }}</p>
    <p class="text-muted-foreground">{{ t('personalCenter.security.unifiedSyncHint') }}</p>
    <a v-if="href" :href="href" class="inline-flex font-medium text-primary underline underline-offset-4">
      {{ recovery ? t('auth.login.forgot') : t('personalCenter.security.manageUnifiedIdentity') }}
    </a>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ href: string; recovery?: boolean }>()
const { t } = useI18n()
</script>
