<template>
  <RouterLink
    :to="`/products/${product.slug}`"
    class="group flex items-center gap-3.5 border-b px-1 py-4 transition-colors hover:bg-secondary/50 sm:px-2"
    :class="{ 'opacity-60': soldOut }"
  >
    <img
      v-if="coverImage"
      :src="coverImage"
      :alt="title"
      loading="lazy"
      class="h-11 w-11 flex-none rounded-md object-cover"
      @error="imageErrored = true"
    />

    <!-- Info -->
    <div class="flex min-w-0 flex-1 flex-col gap-0.5">
      <h3 class="truncate text-[14.5px] font-medium text-foreground">{{ title }}</h3>
      <span class="truncate text-[12.5px] text-muted-foreground">
        <template v-if="categoryName">{{ categoryName }} · </template>{{ stockLabel }}
      </span>
    </div>

    <!-- Price + action -->
    <div class="flex flex-none items-center gap-3 sm:gap-4">
      <div class="text-right">
        <template v-if="promo">
          <span class="block text-[14.5px] font-semibold tabular-nums text-primary">{{ formatPrice(getPromotionPriceAmount(product), siteCurrency) }}</span>
          <span class="block text-[11.5px] text-muted-foreground line-through">{{ formatPrice(product.price_amount, siteCurrency) }}</span>
        </template>
        <span v-else class="block text-[14.5px] font-semibold tabular-nums text-foreground">{{ formatPrice(product.price_amount, siteCurrency) }}</span>
      </div>
      <button
        v-if="!soldOut"
        type="button"
        class="flex-none rounded-md bg-primary px-3 py-1.5 text-[13px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 max-[640px]:hidden"
        :aria-label="t('products.quickBuyAria')"
        @click.prevent.stop="$emit('quickBuy', product)"
      >{{ t('products.quickBuy') }}</button>
      <span v-else class="flex-none text-[12.5px] text-muted-foreground max-[640px]:hidden" aria-disabled="true">{{ t('products.stockStatus.outOfStock') }}</span>
    </div>
  </RouterLink>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getFirstImageUrl, getImageUrl } from '../../../utils/image'
import { useLocalized, useProductLabels } from '../../../composables/useProduct'

const props = defineProps<{ product: any; index?: number }>()

defineEmits<{ quickBuy: [product: any] }>()

const { t } = useI18n()
const { getLocalizedText, siteCurrency, formatPrice } = useLocalized()
const { getStockStatusLabel, isSoldOut, hasPromotionPrice, getPromotionPriceAmount } = useProductLabels()

const title = computed(() => getLocalizedText(props.product?.title))
const categoryName = computed(() => getLocalizedText(props.product?.category?.name))
const soldOut = computed(() => isSoldOut(props.product))
const promo = computed(() => hasPromotionPrice(props.product))
const stockLabel = computed(() => getStockStatusLabel(props.product))

const imageErrored = ref(false)
const coverImage = computed(() => {
  if (imageErrored.value) return ''
  const primary = getFirstImageUrl(props.product?.images)
  if (primary) return primary
  const icon = props.product?.category?.icon
  return icon ? getImageUrl(icon) : ''
})
</script>
