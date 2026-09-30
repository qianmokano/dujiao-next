<template>
  <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 sm:px-6">
    <!-- Loading -->
    <div v-if="loading" class="grid gap-10 pt-24 sm:pt-28 lg:grid-cols-2">
      <div class="h-[360px] rounded-[10px] bg-secondary"></div>
      <div class="grid content-start gap-4 pt-2">
        <div class="h-7 w-3/5 rounded bg-secondary"></div>
        <div class="h-10 w-2/5 rounded bg-secondary"></div>
        <div class="h-[120px] rounded bg-secondary"></div>
      </div>
    </div>

    <!-- Content -->
    <template v-else-if="product">
      <nav class="flex flex-wrap items-center gap-1.5 pb-2 pt-24 text-[13px] text-muted-foreground sm:pt-28">
        <RouterLink to="/" class="transition-colors hover:text-foreground">{{ t('nav.home') }}</RouterLink>
        <ChevronRight class="h-3.5 w-3.5 flex-none" />
        <RouterLink to="/products" class="transition-colors hover:text-foreground">{{ t('nav.products') }}</RouterLink>
        <ChevronRight class="h-3.5 w-3.5 flex-none" />
        <span class="min-w-0 truncate text-foreground">{{ getLocalizedText(product.title) }}</span>
      </nav>

      <section class="grid gap-10 py-6 lg:grid-cols-2">
        <!-- 图区 -->
        <div>
          <div class="relative grid h-[360px] place-items-center overflow-hidden rounded-[10px] border bg-secondary">
            <img v-if="currentImage" :src="currentImage" :alt="getLocalizedText(product.title)" class="absolute inset-0 h-full w-full object-cover" />
            <Package v-else class="h-[88px] w-[88px] text-muted-foreground/40" :stroke-width="1.2" />
          </div>
          <div v-if="images.length > 1" class="mt-3 flex flex-wrap gap-2.5">
            <button
              v-for="(img, idx) in images"
              :key="idx"
              class="h-[56px] w-[70px] overflow-hidden rounded-md border-2 bg-secondary"
              :class="img === currentImage ? 'border-primary' : 'border-border'"
              @click="currentImage = img"
            >
              <img :src="img" :alt="`${idx + 1}`" loading="lazy" class="h-full w-full object-cover" />
            </button>
          </div>
        </div>

        <!-- 购买区 -->
        <div>
          <span v-if="categoryName" class="block truncate text-[13px] text-muted-foreground">{{ categoryName }}</span>
          <h1 class="mb-3 mt-1.5 text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ getLocalizedText(product.title) }}</h1>

          <div class="mb-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-[13px] text-muted-foreground">
            <span>{{ getStockStatusLabel(product) }}</span>
            <span aria-hidden="true">·</span>
            <span>{{ getFulfillmentTypeLabel(product.fulfillment_type) }}</span>
            <span aria-hidden="true">·</span>
            <span>{{ getPurchaseTypeLabel(product.purchase_type) }}</span>
          </div>

          <div v-if="product.tags && product.tags.length" class="mb-1 mt-3 flex flex-wrap gap-1.5">
            <span v-for="(tag, i) in product.tags" :key="i" class="inline-flex items-center rounded-md border px-2 py-0.5 text-[12px] text-muted-foreground">{{ tag }}</span>
          </div>

          <!-- 价格 -->
          <div class="my-6 border-b pb-6">
            <div class="mb-2.5 flex flex-wrap items-center gap-2">
              <span class="text-[13px] text-muted-foreground">{{ t('products.price') }}</span>
              <span v-if="(selectedSku && hasSkuPromotionPrice(selectedSku)) || (!selectedSku && hasPromotionPrice(product))" class="inline-flex items-center rounded-md bg-primary/10 px-2 py-0.5 text-[12px] font-medium text-primary">{{ t('products.promotionTag') }}</span>
              <span v-if="showSelectedSkuMemberBadge" class="inline-flex items-center rounded-md bg-warning/10 px-2 py-0.5 text-[12px] font-medium text-warning">{{ t('products.memberPriceTag') }}</span>
              <span v-if="hasSelectedSkuWholesalePrice" class="inline-flex items-center rounded-md bg-success/10 px-2 py-0.5 text-[12px] font-medium text-success">{{ t('products.wholesaleTag') }}</span>
            </div>

            <!-- 1. SKU 批发价 -->
            <template v-if="selectedSku && hasSelectedSkuWholesalePrice">
              <div class="flex flex-wrap items-baseline gap-3">
                <span class="text-[34px] font-semibold tabular-nums tracking-[-0.01em]" :class="selectedSkuWholesaleFinalIsMember ? 'text-warning' : 'text-success'">{{ formatPrice(selectedSkuWholesaleFinalPrice!, siteCurrency) }}</span>
                <span class="text-muted-foreground line-through">{{ formatPrice(selectedSku.price_amount, siteCurrency) }}</span>
              </div>
              <p class="mt-2 text-[13px]" :class="selectedSkuWholesaleFinalIsMember ? 'text-warning' : 'text-success'">
                {{ selectedSkuWholesaleFinalIsMember ? t('products.memberPriceTag') : t('products.wholesaleTag') }} · {{ t('products.saveAmount') }} {{ formatPrice(Number(selectedSku.price_amount) - Number(selectedSkuWholesaleFinalPrice), siteCurrency) }}
              </p>
            </template>
            <!-- 2. SKU 促销价 -->
            <template v-else-if="selectedSku && hasSkuPromotionPrice(selectedSku)">
              <div class="flex flex-wrap items-baseline gap-3">
                <span v-if="selectedSkuPromotionFinalIsMember" class="text-[34px] font-semibold tabular-nums tracking-[-0.01em] text-warning">{{ formatPrice(selectedSkuPromotionFinalPrice!, siteCurrency) }}</span>
                <span v-else class="text-[34px] font-semibold tabular-nums tracking-[-0.01em] text-primary">{{ formatPrice(selectedSkuPromotionPrice!, siteCurrency) }}</span>
                <span class="text-muted-foreground line-through">{{ formatPrice(selectedSku.price_amount, siteCurrency) }}</span>
              </div>
              <p v-if="selectedSkuPromotionFinalIsMember" class="mt-2 text-[13px] text-warning">{{ t('products.memberPriceTag') }} · {{ t('products.saveAmount') }} {{ formatPrice(Number(selectedSku.price_amount) - Number(selectedSkuPromotionFinalPrice), siteCurrency) }}</p>
              <p v-else class="mt-2 text-[13px] text-destructive">{{ t('products.saveAmount') }} {{ formatPrice(getSkuPromotionSaveAmount(selectedSku), siteCurrency) }}</p>
            </template>
            <!-- 3. SKU 会员价 -->
            <template v-else-if="selectedSku && hasMemberPrice">
              <div class="flex flex-wrap items-baseline gap-3">
                <span class="text-[34px] font-semibold tabular-nums tracking-[-0.01em] text-warning">{{ formatPrice(selectedSkuMemberPrice!, siteCurrency) }}</span>
                <span class="text-muted-foreground line-through">{{ formatPrice(selectedSku.price_amount, siteCurrency) }}</span>
              </div>
              <p class="mt-2 text-[13px] text-warning">{{ t('products.memberPriceTag') }} · {{ t('products.saveAmount') }} {{ formatPrice(Number(selectedSku.price_amount) - selectedSkuMemberPrice!, siteCurrency) }}</p>
            </template>
            <!-- 4. SKU 原价 -->
            <div v-else-if="selectedSku" class="flex flex-wrap items-baseline gap-3">
              <span class="text-[34px] font-semibold tabular-nums tracking-[-0.01em] text-foreground">{{ formatPrice(selectedSku.price_amount, siteCurrency) }}</span>
            </div>
            <!-- 5. 产品级促销 -->
            <template v-else-if="hasPromotionPrice(product)">
              <div class="flex flex-wrap items-baseline gap-3">
                <span class="text-[34px] font-semibold tabular-nums tracking-[-0.01em] text-primary">{{ formatPrice(getPromotionPriceAmount(product), siteCurrency) }}</span>
                <span class="text-muted-foreground line-through">{{ formatPrice(product.price_amount, siteCurrency) }}</span>
              </div>
              <p class="mt-2 text-[13px] text-destructive">{{ t('products.saveAmount') }} {{ formatPrice(getPromotionSaveAmount(product), siteCurrency) }}</p>
            </template>
            <!-- 6. 产品级原价 -->
            <div v-else class="flex flex-wrap items-baseline gap-3">
              <span class="text-[34px] font-semibold tabular-nums tracking-[-0.01em] text-foreground">{{ formatPrice(product.price_amount, siteCurrency) }}</span>
            </div>
          </div>

          <!-- 批发规则 -->
          <div v-if="selectedSkuWholesaleRules.length" class="mb-4 rounded-md bg-secondary px-4 py-3">
            <h4 class="mb-2 text-[13px] font-medium text-foreground">{{ t('products.wholesaleRulesTitle') }}</h4>
            <div class="flex flex-wrap gap-1.5">
              <span v-for="tier in selectedSkuWholesaleRules" :key="`${tier.sku_id || tier.sku_code || 'all'}-${tier.min_quantity}`" class="inline-flex items-center rounded-md border bg-card px-2 py-0.5 text-[12px] text-muted-foreground">{{ formatWholesaleTier(tier) }}</span>
            </div>
          </div>

          <!-- 活动规则 -->
          <div v-if="hasPromotionRules(product)" class="mb-4 rounded-md bg-secondary px-4 py-3">
            <h4 class="mb-2 flex items-center gap-1.5 text-[13px] font-medium text-foreground"><Tag class="h-3.5 w-3.5" /> {{ t('products.promotionRulesTitle') }}</h4>
            <ul class="grid gap-1">
              <li v-for="rule in getPromotionRules(product)" :key="rule.id" class="text-[13px] text-muted-foreground">{{ formatPromotionRule(rule) }}</li>
            </ul>
          </div>

          <!-- 规格 -->
          <div v-if="activeSkus.length" class="my-5">
            <div class="mb-2.5 text-[13px] text-muted-foreground">{{ t('productDetail.skuTitle') }}</div>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="sku in activeSkus"
                :key="sku.id"
                type="button"
                class="flex min-w-[86px] flex-col items-start gap-0.5 rounded-md border px-3.5 py-2.5 text-[13.5px] transition-colors"
                :class="[
                  normalizeSkuId(sku.id) === selectedSkuId ? 'border-primary font-medium text-primary' : 'bg-card text-foreground hover:border-hairline-strong',
                  !isSkuPurchasable(sku) ? 'cursor-not-allowed opacity-40' : '',
                ]"
                :disabled="!isSkuPurchasable(sku)"
                @click="selectedSkuId = normalizeSkuId(sku.id)"
              >
                {{ skuDisplayText(sku) }}
                <span class="text-[12px]" :class="normalizeSkuId(sku.id) === selectedSkuId ? 'text-primary' : 'text-muted-foreground'">{{ skuStockText(sku) }}</span>
              </button>
            </div>
            <p v-if="requiresSKUSelection" class="mt-2 text-[13px] text-warning">{{ t('productDetail.skuRequired') }}</p>
          </div>

          <!-- 数量 -->
          <div class="my-5">
            <div class="mb-2.5 text-[13px] text-muted-foreground">{{ t('productDetail.quantity') }}</div>
            <div class="inline-flex items-center overflow-hidden rounded-md border">
              <button type="button" class="grid h-11 w-[42px] place-items-center bg-card text-foreground transition-colors hover:bg-secondary disabled:opacity-35" :aria-label="t('productDetail.quantity')" :disabled="quantity <= quantityEffectiveMin" @click="quantity = Math.max(quantityEffectiveMin, quantity - 1)"><Minus class="h-4 w-4" /></button>
              <input inputmode="numeric" class="h-11 w-[52px] border-x bg-card text-center font-medium tabular-nums outline-none" :value="quantity" :aria-label="t('productDetail.quantity')" @change="handleQuantityInput($event)" @keydown.enter.prevent="($event.target as HTMLInputElement)?.blur()" />
              <button type="button" class="grid h-11 w-[42px] place-items-center bg-card text-foreground transition-colors hover:bg-secondary disabled:opacity-35" :aria-label="t('productDetail.quantity')" :disabled="quantityEffectiveLimit !== null && quantity >= quantityEffectiveLimit" @click="quantity = quantity + 1"><Plus class="h-4 w-4" /></button>
            </div>
          </div>

          <!-- 提示 -->
          <PageFeedback v-if="cannotPurchaseReason" class="my-3.5" level="error" :message="cannotPurchaseReason" />
          <PageFeedback v-if="purchaseWarning" class="my-3.5" level="warning" :message="purchaseWarning" />

          <!-- 操作 -->
          <div ref="purchaseActionsRef" class="mt-5 flex flex-wrap gap-3">
            <button v-if="requiresLogin" type="button" class="h-12 w-full rounded-md bg-primary text-[15px] font-medium text-primary-foreground transition-colors hover:bg-primary/90" @click="goLogin">{{ t('productDetail.loginToBuy') }}</button>
            <template v-else>
              <button type="button" class="h-12 flex-1 rounded-md bg-primary text-[15px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-40" :disabled="!canPurchase" @click="buyNow">{{ t('productDetail.buyNow') }}</button>
              <button type="button" class="inline-flex h-12 items-center gap-2 rounded-md border px-5 text-[15px] text-foreground transition-colors hover:border-hairline-strong disabled:cursor-not-allowed disabled:opacity-40" :disabled="!canPurchase" @click="addToCart"><ShoppingCart class="h-4 w-4" /> {{ t('productDetail.addToCart') }}</button>
            </template>
          </div>

          <div class="mt-4 flex items-center gap-3 rounded-md bg-secondary px-4 py-3 text-muted-foreground">
            <TicketCheck class="h-5 w-5 flex-none" />
            <span class="text-[13px]">{{ t('productDetail.deliveryReassurance') }}</span>
          </div>
        </div>
      </section>

      <!-- 描述 / 详情 -->
      <section v-if="getLocalizedText(product.description) || product.content" class="py-10">
        <div class="mb-6 border-b">
          <span class="-mb-px inline-block border-b-2 border-primary py-3 text-[15px] font-medium text-foreground">{{ t('productDetail.details') }}</span>
        </div>
        <div class="prose max-w-none dark:prose-invert prose-a:text-primary prose-img:rounded-md">
          <p v-if="getLocalizedText(product.description)">{{ getLocalizedText(product.description) }}</p>
          <div v-if="product.content" v-html="processHtmlForDisplay(getLocalizedText(product.content))"></div>
        </div>
      </section>

      <!-- 相关文章 -->
      <section v-if="relatedPosts.length" class="py-10">
        <div class="mb-6"><h2 class="text-[20px] font-semibold tracking-[-0.01em]">{{ t('productDetail.relatedPosts') }}</h2></div>
        <div class="divide-y border-t">
          <RouterLink v-for="rp in relatedPosts" :key="rp.id" class="group flex flex-wrap items-baseline gap-x-6 gap-y-1 py-4" :to="`/blog/${rp.slug}`">
            <span class="w-28 flex-none text-[13px] tabular-nums text-muted-foreground">{{ formatRelatedPostDate(rp.published_at) }}</span>
            <div class="min-w-0 flex-1">
              <h3 class="text-[15px] font-medium text-foreground transition-colors group-hover:text-primary">{{ getLocalizedText(rp.title) }}</h3>
              <p v-if="rp.summary" class="mt-1 line-clamp-1 text-[13.5px] text-muted-foreground">{{ getLocalizedText(rp.summary) }}</p>
            </div>
          </RouterLink>
        </div>
      </section>

      <div class="py-4">
        <RouterLink to="/products" class="inline-flex items-center gap-1.5 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground"><ArrowLeft class="h-4 w-4" /> {{ t('productDetail.backToProducts') }}</RouterLink>
      </div>

      <!-- 移动端固定购买条（复用 vault 组件，配色经 atlas 兼容令牌映射） -->
      <VaultProductMobileBar
        :visible="showMobileBar && !!product && !loading"
        :requires-login="requiresLogin"
        :can-purchase="canPurchase"
        :show-member-price="mobileBarShowMemberPrice"
        :member-price-display="mobileBarMemberPriceDisplay"
        :show-sku-promotion-price="mobileBarShowSkuPromotionPrice"
        :sku-promotion-price-display="mobileBarSkuPromotionPriceDisplay"
        :show-sku-price="mobileBarShowSkuPrice"
        :sku-price-display="mobileBarSkuPriceDisplay"
        :show-product-promotion-price="mobileBarShowProductPromotionPrice"
        :product-promotion-price-display="mobileBarProductPromotionPriceDisplay"
        :product-price-display="mobileBarProductPriceDisplay"
        @add-to-cart="addToCart"
        @buy-now="buyNow"
        @go-login="goLogin"
      />
    </template>

    <!-- 错误 -->
    <div v-else class="my-10 flex flex-col items-center gap-3 rounded-[10px] border border-dashed py-16 text-center text-muted-foreground">
      <AlertCircle class="h-11 w-11 opacity-50" :stroke-width="1.5" />
      <p class="text-[14px]">{{ t('productDetail.notFound') }}</p>
      <div class="mt-2 flex flex-wrap justify-center gap-2.5">
        <button type="button" class="inline-flex items-center gap-1.5 rounded-md bg-primary px-3.5 py-1.5 text-[13px] font-medium text-primary-foreground transition-colors hover:bg-primary/90" @click="loadProduct"><RotateCw class="h-3.5 w-3.5" /> {{ t('errorBoundary.retry') }}</button>
        <RouterLink to="/products" class="inline-flex items-center rounded-md border px-3.5 py-1.5 text-[13px] text-foreground transition-colors hover:border-hairline-strong">{{ t('productDetail.backToProducts') }}</RouterLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  AlertCircle, ArrowLeft, ChevronRight, Minus, Package, Plus,
  RotateCw, ShoppingCart, Tag, TicketCheck,
} from 'lucide-vue-next'
import { processHtmlForDisplay } from '../../utils/content'
import { useProductDetail } from '../../composables/useProductDetail'
import PageFeedback from '../../components/PageFeedback.vue'
import VaultProductMobileBar from '../vault/components/VaultProductMobileBar.vue'

