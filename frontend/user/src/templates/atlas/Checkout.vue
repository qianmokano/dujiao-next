<template>
  <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 sm:px-6">
    <nav class="flex flex-wrap items-center gap-1.5 pb-2 pt-24 text-[13px] text-muted-foreground sm:pt-28">
      <RouterLink to="/" class="transition-colors hover:text-foreground">{{ t('nav.home') }}</RouterLink>
      <ChevronRight class="h-3.5 w-3.5 flex-none" />
      <RouterLink v-if="!isBuyNowMode" to="/cart" class="transition-colors hover:text-foreground">{{ t('cart.title') }}</RouterLink>
      <ChevronRight v-if="!isBuyNowMode" class="h-3.5 w-3.5 flex-none" />
      <span class="text-foreground">{{ t('checkout.title') }}</span>
    </nav>

    <header class="mb-6 mt-2">
      <h1 class="mb-1.5 text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('checkout.title') }}</h1>
      <p class="text-[14px] text-muted-foreground">{{ t('checkout.subtitle') }}</p>
    </header>

    <VaultCheckoutSteps current="checkout" :skip-cart="isBuyNowMode" />

    <!-- 空 -->
    <div v-if="cartItems.length === 0" class="my-8 flex flex-col items-center gap-3 rounded-[10px] border border-dashed py-16 text-center text-muted-foreground">
      <ShoppingCart class="h-10 w-10 opacity-50" :stroke-width="1.5" />
      <p class="text-[14px]">{{ t('checkout.empty') }}</p>
      <RouterLink to="/products" class="mt-2 inline-flex items-center rounded-md bg-primary px-4 py-2 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90">{{ t('checkout.emptyAction') }}</RouterLink>
    </div>

    <div v-else class="grid items-start gap-8 lg:grid-cols-[1fr_340px]">
      <div class="grid gap-5">
        <!-- 商品确认 -->
        <section class="rounded-[10px] border bg-card p-5 sm:p-6">
          <h2 class="mb-4 text-[16px] font-semibold">{{ t('checkout.itemsTitle') }}</h2>
          <div class="grid gap-3">
            <div
              v-for="item in cartItems"
              :key="cartItemKey(item)"
              class="flex gap-3.5 rounded-md border bg-secondary/50 p-3.5"
              :class="{ 'border-warning bg-warning/10': itemStockExceeded(item) }"
            >
              <div class="relative grid h-14 w-14 flex-none place-items-center overflow-hidden rounded-md border bg-secondary">
                <img v-if="checkoutItemImage(item)" :src="checkoutItemImage(item)" :alt="getLocalizedText(item.title)" loading="lazy" class="absolute inset-0 h-full w-full object-cover" />
                <Package v-else class="h-6 w-6 text-muted-foreground/50" :stroke-width="1.5" />
              </div>
              <div class="min-w-0 flex-1">
                <RouterLink :to="`/products/${item.slug}`" class="block text-[14.5px] font-medium transition-colors hover:text-primary">{{ getLocalizedText(item.title) }}</RouterLink>
                <div class="mt-0.5 text-[12.5px] text-muted-foreground">{{ t('checkout.quantityLabel') }}：{{ item.quantity }}</div>
                <div v-if="itemSkuDisplay(item)" class="mt-0.5 text-[12.5px] text-muted-foreground">{{ t('checkout.skuLabel') }}：{{ itemSkuDisplay(item) }}</div>
                <div v-if="itemStockHint(item)" class="mt-0.5 text-[12.5px]" :class="itemStockExceeded(item) ? 'text-warning' : 'text-muted-foreground'">{{ itemStockHint(item) }}</div>
              </div>
              <div class="flex-none text-right">
                <span class="inline-flex items-baseline font-semibold tabular-nums" :class="checkoutItemHasPriceDiscount(item) ? 'text-primary' : 'text-foreground'">
                  <b class="text-[17px] font-semibold">{{ checkoutItemPriceParts(item).integer }}</b><small class="text-xs">{{ checkoutItemPriceParts(item).decimal }}</small><em class="ml-0.5 text-[11px] not-italic text-muted-foreground">{{ checkoutItemCurrency }}</em>
                </span>
                <span v-if="checkoutItemHasPriceDiscount(item)" class="mt-0.5 block text-[11.5px] text-muted-foreground line-through">
                  {{ checkoutItemOriginalPriceParts(item).integer }}{{ checkoutItemOriginalPriceParts(item).decimal }} {{ checkoutItemCurrency }}
                </span>
              </div>
            </div>
          </div>
        </section>

        <!-- 自定义表单（复用） -->
        <CheckoutManualForm
          v-if="manualFormProducts.length"
          :manual-form-products="manualFormProducts"
          v-model="manualFormData"
          :submit-attempted="submitAttempted"
          :get-manual-field-label="getManualFieldLabel"
          :get-manual-field-placeholder="getManualFieldPlaceholder"
          :manual-field-error="manualFieldError"
        />

        <!-- 优惠码 -->
        <section v-if="!isResellerTenant" class="rounded-[10px] border bg-card p-5 sm:p-6">
          <h2 class="mb-4 text-[16px] font-semibold">{{ t('checkout.couponTitle') }}</h2>
          <Input v-model="couponCode" type="text" class="h-11" :placeholder="t('checkout.couponPlaceholder')" />
        </section>

        <!-- 下单方式 -->
        <section v-if="!userAuthStore.isAuthenticated" class="rounded-[10px] border bg-card p-5 sm:p-6">
          <h2 class="mb-4 text-[16px] font-semibold">{{ t('checkout.modeTitle') }}</h2>
          <div class="mb-4 flex flex-wrap gap-2">
            <button type="button" class="rounded-md px-3.5 py-1.5 text-[13.5px] transition-colors" :class="checkoutMode === 'guest' ? 'bg-primary font-medium text-primary-foreground' : 'border text-foreground hover:border-hairline-strong'" @click="checkoutMode = 'guest'">{{ t('checkout.guestPurchase') }}</button>
            <RouterLink to="/auth/login" class="inline-flex items-center rounded-md border px-3.5 py-1.5 text-[13.5px] text-foreground transition-colors hover:border-hairline-strong">{{ t('checkout.memberPurchase') }}</RouterLink>
          </div>

          <template v-if="checkoutMode === 'guest'">
            <div class="grid gap-3 sm:grid-cols-2">
              <Input v-model="guestEmail" type="email" class="h-11" :placeholder="t('checkout.guestEmailPlaceholder')" />
              <Input v-model="guestPassword" type="password" class="h-11" :placeholder="t('checkout.guestPasswordPlaceholder')" />
            </div>

            <div v-if="guestCaptchaEnabled" class="mt-4">
              <p class="mb-2 text-[12.5px] text-muted-foreground">{{ t('auth.common.captchaLabel') }}</p>
              <ImageCaptcha
                v-if="captchaProvider === 'image'"
                ref="guestImageCaptchaRef"
                v-model="guestCaptchaPayload"
                :disabled="submitting"
                @config-stale="handleGuestCaptchaConfigStale"
              />
              <TurnstileCaptcha
                v-else-if="captchaProvider === 'turnstile'"
                ref="guestTurnstileRef"
                v-model="guestTurnstileToken"
                :site-key="guestTurnstileSiteKey"
              />
            </div>

            <div class="mt-4 rounded-md bg-secondary px-3.5 py-3 text-[13px] text-muted-foreground">
              <p class="font-medium text-foreground">{{ t('checkout.guestInstructions.title') }}</p>
              <ul class="mt-2 grid list-disc gap-0.5 pl-4">
                <li>{{ t('checkout.guestInstructions.email') }}</li>
                <li>{{ t('checkout.guestInstructions.password') }}</li>
              </ul>
            </div>
            <p v-if="guestEmail && !guestEmailValid" class="mt-2 text-[13px] text-destructive">{{ t('error.email_invalid') }}</p>
          </template>
        </section>
      </div>

      <!-- 右栏：汇总 + 支付 -->
      <aside class="sticky top-[84px] rounded-[10px] border bg-card p-5 sm:p-6">
        <h2 class="mb-2.5 text-[16px] font-semibold">{{ t('checkout.submitTitle') }}</h2>
        <p class="mb-4 rounded-md bg-secondary px-3 py-2.5 text-[12.5px] leading-relaxed text-muted-foreground">{{ t('checkout.submitHint') }}</p>

        <div class="grid gap-2.5">
          <div class="flex items-center justify-between text-[13.5px]"><span class="text-muted-foreground">{{ t('cart.itemsCount') }}</span><span class="font-medium text-foreground">{{ totalItems }}</span></div>
          <div class="flex items-center justify-between text-[13.5px]"><span class="text-muted-foreground">{{ t('checkout.previewOriginal') }}</span><span class="font-medium text-foreground">{{ formatPrice(previewOriginal, previewCurrency) }}</span></div>
          <template v-if="!isResellerTenant">
            <div class="flex items-center justify-between text-[13.5px]"><span class="text-muted-foreground">{{ t('checkout.previewCoupon') }}</span><span class="font-medium" :class="hasPositiveAmount(previewCoupon) ? 'text-primary' : 'text-foreground'">{{ formatDiscountPrice(previewCoupon, previewCurrency) }}</span></div>
            <div class="flex items-center justify-between text-[13.5px]"><span class="text-muted-foreground">{{ t('checkout.previewPromotion') }}</span><span class="font-medium" :class="hasPositiveAmount(previewPromotion) ? 'text-primary' : 'text-foreground'">{{ formatDiscountPrice(previewPromotion, previewCurrency) }}</span></div>
            <div class="flex items-center justify-between text-[13.5px]"><span class="text-muted-foreground">{{ t('checkout.previewWholesale') }}</span><span class="font-medium" :class="hasPositiveAmount(previewWholesale) ? 'text-success' : 'text-foreground'">{{ formatDiscountPrice(previewWholesale, previewCurrency) }}</span></div>
          </template>
          <div v-if="Number(previewMemberDiscount) > 0" class="flex items-center justify-between text-[13.5px]"><span class="text-muted-foreground">{{ t('checkout.previewMemberDiscount') }}</span><span class="font-medium text-warning">-{{ formatPrice(previewMemberDiscount, previewCurrency) }}</span></div>
          <div class="mt-1 flex items-center justify-between border-t pt-3 text-foreground"><span class="text-[14px]">{{ t('checkout.previewTotal') }}</span><span class="text-[22px] font-semibold tabular-nums text-primary">{{ formatPrice(previewTotal, previewCurrency) }}</span></div>
        </div>

        <div v-if="previewLoading || couponRefreshing" class="mt-3 text-[12.5px] text-muted-foreground">{{ previewStatusText }}</div>
        <div v-if="checkoutAlert" class="mt-3.5 rounded-md px-3 py-2.5 text-[13px]" :class="checkoutAlert.level === 'error' ? 'bg-destructive/10 text-destructive' : 'bg-warning/10 text-warning'">{{ checkoutAlert.message }}</div>

        <!-- 支付方式 -->
        <div class="my-5 border-t pt-4">
          <h3 class="mb-3 text-[14px] font-medium">{{ t('checkout.paymentMethod') }}</h3>

          <div v-if="showBalanceOption" class="mb-3 rounded-md border bg-secondary/50 p-3.5">
            <div class="flex items-start justify-between gap-2.5">
              <div>
                <div class="text-[12.5px] text-muted-foreground">{{ t('payment.walletBalanceLabel') }}</div>
                <div class="mt-0.5 font-medium text-foreground">{{ walletLoading ? t('common.loading') : formatPrice(walletBalance, previewCurrency) }}</div>
              </div>
              <label class="inline-flex items-center gap-1.5 text-[12.5px] text-muted-foreground">
                <input v-model="useBalance" type="checkbox" class="h-4 w-4 accent-[var(--ui-accent)]" :disabled="walletOnlyPayment" />
                <span>{{ t('payment.useBalance') }}</span>
              </label>
            </div>
            <div v-if="walletOnlyPayment" class="mt-2 text-[12.5px] text-warning">{{ t('payment.walletOnlyHint') }}</div>
            <div v-if="useBalance" class="mt-2.5 grid gap-0.5 text-[12.5px] text-muted-foreground">
              <div>{{ t('payment.walletDeductLabel') }}：{{ expectedWalletPaidDisplay }}</div>
              <div v-if="!walletOnlyPayment">{{ t('payment.onlinePayLabel') }}：{{ expectedOnlinePayDisplay }}</div>
              <div v-if="walletOnlyPayment && expectedOnlinePayCents > 0" class="text-warning">{{ t('payment.walletInsufficientHint') }}</div>
            </div>
          </div>

          <template v-if="!walletOnlyPayment">
            <div v-if="requiresOnlineChannel && paymentChannels.length > 0" class="grid gap-2.5 sm:grid-cols-2">
              <button
                v-for="channel in paymentChannels"
                :key="channel.id"
                type="button"
                class="rounded-md border bg-card p-3 text-left transition-colors"
                :class="[
                  selectedChannelId === channel.id && !isChannelDisabledForAmount(channel) ? 'border-primary bg-primary/5' : 'hover:border-hairline-strong',
                  isChannelDisabledForAmount(channel) ? 'cursor-not-allowed opacity-50' : '',
                ]"
                :disabled="isChannelDisabledForAmount(channel)"
                :title="isChannelDisabledForAmount(channel) ? channelAmountLimitHint(channel) : ''"
                @click="handleSelectChannel(channel)"
              >
                <div class="flex items-center gap-2">
                  <img v-if="channel.icon" :src="getImageUrl(channel.icon)" loading="lazy" class="h-5 w-5 flex-none rounded object-contain" />
                  <span class="truncate text-[13.5px] font-medium text-foreground">{{ channel.name }}</span>
                </div>
                <div v-if="channel.fee_policy === 'customer_surcharge'" class="mt-1.5 grid gap-0.5 text-[11.5px] text-warning">
                  <div>{{ t('payment.feeLabel') }}：{{ formatChannelFeeRate(channel) }}</div>
                  <div>{{ t('payment.fixedFeeLabel') }}：{{ formatChannelFixedFee(channel) }}</div>
                </div>
                <div v-if="isChannelDisabledForAmount(channel)" class="mt-1 text-[11.5px] text-warning">{{ channelAmountLimitHint(channel) }}</div>
              </button>
            </div>
            <div v-else-if="requiresOnlineChannel && paymentChannels.length === 0" class="text-[13px] text-muted-foreground">{{ t('checkout.noPaymentChannels') }}</div>
          </template>
          <div v-if="!requiresOnlineChannel" class="text-[13px] text-success">{{ t('checkout.walletCoversAll') }}</div>
        </div>

        <button type="button" class="h-11 w-full rounded-md bg-primary text-[14.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-40" :disabled="!canSubmit" @click="handleSubmit">
          {{ submitting ? t('checkout.submitting') : t('checkout.submitButton') }}
        </button>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ChevronRight, Package, ShoppingCart } from 'lucide-vue-next'
