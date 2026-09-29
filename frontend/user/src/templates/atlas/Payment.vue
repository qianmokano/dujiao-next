<template>
  <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 sm:px-6">
    <div class="flex items-start justify-between gap-4 pb-2 pt-24 sm:pt-28">
      <div>
        <h1 class="mb-1.5 text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('payment.title') }}</h1>
        <p class="text-[14px] text-muted-foreground">{{ t('payment.subtitle') }}</p>
      </div>
      <RouterLink :to="backLink" class="mt-2 flex-none text-[13px] text-muted-foreground transition-colors hover:text-foreground">{{ t('payment.backToOrders') }}</RouterLink>
    </div>

    <div class="mt-4">
      <VaultCheckoutSteps current="payment" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="mb-5 rounded-[10px] border bg-card p-6">
      <div class="mb-4 h-5 w-[40%] rounded bg-secondary"></div>
      <div class="h-[180px] rounded-md bg-secondary"></div>
    </div>

    <!-- 游客验证 -->
    <div v-else-if="showGuestAuthForm" class="mb-5 rounded-[10px] border bg-card p-6">
      <h2 class="mb-1.5 text-[16px] font-semibold">{{ t('payment.guestAuthTitle') }}</h2>
      <p class="mb-4 text-[13px] text-muted-foreground">{{ t('payment.guestAuthHint') }}</p>
      <div class="grid gap-3 sm:grid-cols-2">
        <Input v-model="guestAuth.email" type="email" class="h-11" :placeholder="t('guestOrders.emailPlaceholder')" />
        <Input v-model="guestAuth.order_password" type="password" class="h-11" :placeholder="t('guestOrders.passwordPlaceholder')" />
      </div>
      <div v-if="guestAuthError" class="mt-3.5 rounded-md bg-destructive/10 px-3 py-2.5 text-[13px] text-destructive">{{ guestAuthError }}</div>
      <button type="button" class="mt-4 inline-flex items-center rounded-md bg-primary px-4 py-2 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90" @click="handleGuestAuthSubmit">{{ t('payment.guestAuthSubmit') }}</button>
    </div>

    <!-- 订单不存在 -->
    <div v-else-if="!order" class="my-8 flex flex-col items-center gap-3 rounded-[10px] border border-dashed py-16 text-center text-muted-foreground">
      <AlertCircle class="h-10 w-10 opacity-50" :stroke-width="1.5" />
      <p class="text-[14px]">{{ t('payment.orderNotFound') }}</p>
      <RouterLink :to="backLink" class="mt-2 inline-flex items-center rounded-md border px-4 py-2 text-[13.5px] text-foreground transition-colors hover:border-hairline-strong">{{ t('payment.backToOrders') }}</RouterLink>
    </div>

    <!-- 结果视图 -->
    <div v-else-if="showResultView" class="mb-5 rounded-[10px] border bg-card p-6">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 class="text-[16px] font-semibold">{{ paymentResultTitle }}</h2>
          <p class="mt-1 text-[13px] text-muted-foreground">{{ paymentGuideTip }}</p>
          <p class="mt-1.5 text-[12.5px] text-muted-foreground">{{ t('payment.methodLabel') }}：{{ resultChannelName }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="rounded-md border px-3 py-1.5 text-[13px] text-foreground transition-colors hover:border-hairline-strong disabled:opacity-40" :disabled="loading" @click="handleRefresh">{{ t('payment.refreshStatus') }}</button>
          <button type="button" class="rounded-md border px-3 py-1.5 text-[13px] text-foreground transition-colors hover:border-hairline-strong" @click="handleChangePaymentMethod">{{ t('payment.changeMethod') }}</button>
        </div>
      </div>

      <div class="mt-5 grid gap-6 lg:grid-cols-[1.4fr_1fr]">
        <div>
          <!-- QR -->
          <div v-if="showQRCode" class="flex flex-col items-center rounded-[10px] border bg-secondary/50 p-6 text-center">
            <div class="mb-3 text-[13px] text-muted-foreground">{{ paymentGuideTitle }}</div>
            <div class="aspect-square w-full max-w-[240px] overflow-hidden rounded-md border bg-white p-2"><img :src="qrImageUrl" alt="QR Code" class="h-full w-full object-contain" /></div>
            <div v-if="qrUsingPayLinkFallback" class="mt-2.5 text-[12.5px] text-muted-foreground">{{ t('payment.qrFallbackHint') }}</div>
            <div v-if="hasCryptoPaymentDetails" class="mt-4 grid w-full gap-2 rounded-md border bg-card p-3 text-left">
              <div v-for="item in cryptoPaymentDetails" :key="item.key" class="flex justify-between gap-3 border-b pb-1.5 last:border-b-0 last:pb-0">
                <span class="flex-none text-[12.5px] text-muted-foreground">{{ item.label }}</span>
                <span class="break-all text-right text-[13px] font-medium text-foreground">{{ item.value }}<span v-if="item.detail" class="text-muted-foreground"> ({{ item.detail }})</span></span>
              </div>
              <div v-if="cryptoWalletAddress" class="flex items-center justify-end gap-2 pt-1.5">
                <button type="button" class="rounded-md border px-3 py-1.5 text-[12.5px] text-foreground transition-colors hover:border-hairline-strong" @click="handleCopyWalletAddress">{{ t('payment.copyWalletAddress') }}</button>
                <span v-if="walletAddressCopied" class="text-[12.5px] text-success">{{ t('payment.copied') }}</span>
              </div>
            </div>
          </div>

          <!-- 跳转链接 -->
          <div v-else class="rounded-[10px] border bg-secondary/50 p-6">
            <div class="mb-2.5 text-[13px] text-muted-foreground">{{ t('payment.openPayLink') }}</div>
            <button type="button" class="inline-flex items-center rounded-md bg-primary px-4 py-2 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90" @click="handleOpenPayLink">{{ t('payment.openPayLink') }}</button>
            <div v-if="openedPayWindow" class="mt-2.5 text-[12.5px] text-success">{{ payLinkOpenedTip }}</div>
            <div v-if="showTelegramPayHint" class="mt-2.5 text-[12.5px] text-muted-foreground">{{ t('payment.telegramExternalHint') }}</div>
            <div class="mt-2.5 flex items-center gap-2">
              <button type="button" class="rounded-md border px-3 py-1.5 text-[12.5px] text-foreground transition-colors hover:border-hairline-strong" @click="handleCopyPayLink">{{ t('payment.copyPayLink') }}</button>
              <span v-if="copied" class="text-[12.5px] text-success">{{ t('payment.copied') }}</span>
            </div>
          </div>
        </div>

        <div class="grid content-start gap-3.5">
          <div class="rounded-md border bg-secondary/50 p-3.5">
            <div class="text-[12.5px] text-muted-foreground">{{ t('payment.orderNo') }}</div>
            <div class="mt-1 font-medium text-foreground">{{ order.order_no }}</div>
            <div class="mt-2.5 text-[12.5px] text-muted-foreground">{{ t('payment.orderStatus') }}：{{ statusLabel(order.status) }}</div>
            <div class="mt-1 text-[12.5px] text-muted-foreground">{{ t('payment.methodLabel') }}：{{ resultChannelName }}</div>
          </div>
          <PaymentAmountBreakdown
            :order="order"
            :payment-result="paymentResult"
            :customer-fee-applied="customerFeeApplied"
            :customer-fee-amount-display="customerFeeAmountDisplay"
            :payable-amount-display="payableAmountDisplay"
            :wallet-paid-display="paymentWalletPaidDisplay"
            :online-pay-display="paymentOnlinePayDisplay"
            :show-countdown="showCountdown"
            :countdown-text="countdownText"
            :polling-active="pollingActive"
            :format-money="formatMoney"
            :format-discount-money="formatDiscountMoney"
            :has-discount-amount="hasDiscountAmount"
          />
          <div v-if="paymentResult.expires_at" class="rounded-md border bg-secondary/50 p-3.5 text-[12.5px] text-muted-foreground">
            {{ t('payment.expiresAt') }}：{{ formatDate(paymentResult.expires_at) }}
          </div>
        </div>
      </div>
    </div>

    <!-- 过期 / 取消 -->
    <div v-else-if="orderExpired || orderCanceled" class="mb-5 rounded-[10px] border bg-card p-6">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 class="text-[16px] font-semibold">{{ orderCanceled ? t('payment.orderCanceled') : t('payment.orderExpired') }}</h2>
          <p class="mt-1 text-[13px] text-muted-foreground">{{ order.order_no }}</p>
        </div>
        <RouterLink :to="backLink" class="inline-flex items-center rounded-md border px-3 py-1.5 text-[13px] text-foreground transition-colors hover:border-hairline-strong">{{ t('payment.backToOrders') }}</RouterLink>
      </div>
      <div class="mt-5 grid gap-3 sm:grid-cols-3">
        <div class="rounded-md border bg-secondary/50 p-3.5"><div class="text-[12.5px] text-muted-foreground">{{ t('payment.orderNo') }}</div><div class="mt-1 font-medium text-foreground">{{ order.order_no }}</div></div>
        <div class="rounded-md border bg-secondary/50 p-3.5"><div class="text-[12.5px] text-muted-foreground">{{ t('payment.orderStatus') }}</div><div class="mt-1 font-medium text-foreground">{{ statusLabel(order.status) }}</div></div>
        <div class="rounded-md border bg-secondary/50 p-3.5"><div class="text-[12.5px] text-muted-foreground">{{ t('orderDetail.amountTotal') }}</div><div class="mt-1 font-medium tabular-nums text-foreground">{{ formatMoney(order.total_amount, order.currency) }}</div></div>
      </div>
    </div>

    <!-- 默认：订单 + 渠道 + 操作 -->
    <div v-else class="grid items-start gap-6 lg:grid-cols-[1fr_320px]">
      <div class="grid gap-5">
        <!-- 订单信息 -->
        <section class="rounded-[10px] border bg-card p-6">
          <h2 class="mb-4 text-[16px] font-semibold">{{ t('payment.orderInfo') }}</h2>
          <div class="flex flex-wrap items-start justify-between gap-5">
            <div>
              <div class="text-[12.5px] text-muted-foreground">{{ t('payment.orderNo') }}</div>
              <div class="mt-1 font-medium text-foreground">{{ order.order_no }}</div>
              <div class="mt-2 text-[12.5px] text-muted-foreground">{{ t('orderDetail.createdAtLabel') }}：{{ formatDate(order.created_at) }}</div>
            </div>
            <div class="min-w-[280px] flex-1 rounded-md border bg-secondary/50 p-4">
              <div class="text-[12.5px] text-muted-foreground">{{ t('payment.payableAmountLabel') }}</div>
              <div class="my-1 mb-3 text-[24px] font-semibold tabular-nums text-foreground">{{ payableAmountDisplay }}</div>
              <div class="grid gap-[7px] text-[12.5px]">
                <div class="flex justify-between gap-3"><span class="text-muted-foreground">{{ t('orderDetail.amountTotal') }}</span><span class="font-medium text-foreground">{{ formatMoney(order.total_amount, order.currency) }}</span></div>
                <div v-if="showBalanceOption && useBalance" class="flex justify-between gap-3"><span class="text-muted-foreground">{{ t('payment.walletDeductLabel') }}</span><span class="font-medium text-foreground">{{ expectedWalletPaidDisplay }}</span></div>
                <div v-if="showBalanceOption && useBalance" class="flex justify-between gap-3"><span class="text-muted-foreground">{{ t('payment.onlinePayLabel') }}</span><span class="font-medium text-foreground">{{ expectedOnlinePayDisplay }}</span></div>
                <div class="flex justify-between gap-3 border-t pt-2"><span class="text-muted-foreground">{{ t('payment.orderStatus') }}</span><span class="font-medium text-foreground">{{ statusLabel(order.status) }}</span></div>
              </div>
            </div>
          </div>
          <div class="mt-4 grid gap-3 sm:grid-cols-3">
            <div class="rounded-md border bg-secondary/50 p-3.5"><div class="text-[12.5px] text-muted-foreground">{{ t('orderDetail.amountOriginal') }}</div><div class="mt-1 font-medium tabular-nums text-foreground">{{ formatMoney(order.original_amount, order.currency) }}</div></div>
            <div class="rounded-md border bg-secondary/50 p-3.5"><div class="text-[12.5px] text-muted-foreground">{{ t('orderDetail.amountDiscount') }}</div><div class="mt-1 font-medium tabular-nums" :class="hasDiscountAmount(order.discount_amount) ? 'text-destructive' : 'text-foreground'">{{ formatDiscountMoney(order.discount_amount, order.currency) }}</div></div>
            <div class="rounded-md border bg-secondary/50 p-3.5"><div class="text-[12.5px] text-muted-foreground">{{ t('orderDetail.promotionDiscountLabel') }}</div><div class="mt-1 font-medium tabular-nums" :class="hasDiscountAmount(order.promotion_discount_amount) ? 'text-destructive' : 'text-foreground'">{{ formatDiscountMoney(order.promotion_discount_amount, order.currency) }}</div></div>
            <div v-if="hasDiscountAmount(order.wholesale_discount_amount)" class="rounded-md border bg-secondary/50 p-3.5"><div class="text-[12.5px] text-muted-foreground">{{ t('orderDetail.amountWholesaleDiscount') }}</div><div class="mt-1 font-medium tabular-nums text-success">{{ formatDiscountMoney(order.wholesale_discount_amount, order.currency) }}</div></div>
          </div>
          <div v-if="order.expires_at" class="mt-3 text-[13px] text-muted-foreground">{{ t('payment.expiresAt') }}：{{ formatDate(order.expires_at) }}</div>
          <div v-if="showCountdown" class="mt-3 inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-[13px]" :class="countdownExpired ? 'bg-destructive/10 text-destructive' : 'bg-secondary text-muted-foreground'">
            <span>{{ t('payment.countdownLabel') }}</span><span class="tabular-nums">{{ countdownText }}</span>
          </div>
          <div v-if="pollingActive" class="mt-2.5 text-[12.5px] text-muted-foreground">{{ t('payment.pollingHint') }}</div>
        </section>

        <!-- 商品 -->
        <section v-if="orderItems.length" class="rounded-[10px] border bg-card p-6">
          <h2 class="mb-4 text-[16px] font-semibold">{{ t('payment.itemsTitle') }}</h2>
          <div class="grid gap-3">
            <div v-for="(item, idx) in orderItems" :key="idx" class="flex flex-wrap justify-between gap-3 border-b pb-3 last:border-b-0 last:pb-0">
              <div>
                <div class="text-[14.5px] font-medium text-foreground">{{ getLocalizedText(item.title) }}</div>
                <div class="mt-0.5 text-[12.5px] text-muted-foreground">{{ t('orderDetail.quantityLabel') }}：{{ item.quantity }} · {{ t('orderDetail.itemFulfillmentLabel') }}：{{ fulfillmentTypeLabelText(item.fulfillment_type) }}</div>
                <div v-if="orderItemSkuText(item)" class="mt-0.5 text-[12.5px] text-muted-foreground">{{ t('orderDetail.itemSkuLabel') }}：{{ orderItemSkuText(item) }}</div>
              </div>
              <div class="text-[12.5px] text-muted-foreground">{{ t('orderDetail.totalPriceLabel') }}：{{ formatMoney(item.total_price, order.currency) }}</div>
            </div>
          </div>
        </section>

        <!-- 渠道选择 -->
        <section class="rounded-[10px] border bg-card p-6">
          <h2 class="mb-4 text-[16px] font-semibold">{{ t('payment.channelTitle') }}</h2>
          <div v-if="!configReady" class="text-[13px] text-muted-foreground">{{ t('common.loading') }}</div>
          <template v-else>
            <div v-if="showBalanceOption" class="mb-3 rounded-md border bg-secondary/50 p-3.5">
              <div class="flex items-start justify-between gap-2.5">
                <div>
                  <div class="text-[12.5px] text-muted-foreground">{{ t('payment.walletBalanceLabel') }}</div>
                  <div class="mt-0.5 font-medium text-foreground">{{ walletLoading ? t('common.loading') : walletBalanceDisplay }}</div>
                </div>
                <label class="inline-flex items-center gap-1.5 text-[12.5px] text-muted-foreground"><input v-model="useBalance" type="checkbox" class="h-4 w-4 accent-[var(--ui-accent)]" :disabled="walletOnlyPayment" /><span>{{ t('payment.useBalance') }}</span></label>
              </div>
              <div v-if="walletOnlyPayment" class="mt-2 text-[12.5px] text-warning">{{ t('payment.walletOnlyHint') }}</div>
              <div v-if="useBalance" class="mt-2.5 grid gap-0.5 text-[12.5px] text-muted-foreground">
                <div>{{ t('payment.walletDeductLabel') }}：{{ expectedWalletPaidDisplay }}</div>
                <div v-if="!walletOnlyPayment">{{ t('payment.onlinePayLabel') }}：{{ expectedOnlinePayDisplay }}</div>
                <div v-if="walletOnlyPayment && expectedOnlinePayCents > 0" class="text-warning">{{ t('payment.walletInsufficientHint') }}</div>
              </div>
            </div>
            <div v-if="cachedPayment" class="mb-3 grid gap-1.5 rounded-md border border-warning/40 bg-warning/10 p-3 text-[13px] text-warning">
              <div class="font-medium">{{ t('payment.cachedTitle') }}</div>
              <div>{{ t('payment.cachedHint', { channel: cachedChannelName }) }}</div>
              <div class="mt-1.5"><button type="button" class="rounded-md border bg-card px-3 py-1.5 text-[12.5px] text-foreground transition-colors hover:border-hairline-strong" @click="restoreCachedPayment">{{ t('payment.useCached') }}</button></div>
            </div>
            <PaymentChannelSelector
              v-if="!walletOnlyPayment"
              :channels="channels"
              :model-value="selectedChannelId"
              :show-balance-option="showBalanceOption"
              :is-channel-disabled-for-amount="isChannelDisabledForAmount"
              :channel-amount-limit-hint="channelAmountLimitHint"
              @update:model-value="selectedChannelId = $event"
            />
          </template>
        </section>

        <!-- 支付信息 -->
        <section v-if="paymentResult" class="rounded-[10px] border bg-card p-6">
          <h2 class="mb-4 text-[16px] font-semibold">{{ t('payment.infoTitle') }}</h2>
          <div class="grid gap-1.5 text-[13px] text-muted-foreground">
            <div>{{ t('payment.methodLabel') }}：{{ resultChannelName }}</div>
            <div>{{ t('payment.interactionLabel') }}：{{ interactionLabel }}</div>
            <div v-if="paymentResult.expires_at">{{ t('payment.expiresAt') }}：{{ formatDate(paymentResult.expires_at) }}</div>
          </div>
          <div v-if="showPayLink" class="mt-3.5 flex flex-wrap gap-2">
            <button type="button" class="rounded-md bg-primary px-3.5 py-1.5 text-[13px] font-medium text-primary-foreground transition-colors hover:bg-primary/90" @click="handleOpenPayLink">{{ t('payment.openPayLink') }}</button>
            <button type="button" class="rounded-md border px-3.5 py-1.5 text-[13px] text-foreground transition-colors hover:border-hairline-strong" @click="handleCopyPayLink">{{ t('payment.copyPayLink') }}</button>
          </div>
        </section>
      </div>

      <!-- 右栏：操作 -->
      <aside class="sticky top-[84px] rounded-[10px] border bg-card p-6">
        <h2 class="mb-4 text-[16px] font-semibold">{{ t('payment.actionTitle') }}</h2>
        <div v-if="showCountdown" class="mb-3 text-[12.5px] text-muted-foreground">{{ t('payment.countdownLabel') }}：<span class="tabular-nums">{{ countdownText }}</span></div>
        <div v-if="paymentAlert" class="mb-3 rounded-md px-3 py-2.5 text-[13px]" :class="paymentAlert.level === 'error' ? 'bg-destructive/10 text-destructive' : (paymentAlert.level === 'success' ? 'bg-success/10 text-success' : 'bg-warning/10 text-warning')">{{ paymentAlert.message }}</div>

        <div v-if="selectedChannel" class="mb-3 rounded-md bg-secondary px-3 py-2.5 text-[12.5px] text-muted-foreground">{{ t('payment.methodLabel') }}：{{ selectedChannelName }}</div>
        <div v-else-if="!requiresOnlineChannel && !orderExpired && !orderCanceled" class="mb-3 rounded-md bg-secondary px-3 py-2.5 text-[12.5px] text-muted-foreground">{{ t('payment.walletPayOnly') }}</div>
        <div v-else-if="walletOnlyPayment && expectedOnlinePayCents > 0 && !orderExpired && !orderCanceled" class="mb-3 rounded-md bg-warning/10 px-3 py-2.5 text-[12.5px] text-warning">{{ t('payment.walletInsufficientHint') }}</div>
        <div v-else-if="!walletOnlyPayment && requiresOnlineChannel && !orderExpired && !orderCanceled" class="mb-3 rounded-md bg-warning/10 px-3 py-2.5 text-[12.5px] text-warning">{{ t('payment.selectChannelError') }}</div>

        <button type="button" class="h-11 w-full rounded-md bg-primary text-[14.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-40" :disabled="!canSubmitPayment" @click="handlePayment">
          {{ submitting ? t('payment.submitting') : t('payment.submitButton') }}
        </button>
        <button type="button" class="mt-2.5 h-11 w-full rounded-md border text-[14.5px] text-foreground transition-colors hover:border-hairline-strong disabled:cursor-not-allowed disabled:opacity-40" :disabled="loading" @click="handleRefresh">{{ t('payment.refreshStatus') }}</button>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AlertCircle } from 'lucide-vue-next'
