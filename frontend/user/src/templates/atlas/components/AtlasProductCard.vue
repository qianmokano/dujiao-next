<template>
  <RouterLink :to="`/products/${product.slug}`" class="flex h-full flex-col rounded-[10px] border bg-card p-4 text-left transition-colors hover:border-hairline-strong" :class="{ 'opacity-60': soldOut }">
    <div class="min-w-0">
      <span v-if="categoryName" class="text-[12.5px] text-muted-foreground">{{ categoryName }}</span>
      <h3 class="mt-1 line-clamp-2 text-[15.5px] font-semibold leading-snug">{{ title }}</h3>
    </div>

    <div class="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-[12.5px] text-muted-foreground">
      <span>{{ getFulfillmentTypeLabel(product.fulfillment_type) }}</span>
      <span aria-hidden="true">·</span>
      <span>{{ stockLabel }}</span>
    </div>

    <div class="mt-auto flex items-end justify-between gap-2 pt-4">
      <div class="flex min-w-0 flex-wrap items-baseline gap-x-1.5">
        <template v-if="promo">
          <span class="text-[18px] font-semibold tabular-nums text-primary">{{ formatPrice(getPromotionPriceAmount(product), siteCurrency) }}</span>
          <span class="text-[12.5px] text-muted-foreground line-through">{{ formatPrice(product.price_amount, siteCurrency) }}</span>
        </template>
        <span v-else class="text-[18px] font-semibold tabular-nums text-foreground">{{ formatPrice(product.price_amount, siteCurrency) }}</span>
      </div>
      <span v-if="soldOut" class="flex-none text-[13px] text-muted-foreground" aria-disabled="true">{{ t('products.stockStatus.outOfStock') }}</span>
      <button
        v-else
        type="button"
        class="flex-none rounded-md bg-primary px-3 py-1.5 text-[13px] font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        :aria-label="t('products.quickBuyAria')"
        @click.prevent.stop="$emit('quickBuy', product)"
      >{{ t('products.quickBuy') }}</button>
    </div>
  </RouterLink>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLocalized, useProductLabels } from '../../../composables/useProduct'

const props = defineProps<{ product: any; index?: number }>()

defineEmits<{ quickBuy: [product: any] }>()

const { t } = useI18n()
const { getLocalizedText, siteCurrency, formatPrice } = useLocalized()
const { getStockStatusLabel, getFulfillmentTypeLabel, isSoldOut, hasPromotionPrice, getPromotionPriceAmount } = useProductLabels()

const title = computed(() => getLocalizedText(props.product?.title))
const categoryName = computed(() => getLocalizedText(props.product?.category?.name))
const soldOut = computed(() => isSoldOut(props.product))
const promo = computed(() => hasPromotionPrice(props.product))
const stockLabel = computed(() => getStockStatusLabel(props.product))
</script>
