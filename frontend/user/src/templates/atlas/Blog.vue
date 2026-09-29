<template>
  <div class="mx-auto w-full max-w-[840px] px-5 pb-10 sm:px-6">
    <header class="pb-2 pt-24 sm:pt-28">
      <h1 class="text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('nav.blog') }}</h1>
      <p class="mt-2 text-[14px] text-muted-foreground">{{ t('blog.subtitle') }}</p>
    </header>

    <!-- 搜索 -->
    <div class="relative my-6">
      <Search class="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
      <input
        v-model="searchKeyword"
        type="text"
        class="h-11 w-full rounded-md border bg-card pl-10 pr-10 text-[14.5px] text-foreground transition-colors placeholder:text-muted-foreground focus:border-hairline-strong focus:outline-none"
        :placeholder="t('blog.searchPlaceholder')"
        :aria-label="t('blog.searchPlaceholder')"
      />
      <button v-if="searchKeyword" type="button" class="absolute right-2.5 top-1/2 grid h-6 w-6 -translate-y-1/2 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" :aria-label="t('blog.searchClear')" @click="searchKeyword = ''"><X class="h-3.5 w-3.5" /></button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="divide-y border-t">
      <div v-for="i in 5" :key="i" class="flex items-baseline gap-6 py-5">
        <div class="h-3.5 w-24 flex-none rounded bg-secondary"></div>
        <div class="flex-1 space-y-2">
          <div class="h-4 w-2/5 rounded bg-secondary"></div>
          <div class="h-3 w-3/5 rounded bg-secondary"></div>
        </div>
      </div>
    </div>

    <!-- 文章列表：发丝分隔行 -->
    <template v-else-if="posts.length > 0">
      <div class="divide-y border-t">
        <article v-for="post in posts" :key="post.id">
          <RouterLink :to="`/blog/${post.slug}`" class="group flex flex-wrap items-baseline gap-x-6 gap-y-1 py-5">
            <time class="w-28 flex-none text-[13px] tabular-nums text-muted-foreground">{{ formatDate(post.published_at) }}</time>
            <div class="min-w-0 flex-1">
              <h2 class="text-[15.5px] font-medium text-foreground transition-colors group-hover:text-primary">{{ getLocalizedText(post.title) }}</h2>
              <p v-if="getLocalizedText(post.summary)" class="mt-1 line-clamp-2 text-[13.5px] leading-relaxed text-muted-foreground">{{ getLocalizedText(post.summary) }}</p>
            </div>
            <Badge v-if="post.type === 'notice'" variant="warning" size="sm">{{ t('nav.notice') }}</Badge>
          </RouterLink>
        </article>
      </div>

      <nav v-if="totalPages > 1" class="mt-8 flex flex-wrap items-center justify-center gap-1.5">
        <button type="button" class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="currentPage <= 1" :aria-label="t('common.previousBanner')" @click="changePage(currentPage - 1)"><ChevronLeft class="h-4 w-4" /></button>
        <span class="px-2.5 text-[13.5px] tabular-nums text-muted-foreground">{{ currentPage }} / {{ totalPages }}</span>
        <button type="button" class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="currentPage >= totalPages" :aria-label="t('common.nextBanner')" @click="changePage(currentPage + 1)"><ChevronRight class="h-4 w-4" /></button>
      </nav>
    </template>

    <!-- 空态 -->
    <div v-else class="flex flex-col items-center gap-3 border-t py-16 text-center text-muted-foreground">
      <BookOpen class="h-10 w-10 opacity-50" :stroke-width="1.5" />
      <p class="text-[14px]">{{ searchKeyword.trim() ? t('blog.noResults') : t('blog.empty') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { BookOpen, ChevronLeft, ChevronRight, Search, X } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { usePostList } from '../../composables/usePostList'

const { t } = useI18n()

const {
  loading, posts, currentPage, totalPages, searchKeyword,
  getLocalizedText, formatDate, changePage,
} = usePostList('blog', { title: () => t('nav.blog'), canonicalPath: '/blog' })
</script>
