<template>
  <div class="flex h-full flex-col rounded-[10px] border bg-card p-5 transition-colors hover:border-hairline-strong sm:p-6" :class="{ 'opacity-60': soldOut }">
    <div class="min-w-0">
      <div class="flex items-start justify-between gap-3">
        <span v-if="categoryName" class="min-w-0 text-[12.5px] text-muted-foreground">{{ categoryName }}</span>
        <span
          v-if="planTags.isRecommended"
          class="ml-auto inline-flex shrink-0 items-center rounded-md border border-primary/20 bg-primary/5 px-2 py-0.5 text-[11.5px] font-medium leading-4 text-primary"
        >{{ t('atlas.plans.recommended') }}</span>
      </div>
      <h3 class="mt-1 line-clamp-2 text-[17px] font-semibold leading-snug">{{ title }}</h3>
      <p v-if="summary" class="mt-2 line-clamp-2 text-[13.5px] leading-relaxed text-muted-foreground">{{ summary }}</p>
    </div>

    <!-- 商家配置的真实标签 -->
    <div v-if="tags.length" class="mt-3 flex flex-wrap gap-1.5">
      <span v-for="tag in tags" :key="tag" class="inline-flex max-w-full items-center truncate rounded-md border px-2 py-0.5 text-[12px] text-muted-foreground">{{ tag }}</span>
    </div>

    <div class="mt-5 flex items-baseline gap-x-2">
      <template v-if="promo">
        <span class="text-[22px] font-semibold tabular-nums tracking-[-0.01em] text-primary">{{ formatPrice(getPromotionPriceAmount(product), siteCurrency) }}</span>
        <span class="text-[13.5px] text-muted-foreground line-through">{{ formatPrice(product.price_amount, siteCurrency) }}</span>
      </template>
      <span v-else class="text-[22px] font-semibold tabular-nums tracking-[-0.01em] text-foreground">{{ formatPrice(product.price_amount, siteCurrency) }}</span>
    </div>
    <div v-if="!soldOut" class="mt-1 text-[12.5px] text-muted-foreground">{{ stockLabel }}</div>

    <div class="mt-auto flex items-center gap-2 pt-5">
      <button
        v-if="!soldOut"
        type="button"
        class="inline-flex h-10 flex-1 items-center justify-center rounded-md bg-primary px-4 text-[14px] font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        :aria-label="t('products.quickBuyAria')"
        @click="$emit('quickBuy', product)"
      >{{ t('products.quickBuy') }}</button>
      <span v-else class="inline-flex h-10 flex-1 items-center justify-center rounded-md border px-4 text-[14px] text-muted-foreground" aria-disabled="true">{{ t('products.stockStatus.outOfStock') }}</span>
      <RouterLink
        :to="`/products/${product.slug}`"
        class="inline-flex h-10 flex-none items-center justify-center rounded-md border px-4 text-[14px] text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground"
      >{{ t('common.viewDetails') }}</RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLocalized, useProductLabels } from '../../../composables/useProduct'
import { resolveAtlasPlanTags } from '../../../utils/atlasPlanTags'

const props = defineProps<{ product: any }>()

defineEmits<{ quickBuy: [product: any] }>()

const { t } = useI18n()
const { getLocalizedText, siteCurrency, formatPrice } = useLocalized()
const { getStockStatusLabel, isSoldOut, hasPromotionPrice, getPromotionPriceAmount } = useProductLabels()

const title = computed(() => getLocalizedText(props.product?.title))
const summary = computed(() => getLocalizedText(props.product?.description))
const categoryName = computed(() => getLocalizedText(props.product?.category?.name))
const soldOut = computed(() => isSoldOut(props.product))
const promo = computed(() => hasPromotionPrice(props.product))
const stockLabel = computed(() => getStockStatusLabel(props.product))
const planTags = computed(() => resolveAtlasPlanTags(props.product?.tags))
const tags = computed(() => planTags.value.tags)
</script>
