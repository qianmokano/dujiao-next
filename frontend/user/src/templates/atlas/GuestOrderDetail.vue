<template>
  <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 sm:px-6">
    <div class="flex items-start justify-between gap-4 pb-2 pt-24 sm:pt-28">
      <div>
        <h1 class="mb-1.5 text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('guestOrderDetail.title') }}</h1>
        <p class="text-[14px] text-muted-foreground">{{ t('guestOrderDetail.subtitle') }}</p>
      </div>
      <RouterLink class="mt-2 flex-none text-[13px] text-muted-foreground transition-colors hover:text-foreground" to="/guest/orders">{{ t('guestOrderDetail.backSearch') }}</RouterLink>
    </div>

    <div class="mt-6">
      <!-- 游客验证 -->
      <section v-if="viewState === 'auth'" class="mb-5 rounded-[10px] border bg-card p-5 sm:p-6">
        <h2 class="mb-1.5 text-[16px] font-semibold">{{ t('guestOrderDetail.authTitle') }}</h2>
        <p class="mb-4 text-[13px] text-muted-foreground">{{ t('guestOrderDetail.authHint') }}</p>
        <div class="grid gap-3 sm:grid-cols-2">
          <Input v-model="auth.email" type="email" class="h-11" :placeholder="t('guestOrders.emailPlaceholder')" />
          <Input v-model="auth.order_password" type="password" class="h-11" :placeholder="t('guestOrders.passwordPlaceholder')" />
        </div>
        <PageFeedback v-if="authError" class="mt-3.5" level="error" :message="authError" />
        <div class="mt-4 flex flex-wrap gap-2.5">
          <button type="button" class="inline-flex h-10 items-center rounded-md bg-primary px-4 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90" @click="handleAuthSubmit">{{ t('guestOrderDetail.authSubmit') }}</button>
          <button type="button" class="inline-flex h-10 items-center rounded-md px-4 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground" @click="clearAuth">{{ t('guestOrderDetail.authClear') }}</button>
        </div>
      </section>

      <!-- Loading -->
      <div v-if="viewState === 'loading'" class="rounded-[10px] border bg-card p-6">
        <div class="mb-4 h-5 w-[35%] rounded bg-secondary"></div>
        <div class="h-[200px] rounded-md bg-secondary"></div>
      </div>

      <!-- 不存在 -->
      <div v-else-if="viewState === 'empty'" class="my-8 flex flex-col items-center gap-3 rounded-[10px] border border-dashed py-16 text-center text-muted-foreground">
        <AlertCircle class="h-10 w-10 opacity-50" :stroke-width="1.5" />
        <p class="text-[14px]">{{ t('guestOrderDetail.notFound') }}</p>
      </div>

      <template v-else-if="viewState === 'detail' && order">
        <!-- 摘要 -->
        <section class="mb-5 flex flex-wrap items-start justify-between gap-5 rounded-[10px] border bg-card p-5 sm:p-6">
          <div class="min-w-0">
            <div class="text-[12px] text-muted-foreground">{{ t('orders.orderNo') }}</div>
            <div class="mt-1 font-medium tabular-nums">{{ order.order_no }}</div>
            <div class="mt-1.5 text-[13px] text-muted-foreground">{{ t('orderDetail.createdAtLabel') }}：{{ formatDate(order.created_at) }}</div>
          </div>
          <div class="text-right">
            <div class="text-[12px] text-muted-foreground">{{ t('orderDetail.amountTotal') }}</div>
            <div class="mt-1 text-[24px] font-semibold tabular-nums">{{ formatMoney(order.total_amount, order.currency) }}</div>
          </div>
          <div class="flex flex-wrap items-center gap-2.5">
            <Badge :variant="statusVariant(order.status)">{{ statusLabel(order.status) }}</Badge>
            <RouterLink
              v-if="order.status === 'pending_payment'"
              class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90"
              :to="`/pay?guest=1&order_no=${order.order_no}`"
            >{{ t('orders.payNow') }}</RouterLink>
          </div>
        </section>

        <!-- 订单内容（复用 vault 订单主体） -->
        <VaultOrderBody
          :order="order"
          variant="guest"
          :fulfillment-downloading="fulfillmentDownloading"
          @download="handleDownloadFulfillment"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AlertCircle } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import VaultOrderBody from '../vault/components/VaultOrderBody.vue'
import { useGuestOrderDetail } from '../../composables/useGuestOrderDetail'
import PageFeedback from '../../components/PageFeedback.vue'

const { t } = useI18n()

const {
  order, authError, auth, viewState, handleAuthSubmit, clearAuth,
  fulfillmentDownloading, handleDownloadFulfillment,
  statusLabel, statusVariant, formatDate, formatMoney,
} = useGuestOrderDetail()
</script>
