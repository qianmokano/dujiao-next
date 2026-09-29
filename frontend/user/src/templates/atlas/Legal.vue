<template>
  <div class="mx-auto w-full max-w-[760px] px-5 pb-10 sm:px-6">
    <!-- Loading -->
    <div v-if="loading" class="grid gap-4 pt-24 sm:pt-28">
      <div class="h-8 w-2/5 rounded bg-secondary"></div>
      <div class="h-64 rounded-md bg-secondary"></div>
    </div>

    <template v-else>
      <header class="border-b pb-6 pt-24 sm:pt-28">
        <h1 class="text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ title }}</h1>
      </header>

      <div
        v-if="content"
        class="prose mt-7 max-w-none dark:prose-invert prose-a:text-primary prose-img:rounded-md"
        v-html="content"
      ></div>

      <div v-else class="mt-6 flex flex-col items-center gap-3 border-t py-16 text-center text-muted-foreground">
        <FileText class="h-10 w-10 opacity-50" :stroke-width="1.5" />
        <p class="text-[14px]">{{ t('common.noContent') }}</p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { FileText } from 'lucide-vue-next'
import { useLegal } from '../../composables/useLegal'

const { t } = useI18n()

const props = defineProps<{
  type: 'terms' | 'privacy'
}>()

const { loading, title, content } = useLegal(() => props.type)
</script>