import { Input } from '@/components/ui/input'
import ImageCaptcha from '../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../components/captcha/TurnstileCaptcha.vue'
import CheckoutManualForm from '../../components/checkout/CheckoutManualForm.vue'
import VaultCheckoutSteps from '../vault/components/VaultCheckoutSteps.vue'
import { useCheckout } from '../../composables/useCheckout'

const { t } = useI18n()

const {
  userAuthStore, getLocalizedText, formatPrice, getImageUrl,
  isBuyNowMode, cartItems, totalItems, cartItemKey, checkoutItemImage, itemSkuDisplay,
  itemStockExceeded, itemStockHint,
  checkoutItemCurrency, checkoutItemPriceParts, checkoutItemOriginalPriceParts, checkoutItemHasPriceDiscount,
  manualFormProducts, manualFormData, submitAttempted, getManualFieldLabel, getManualFieldPlaceholder, manualFieldError,
  couponCode, isResellerTenant,
  checkoutMode, guestEmail, guestPassword, guestEmailValid,
  guestCaptchaEnabled, captchaProvider, guestCaptchaPayload, guestTurnstileToken, guestTurnstileSiteKey,
  guestImageCaptchaRef, guestTurnstileRef, handleGuestCaptchaConfigStale,
  previewCurrency, previewOriginal, previewCoupon, previewPromotion, previewWholesale, previewMemberDiscount, previewTotal,
  previewLoading, couponRefreshing, previewStatusText, hasPositiveAmount, formatDiscountPrice, checkoutAlert,
  showBalanceOption, walletLoading, walletBalance, useBalance, walletOnlyPayment,
  expectedWalletPaidDisplay, expectedOnlinePayDisplay, expectedOnlinePayCents,
  requiresOnlineChannel, paymentChannels, selectedChannelId, isChannelDisabledForAmount, channelAmountLimitHint,
  handleSelectChannel, formatChannelFeeRate, formatChannelFixedFee,
  submitting, canSubmit, handleSubmit,
} = useCheckout()

// 字符串模板 ref，逻辑在 composable 内，显式标记避免 noUnusedLocals 误报。
void guestImageCaptchaRef
void guestTurnstileRef
</script>