import { Input } from '@/components/ui/input'
import PaymentAmountBreakdown from '../../components/payment/PaymentAmountBreakdown.vue'
import PaymentChannelSelector from '../../components/payment/PaymentChannelSelector.vue'
import VaultCheckoutSteps from '../vault/components/VaultCheckoutSteps.vue'
import { usePayment } from '../../composables/usePayment'

const { t } = useI18n()

const {
  loading, submitting, order, paymentResult, selectedChannelId, copied, walletAddressCopied,
  openedPayWindow, cachedPayment, guestAuth, guestAuthError, walletLoading, useBalance,
  backLink, showGuestAuthForm, walletOnlyPayment, showBalanceOption, configReady, channels,
  selectedChannel, selectedChannelName, cachedChannelName, resultChannelName, interactionLabel,
  paymentResultTitle, paymentGuideTitle, paymentGuideTip, showPayLink, showTelegramPayHint, payLinkOpenedTip,
  cryptoWalletAddress, cryptoPaymentDetails, hasCryptoPaymentDetails, qrUsingPayLinkFallback, showQRCode, qrImageUrl,
  orderExpired, orderCanceled, paymentAlert, countdownExpired, countdownText, showCountdown, showResultView, pollingActive, orderItems,
  customerFeeApplied, customerFeeAmountDisplay, payableAmountDisplay, walletBalanceDisplay,
  expectedWalletPaidDisplay, expectedOnlinePayDisplay, expectedOnlinePayCents, requiresOnlineChannel,
  paymentWalletPaidDisplay, paymentOnlinePayDisplay, isChannelDisabledForAmount, channelAmountLimitHint, canSubmitPayment,
  formatDate, statusLabel, formatMoney, hasDiscountAmount, formatDiscountMoney, getLocalizedText, orderItemSkuText, fulfillmentTypeLabelText,
  handleCopyPayLink, handleCopyWalletAddress, handleOpenPayLink, restoreCachedPayment, handleChangePaymentMethod,
  handlePayment, handleGuestAuthSubmit, handleRefresh,
} = usePayment()
</script>
