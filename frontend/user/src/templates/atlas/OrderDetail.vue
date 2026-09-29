<template>
  <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 sm:px-6">
    <nav class="flex flex-wrap items-center gap-1.5 pb-2 pt-24 text-[13px] text-muted-foreground sm:pt-28">
      <RouterLink to="/" class="transition-colors hover:text-foreground">{{ t('nav.home') }}</RouterLink>
      <ChevronRight class="h-3.5 w-3.5 flex-none" />
      <RouterLink to="/me/orders" class="transition-colors hover:text-foreground">{{ t('orders.title') }}</RouterLink>
      <ChevronRight class="h-3.5 w-3.5 flex-none" />
      <span class="text-foreground">{{ t('orderDetail.title') }}</span>
    </nav>

    <header class="mb-6 mt-2">
      <h1 class="mb-1.5 text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('orderDetail.title') }}</h1>
      <p class="text-[14px] text-muted-foreground">{{ t('orderDetail.subtitle') }}</p>
    </header>

    <!-- Loading -->
    <div v-if="loading" class="rounded-[10px] border bg-card p-6">
      <div class="mb-4 h-5 w-[35%] rounded bg-secondary"></div>
      <div class="h-[200px] rounded-md bg-secondary"></div>
    </div>

    <!-- 不存在 -->
    <div v-else-if="!order" class="my-8 flex flex-col items-center gap-3 rounded-[10px] border border-dashed py-16 text-center text-muted-foreground">
      <AlertCircle class="h-10 w-10 opacity-50" :stroke-width="1.5" />
      <p class="text-[14px]">{{ t('orderDetail.notFound') }}</p>
      <button type="button" class="mt-2 rounded-md border px-4 py-2 text-[13.5px] text-foreground transition-colors hover:border-hairline-strong" @click="debouncedLoadOrder()">{{ t('errorBoundary.retry') }}</button>
    </div>

    <template v-else>
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
            :to="`/pay?order_no=${order.order_no}`"
          >{{ t('orderDetail.payNow') }}</RouterLink>
          <button
            v-if="order.status === 'pending_payment'"
            type="button"
            class="inline-flex h-9 items-center rounded-md border border-destructive/40 px-4 text-[13.5px] text-destructive transition-colors hover:bg-destructive/10"
            @click="cancelOrder"
          >{{ t('orderDetail.cancel') }}</button>
        </div>
      </section>

      <!-- 订单内容（复用 vault 订单主体，配色经 atlas 兼容令牌映射） -->
      <VaultOrderBody
        :order="order"
        variant="user"
        :fulfillment-downloading="fulfillmentDownloading"
        @download="handleDownloadFulfillment"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AlertCircle, ChevronRight } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import VaultOrderBody from '../vault/components/VaultOrderBody.vue'
import { useOrderDetail } from '../../composables/useOrderDetail'

const { t } = useI18n()

const {
  loading, order, debouncedLoadOrder, cancelOrder, fulfillmentDownloading, handleDownloadFulfillment,
  statusLabel, statusVariant, formatDate, formatMoney,
} = useOrderDetail()
</script>
