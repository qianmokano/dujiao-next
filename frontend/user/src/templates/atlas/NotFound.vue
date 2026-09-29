<template>
  <div class="mx-auto flex min-h-[60vh] w-full max-w-[760px] flex-col items-start justify-center px-5 py-16 sm:px-6">
    <p class="text-[13px] font-medium tabular-nums text-muted-foreground">{{ brandSiteName }}</p>
    <p class="mt-4 text-[72px] font-semibold leading-none tracking-[-0.02em] text-foreground sm:text-[88px]">404</p>
    <h1 class="mt-5 text-[22px] font-semibold tracking-[-0.01em]">{{ t('notFoundPage.title') }}</h1>
    <p class="mt-2.5 max-w-[52ch] text-[14.5px] leading-relaxed text-muted-foreground">{{ t('notFoundPage.description', { site: brandSiteName }) }}</p>

    <div class="mt-8 flex flex-wrap gap-3">
      <RouterLink to="/" class="inline-flex h-11 items-center rounded-md bg-primary px-5 text-[14px] font-medium text-primary-foreground transition-colors hover:bg-primary/90">
        <Home class="mr-2 h-4 w-4" /> {{ t('notFoundPage.backHome') }}
      </RouterLink>
      <button type="button" class="inline-flex h-11 items-center rounded-md border px-5 text-[14px] text-foreground transition-colors hover:border-hairline-strong" @click="goBack">
        <ArrowLeft class="mr-2 h-4 w-4" /> {{ t('notFoundPage.backPrevious') }}
      </button>
    </div>

    <div class="mt-10 flex flex-wrap gap-x-6 gap-y-2 border-t pt-6 text-[13.5px]">
      <RouterLink to="/products" class="text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.products') }}</RouterLink>
      <RouterLink to="/notice" class="text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.notice') }}</RouterLink>
      <RouterLink to="/blog" class="text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.blog') }}</RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Home } from 'lucide-vue-next'
import { useNotFound } from '../../composables/useNotFound'

const { t } = useI18n()

const { brandSiteName, goBack } = useNotFound()
</script>
