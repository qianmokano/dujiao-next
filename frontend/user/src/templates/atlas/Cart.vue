<template>
  <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 sm:px-6">
    <nav class="flex flex-wrap items-center gap-1.5 pb-2 pt-24 text-[13px] text-muted-foreground sm:pt-28">
      <RouterLink to="/" class="transition-colors hover:text-foreground">{{ t('nav.home') }}</RouterLink>
      <ChevronRight class="h-3.5 w-3.5 flex-none" />
      <span class="text-foreground">{{ t('cart.title') }}</span>
    </nav>

    <header class="mb-6 mt-2">
      <h1 class="mb-1.5 text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('cart.title') }}</h1>
      <p class="text-[14px] text-muted-foreground">{{ t('cart.subtitle') }}</p>
    </header>

    <VaultCheckoutSteps current="cart" />

    <!-- 空购物车 -->
    <div v-if="cartItems.length === 0" class="my-8 flex flex-col items-center gap-3 rounded-[10px] border border-dashed py-16 text-center text-muted-foreground">
      <ShoppingCart class="h-10 w-10 opacity-50" :stroke-width="1.5" />
      <p class="text-[14px]">{{ t('cart.empty') }}</p>
      <RouterLink to="/products" class="mt-2 inline-flex items-center rounded-md bg-primary px-4 py-2 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90">{{ t('cart.emptyAction') }}</RouterLink>
    </div>

    <div v-else class="grid items-start gap-8 lg:grid-cols-[1fr_320px]">
      <!-- 商品列表 -->
      <div class="divide-y border-t">
        <article v-for="item in cartItems" :key="cartItemKey(item)" class="flex gap-4 py-5">
          <RouterLink :to="`/products/${item.slug}`" class="relative grid h-[72px] w-[72px] flex-none place-items-center overflow-hidden rounded-md border bg-secondary">
            <img v-if="cartItemImage(item)" :src="cartItemImage(item)" :alt="getLocalizedText(item.title)" loading="lazy" class="absolute inset-0 h-full w-full object-cover" />
            <Package v-else class="h-8 w-8 text-muted-foreground/50" :stroke-width="1.5" />
          </RouterLink>

          <div class="min-w-0 flex-1">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <RouterLink :to="`/products/${item.slug}`" class="block truncate text-[15px] font-medium transition-colors hover:text-primary">{{ getLocalizedText(item.title) }}</RouterLink>
                <p class="mt-1 text-[13px] text-muted-foreground">{{ t('cart.priceLabel') }}：{{ formatPrice(item.priceAmount, totalCurrency) }}</p>
                <p v-if="itemSkuDisplay(item)" class="mt-1 text-[13px] text-muted-foreground">{{ t('cart.skuLabel') }}：{{ itemSkuDisplay(item) }}</p>
                <p v-if="itemStockHint(item)" class="mt-1 text-[13px] text-muted-foreground">{{ itemStockHint(item) }}</p>
                <p class="mt-1.5 text-[12.5px] text-muted-foreground">
                  {{ item.purchaseType === 'guest' ? t('productPurchase.guest') : t('productPurchase.member') }} · {{ item.fulfillmentType === 'auto' ? t('products.fulfillmentType.auto') : t('products.fulfillmentType.manual') }}
                </p>
              </div>
              <button type="button" class="grid h-9 w-9 flex-none place-items-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive" :aria-label="t('cart.remove')" @click="removeWithUndo(item)">
                <Trash2 class="h-4 w-4" />
              </button>
            </div>

            <div class="mt-3.5 flex flex-wrap items-center justify-between gap-3">
              <div class="inline-flex items-center overflow-hidden rounded-md border">
                <button type="button" class="grid h-9 w-[36px] place-items-center bg-card text-foreground transition-colors hover:bg-secondary disabled:opacity-35" :aria-label="t('cart.remove')" :disabled="item.quantity <= itemPurchaseMin(item)" @click="updateQty(item, item.quantity - 1)"><Minus class="h-3.5 w-3.5" /></button>
                <input
                  type="number"
                  class="h-9 w-[46px] border-x bg-card text-center text-[13.5px] font-medium tabular-nums outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                  :value="item.quantity"
                  :min="itemPurchaseMin(item)"
                  :max="itemMaxQuantity(item)"
                  @change="handleQtyChange(item, $event)"
                />
                <button type="button" class="grid h-9 w-[36px] place-items-center bg-card text-foreground transition-colors hover:bg-secondary disabled:opacity-35" :aria-label="t('cart.remove')" :disabled="item.quantity >= itemMaxQuantity(item)" @click="updateQty(item, item.quantity + 1)"><Plus class="h-3.5 w-3.5" /></button>
              </div>
              <div class="text-right">
                <span class="block text-[12px] text-muted-foreground">{{ t('checkout.totalPriceLabel') }}</span>
                <strong class="text-[15px] font-semibold tabular-nums">{{ itemSubtotal(item) }}</strong>
              </div>
            </div>

            <div v-if="quantityWarning(item)" class="mt-3 rounded-md bg-warning/10 px-3 py-2 text-[13px] text-warning">{{ quantityWarning(item) }}</div>
          </div>
        </article>
      </div>

      <!-- 汇总 -->
      <aside class="sticky top-[84px] rounded-[10px] border bg-card p-6">
        <h2 class="mb-4 text-[16px] font-semibold">{{ t('cart.summaryTitle') }}</h2>
        <div class="grid gap-3">
          <div class="flex items-center justify-between text-[13.5px] text-muted-foreground">
            <span>{{ t('cart.itemsCount') }}</span>
            <span class="font-medium text-foreground">{{ totalItems }}</span>
          </div>
          <div class="flex items-center justify-between border-t pt-3 text-foreground">
            <span class="text-[14px]">{{ t('cart.totalLabel') }}</span>
            <span class="text-[22px] font-semibold tabular-nums text-primary">{{ formatPrice(totalAmount, totalCurrency) }}</span>
          </div>
        </div>
        <p class="my-4 rounded-md bg-secondary px-3 py-2.5 text-[12.5px] leading-relaxed text-muted-foreground">{{ t('cart.disclaimer') }}</p>
        <RouterLink to="/checkout" class="inline-flex h-11 w-full items-center justify-center gap-1.5 rounded-md bg-primary text-[14.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90">{{ t('cart.checkout') }}</RouterLink>
        <RouterLink to="/products" class="mt-2.5 inline-flex h-11 w-full items-center justify-center rounded-md border text-[14.5px] text-foreground transition-colors hover:border-hairline-strong">{{ t('cart.emptyAction') }}</RouterLink>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ChevronRight, Minus, Package, Plus, ShoppingCart, Trash2 } from 'lucide-vue-next'
import VaultCheckoutSteps from '../vault/components/VaultCheckoutSteps.vue'
import { useCart } from '../../composables/useCart'

const { t } = useI18n()

const {
  getLocalizedText, formatPrice, totalCurrency,
  cartItems, totalItems, totalAmount,
  cartItemKey, cartItemImage, itemSkuDisplay, itemSubtotal, itemStockHint, quantityWarning,
  itemPurchaseMin, itemMaxQuantity,
  removeWithUndo, updateQty, handleQtyChange,
} = useCart()
</script>