const { t } = useI18n()

// 移动端固定购买条：IntersectionObserver 监听桌面购买区是否出屏
const purchaseActionsRef = ref<HTMLElement | null>(null)
const showMobileBar = ref(false)
let observer: IntersectionObserver | null = null

const setupMobileBarObserver = () => {
  if (observer) observer.disconnect()
  if (!purchaseActionsRef.value) return
  observer = new IntersectionObserver(
    (entries) => {
      const entry = entries[0]
      if (entry) showMobileBar.value = !entry.isIntersecting
    },
    { threshold: 0.1 },
  )
  observer.observe(purchaseActionsRef.value)
}

const {
  getLocalizedText, siteCurrency, formatPrice,
  getFulfillmentTypeLabel, getPurchaseTypeLabel, getStockStatusLabel,
  hasPromotionPrice, getPromotionPriceAmount, getPromotionSaveAmount,
  hasSkuPromotionPrice, getSkuPromotionSaveAmount,
  hasPromotionRules, getPromotionRules,
  formatPromotionRule, formatWholesaleTier, formatRelatedPostDate, normalizeSkuId,
  loading, product, relatedPosts, currentImage, selectedSkuId, quantity, purchaseWarning,
  activeSkus, selectedSku,
  selectedSkuMemberPrice, hasMemberPrice,
  hasSelectedSkuWholesalePrice, selectedSkuWholesaleFinalIsMember, selectedSkuWholesaleFinalPrice,
  selectedSkuWholesaleRules,
  selectedSkuPromotionPrice, selectedSkuPromotionFinalIsMember, selectedSkuPromotionFinalPrice,
  showSelectedSkuMemberBadge,
  isSkuPurchasable, skuDisplayText, skuStockText,
  quantityEffectiveLimit, quantityEffectiveMin, handleQuantityInput,
  requiresLogin, requiresSKUSelection, canPurchase, cannotPurchaseReason,
  categoryName, images,
  addToCart, buyNow, goLogin, loadProduct,
  mobileBarShowMemberPrice, mobileBarMemberPriceDisplay,
  mobileBarShowSkuPromotionPrice, mobileBarSkuPromotionPriceDisplay,
  mobileBarShowSkuPrice, mobileBarSkuPriceDisplay,
  mobileBarShowProductPromotionPrice, mobileBarProductPromotionPriceDisplay, mobileBarProductPriceDisplay,
} = useProductDetail({ onLoaded: () => setupMobileBarObserver() })

onUnmounted(() => {
  if (observer) {
    observer.disconnect()
    observer = null
  }
})
</script>
