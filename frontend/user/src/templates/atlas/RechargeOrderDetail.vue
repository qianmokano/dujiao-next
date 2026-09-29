<template>
  <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 sm:px-6">
    <div class="flex items-start justify-between gap-4 pb-2 pt-24 sm:pt-28">
      <div>
        <h1 class="mb-1.5 text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('rechargeOrder.title') }}</h1>
        <p class="text-[14px] text-muted-foreground">{{ t('rechargeOrder.subtitle') }}</p>
      </div>
      <RouterLink class="mt-2 flex-none text-[13px] text-muted-foreground transition-colors hover:text-foreground" to="/me/orders">{{ t('rechargeOrder.backList') }}</RouterLink>
    </div>

    <div class="mt-6">
      <!-- Loading -->
      <div v-if="loading" class="rounded-[10px] border bg-card p-6">
        <div class="mb-4 h-5 w-[35%] rounded bg-secondary"></div>
        <div class="h-[180px] rounded-md bg-secondary"></div>
      </div>

      <!-- 不存在 -->
      <div v-else-if="!recharge" class="my-8 flex flex-col items-center gap-3 rounded-[10px] border border-dashed py-16 text-center text-muted-foreground">
        <AlertCircle class="h-10 w-10 opacity-50" :stroke-width="1.5" />
        <p class="text-[14px]">{{ t('rechargeOrder.notFound') }}</p>
        <button type="button" class="mt-2 rounded-md border px-4 py-2 text-[13.5px] text-foreground transition-colors hover:border-hairline-strong" @click="loadDetail()">{{ t('errorBoundary.retry') }}</button>
      </div>

      <template v-else>
        <!-- 摘要 -->
        <section class="mb-5 flex flex-wrap items-start justify-between gap-5 rounded-[10px] border bg-card p-5 sm:p-6">
          <div class="min-w-0">
            <div class="text-[12px] text-muted-foreground">{{ t('personalCenter.wallet.rechargeNoLabel') }}</div>
            <div class="mt-1 font-medium tabular-nums">{{ recharge.recharge_no }}</div>
            <div class="mt-1.5 text-[13px] text-muted-foreground">{{ t('rechargeOrder.createdAtLabel') }}：{{ formatDate(recharge.created_at) }}</div>
          </div>
          <div class="text-right">
            <div class="text-[12px] text-muted-foreground">{{ t('rechargeOrder.rechargeAmount') }}</div>
            <div class="mt-1 text-[24px] font-semibold tabular-nums">{{ formatMoney(recharge.amount, recharge.currency) }}</div>
          </div>
          <Badge :variant="rechargeStatusVariant(recharge.status)">{{ rechargeStatusText(recharge.status) }}</Badge>
        </section>

        <!-- 金额与时间 -->
        <section class="mb-5 rounded-[10px] border bg-card p-5 sm:p-6">
          <h2 class="mb-4 text-[16px] font-semibold">{{ t('rechargeOrder.amountTitle') }}</h2>
          <dl class="divide-y">
            <div class="flex items-center justify-between gap-4 py-3 text-[14px]">
              <dt class="text-muted-foreground">{{ t('rechargeOrder.rechargeAmount') }}</dt>
              <dd class="font-medium tabular-nums">{{ formatMoney(recharge.amount, recharge.currency) }}</dd>
            </div>
            <div v-if="customerFeeApplied" class="flex items-center justify-between gap-4 py-3 text-[14px]">
              <dt class="text-warning">{{ t('payment.feeAmountLabel') }}</dt>
              <dd class="font-medium tabular-nums text-warning">{{ formatMoney(recharge.fee_amount, recharge.currency) }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3 text-[14px]">
              <dt class="text-muted-foreground">{{ t('personalCenter.wallet.payAmountLabel') }}</dt>
              <dd class="font-semibold tabular-nums text-primary">{{ formatMoney(recharge.payable_amount, recharge.currency) }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3 text-[14px]">
              <dt class="text-muted-foreground">{{ t('rechargeOrder.createdAtLabel') }}</dt>
              <dd class="font-medium">{{ formatDate(recharge.created_at) }}</dd>
            </div>
            <div v-if="recharge.paid_at" class="flex items-center justify-between gap-4 py-3 text-[14px]">
              <dt class="text-muted-foreground">{{ t('rechargeOrder.paidAtLabel') }}</dt>
              <dd class="font-medium">{{ formatDate(recharge.paid_at) }}</dd>
            </div>
            <div v-if="payment?.expires_at" class="flex items-center justify-between gap-4 py-3 text-[14px]">
              <dt class="text-muted-foreground">{{ t('payment.expiresAt') }}</dt>
              <dd class="font-medium">{{ formatDate(payment.expires_at) }}</dd>
            </div>
          </dl>
        </section>

        <!-- 备注 -->
        <section v-if="recharge.remark" class="mb-5 rounded-[10px] border bg-card p-5 sm:p-6">
          <h2 class="mb-2 text-[16px] font-semibold">{{ t('rechargeOrder.remarkLabel') }}</h2>
          <p class="text-[14px] leading-relaxed text-muted-foreground">{{ recharge.remark }}</p>
        </section>

        <!-- 支付区域 -->
        <section v-if="isPending" class="mb-5 rounded-[10px] border bg-card p-5 sm:p-6">
          <h2 class="mb-2 text-[16px] font-semibold">{{ t('rechargeOrder.paymentTitle') }}</h2>
          <p class="mb-5 text-[13px] text-muted-foreground">{{ t('personalCenter.wallet.pendingHint') }}</p>
          <div class="grid gap-6 md:grid-cols-2">
            <div v-if="showQRCode" class="flex flex-col items-center rounded-[10px] bg-secondary/50 p-5 text-center">
              <div class="mb-3 text-[13px] text-muted-foreground">{{ t('payment.qrTitle') }}</div>
              <div class="aspect-square w-full max-w-[220px] overflow-hidden rounded-md border bg-white p-2"><img :src="qrImageUrl" alt="Recharge QR" class="h-full w-full object-contain" /></div>
              <div v-if="qrUsingPayLinkFallback" class="mt-3 text-[12.5px] text-muted-foreground">{{ t('payment.qrFallbackHint') }}</div>
            </div>
            <div class="flex flex-col justify-center rounded-[10px] border p-5">
              <div v-if="hasCryptoPaymentDetails" class="grid gap-2">
                <div v-for="item in cryptoPaymentDetails" :key="item.key" class="flex justify-between gap-3 border-b pb-1.5 last:border-b-0 last:pb-0">
                  <span class="flex-none text-[12.5px] text-muted-foreground">{{ item.label }}</span>
                  <span class="break-all text-right text-[13px] font-medium text-foreground">{{ item.value }}<span v-if="item.detail" class="text-muted-foreground"> ({{ item.detail }})</span></span>
                </div>
                <div v-if="cryptoWalletAddress" class="flex items-center justify-end gap-2 pt-1.5">
                  <button type="button" class="rounded-md border px-3 py-1.5 text-[12.5px] text-foreground transition-colors hover:border-hairline-strong" @click="handleCopyWalletAddress">{{ t('payment.copyWalletAddress') }}</button>
                  <span v-if="walletAddressCopied" class="text-[12.5px] text-success">{{ t('payment.copied') }}</span>
                </div>
              </div>
              <div class="mt-4 flex flex-wrap gap-2.5 border-t pt-4">
                <button v-if="payLink" type="button" class="inline-flex h-10 items-center rounded-md border px-4 text-[13.5px] text-foreground transition-colors hover:border-hairline-strong" @click="handleOpenPayLink">{{ t('payment.openPayLink') }}</button>
                <button type="button" class="inline-flex h-10 items-center rounded-md bg-primary px-4 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50" :disabled="checkingPayment" @click="checkPayment">
                  {{ checkingPayment ? t('personalCenter.wallet.checkingPayStatus') : t('personalCenter.wallet.checkPayStatus') }}
                </button>
              </div>
              <div v-if="showTelegramPayHint" class="mt-3 text-[12.5px] text-muted-foreground">{{ t('payment.telegramExternalHint') }}</div>
            </div>
          </div>
        </section>

        <!-- 成功 -->
        <section v-if="recharge.status === 'success'" class="mb-5 flex items-center gap-3 rounded-[10px] border bg-card p-5">
          <CheckCircle2 class="h-6 w-6 flex-none text-success" :stroke-width="1.5" />
          <p class="font-medium text-foreground">{{ t('personalCenter.wallet.rechargeSuccess') }}</p>
        </section>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AlertCircle, CheckCircle2 } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { useRechargeOrderDetail } from '../../composables/useRechargeOrderDetail'

const { t } = useI18n()

const {
  loading, checkingPayment, recharge, payment, walletAddressCopied, qrImageUrl,
  isPending, payLink, showTelegramPayHint, qrUsingPayLinkFallback, showQRCode,
  cryptoWalletAddress, cryptoPaymentDetails, hasCryptoPaymentDetails, customerFeeApplied,
  rechargeStatusText, rechargeStatusVariant, formatMoney, formatDate,
  loadDetail, checkPayment, handleOpenPayLink, handleCopyWalletAddress,
} = useRechargeOrderDetail()
</script>
