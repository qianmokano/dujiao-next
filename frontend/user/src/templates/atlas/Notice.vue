<template>
  <div class="mx-auto w-full max-w-[840px] px-5 pb-10 sm:px-6">
    <header class="pb-2 pt-24 sm:pt-28">
      <h1 class="text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('nav.notice') }}</h1>
      <p class="mt-2 text-[14px] text-muted-foreground">{{ t('notice.subtitle') }}</p>
    </header>

    <!-- Loading -->
    <div v-if="loading" class="mt-6 divide-y border-t">
      <div v-for="i in 5" :key="i" class="flex items-baseline gap-6 py-5">
        <div class="h-3.5 w-24 flex-none rounded bg-secondary"></div>
        <div class="flex-1 space-y-2">
          <div class="h-4 w-2/5 rounded bg-secondary"></div>
          <div class="h-3 w-3/5 rounded bg-secondary"></div>
        </div>
      </div>
    </div>

    <!-- 公告列表：发丝分隔行 -->
    <template v-else-if="notices.length > 0">
      <div class="mt-6 divide-y border-t">
        <button
          v-for="notice in notices"
          :key="notice.id"
          type="button"
          class="group flex w-full items-baseline gap-x-6 gap-y-1 py-5 text-left"
          @click="goToNotice(notice.slug)"
        >
          <time class="w-28 flex-none text-[13px] tabular-nums text-muted-foreground">{{ formatDate(notice.published_at) }}</time>
          <div class="min-w-0 flex-1">
            <h2 class="text-[15.5px] font-medium text-foreground transition-colors group-hover:text-primary">{{ getLocalizedText(notice.title) }}</h2>
            <p v-if="getLocalizedText(notice.summary)" class="mt-1 line-clamp-2 text-[13.5px] leading-relaxed text-muted-foreground">{{ getLocalizedText(notice.summary) }}</p>
          </div>
          <ChevronRight class="h-4 w-4 flex-none self-center text-muted-foreground/60 transition-colors group-hover:text-primary" />
        </button>
      </div>

      <nav v-if="totalPages > 1" class="mt-8 flex flex-wrap items-center justify-center gap-1.5">
        <button type="button" class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="currentPage <= 1" :aria-label="t('common.previousBanner')" @click="changePage(currentPage - 1)"><ChevronLeft class="h-4 w-4" /></button>
        <span class="px-2.5 text-[13.5px] tabular-nums text-muted-foreground">{{ currentPage }} / {{ totalPages }}</span>
        <button type="button" class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="currentPage >= totalPages" :aria-label="t('common.nextBanner')" @click="changePage(currentPage + 1)"><ChevronRight class="h-4 w-4" /></button>
      </nav>
    </template>

    <!-- 空态 -->
    <div v-else class="mt-6 flex flex-col items-center gap-3 border-t py-16 text-center text-muted-foreground">
      <Bell class="h-10 w-10 opacity-50" :stroke-width="1.5" />
      <p class="text-[14px]">{{ t('notice.empty') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Bell, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { usePostList } from '../../composables/usePostList'

const { t } = useI18n()

const {
  loading, posts: notices, currentPage, totalPages,
  getLocalizedText, formatDate, goToPost: goToNotice, changePage,
} = usePostList('notice', { title: () => t('nav.notice'), canonicalPath: '/notice' })
</script>
