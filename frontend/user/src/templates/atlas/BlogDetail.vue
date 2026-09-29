<template>
  <div class="mx-auto w-full max-w-[760px] px-5 pb-10 sm:px-6">
    <!-- Loading -->
    <div v-if="loading" class="grid gap-4 pt-24 sm:pt-28">
      <div class="h-4 w-24 rounded bg-secondary"></div>
      <div class="h-9 w-3/4 rounded bg-secondary"></div>
      <div class="h-64 rounded-md bg-secondary"></div>
    </div>

    <!-- 文章 -->
    <article v-else-if="post">
      <nav class="flex flex-wrap items-center gap-1.5 py-2 pt-24 text-[13px] text-muted-foreground sm:pt-28">
        <RouterLink to="/" class="transition-colors hover:text-foreground">{{ t('nav.home') }}</RouterLink>
        <ChevronRight class="h-3.5 w-3.5 flex-none" />
        <RouterLink :to="backLink" class="transition-colors hover:text-foreground">{{ backText }}</RouterLink>
        <ChevronRight class="h-3.5 w-3.5 flex-none" />
        <span class="max-w-[240px] truncate text-foreground">{{ getLocalizedText(post.title) }}</span>
      </nav>

      <header class="mt-6 border-b pb-7">
        <div class="mb-3 flex items-center gap-3 text-[13px] text-muted-foreground">
          <Badge v-if="post.type === 'notice'" variant="warning" size="sm">{{ t('nav.notice') }}</Badge>
          <time class="tabular-nums">{{ formatDate(post.published_at) }}</time>
        </div>
        <h1 class="text-[28px] font-semibold leading-[1.25] tracking-[-0.01em] sm:text-[32px]">{{ getLocalizedText(post.title) }}</h1>
        <p v-if="post.summary" class="mt-3.5 text-[15px] leading-relaxed text-muted-foreground">{{ getLocalizedText(post.summary) }}</p>
      </header>

      <div
        class="prose mt-7 max-w-none dark:prose-invert prose-a:text-primary prose-img:rounded-md"
        v-html="processHtmlForDisplay(getLocalizedText(post.content))"
      ></div>

      <!-- 相关商品 -->
      <section v-if="relatedProducts.length" class="mt-10 border-t pt-7">
        <h2 class="mb-4 text-[16px] font-semibold">{{ t('blog.relatedProducts') }}</h2>
        <div class="grid gap-3 sm:grid-cols-2">
          <RouterLink
            v-for="rp in relatedProducts"
            :key="rp.id"
            :to="`/products/${rp.slug}`"
            class="flex items-center gap-3.5 rounded-md border bg-card p-3 transition-colors hover:border-hairline-strong"
          >
            <div v-if="rp.image" class="h-12 w-12 flex-none overflow-hidden rounded-md">
              <img :src="getImageUrl(rp.image)" :alt="getLocalizedText(rp.title)" loading="lazy" class="h-full w-full object-cover" />
            </div>
            <div class="min-w-0">
              <div class="truncate text-[14px] font-medium">{{ getLocalizedText(rp.title) }}</div>
              <div class="mt-0.5 text-[13px] text-primary">{{ formatPrice(rp.price_amount) }}</div>
            </div>
          </RouterLink>
        </div>
      </section>

      <footer class="mt-10 border-t pt-6">
        <RouterLink :to="backLink" class="inline-flex items-center gap-2 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">
          <ArrowLeft class="h-4 w-4" /> {{ backText }}
        </RouterLink>
      </footer>
    </article>

    <!-- 不存在 -->
    <div v-else class="my-10 flex flex-col items-center gap-3 border-t py-16 text-center text-muted-foreground">
      <AlertCircle class="h-10 w-10 opacity-50" :stroke-width="1.5" />
      <p class="text-[14px]">{{ t('blogDetail.notFound') }}</p>
      <RouterLink to="/blog" class="mt-2 inline-flex items-center rounded-md bg-primary px-4 py-2 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90">{{ t('blogDetail.backToBlog') }}</RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AlertCircle, ArrowLeft, ChevronRight } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { getImageUrl } from '../../utils/image'
import { processHtmlForDisplay } from '../../utils/content'
import { useBlogDetail } from '../../composables/useBlogDetail'

const { t } = useI18n()

const {
  loading, post, relatedProducts, getLocalizedText, formatDate, formatPrice, backLink, backText,
} = useBlogDetail()
</script>
