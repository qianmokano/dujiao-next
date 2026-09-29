<template>
  <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 sm:px-6">
    <nav class="flex flex-wrap items-center gap-1.5 pb-2 pt-24 text-[13px] text-muted-foreground sm:pt-28">
      <RouterLink to="/" class="transition-colors hover:text-foreground">{{ t('nav.home') }}</RouterLink>
      <ChevronRight class="h-3.5 w-3.5 flex-none" />
      <span class="text-foreground">{{ t('guestOrders.title') }}</span>
    </nav>

    <header class="mb-6 mt-2">
      <h1 class="mb-1.5 text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('guestOrders.title') }}</h1>
      <p class="text-[14px] text-muted-foreground">{{ t('guestOrders.subtitle') }}</p>
    </header>

    <!-- 查询表单 -->
    <section class="mb-8 rounded-[10px] border bg-card p-5 sm:p-6">
      <div v-if="hasSavedAuth" class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-md bg-secondary px-3.5 py-2.5 text-[12.5px] text-muted-foreground">
        <span>{{ t('guestOrders.savedHint', { email: savedAuth.email || '-' }) }}</span>
        <button type="button" class="text-[12.5px] text-muted-foreground underline underline-offset-2 transition-colors hover:text-foreground" @click="clearSaved">{{ t('guestOrders.clearSaved') }}</button>
      </div>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-[1fr_1fr_1fr_auto]">
        <Input v-model="email" type="email" class="h-11" :placeholder="t('guestOrders.emailPlaceholder')" />
        <Input v-model="orderPassword" type="password" class="h-11" :placeholder="t('guestOrders.passwordPlaceholder')" />
        <Input v-model="orderNo" type="text" class="h-11" :placeholder="t('guestOrders.orderNoPlaceholder')" />
        <button type="button" class="inline-flex h-11 items-center justify-center gap-2 whitespace-nowrap rounded-md bg-primary px-5 text-[14px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50" :disabled="loading" @click="handleSearch">
          <Search class="h-4 w-4" />
          {{ loading ? t('guestOrders.searching') : t('guestOrders.search') }}
        </button>
      </div>
      <p class="mt-3 text-[13px] text-muted-foreground">{{ t('guestOrders.tip') }}</p>
      <div v-if="error" class="mt-3.5 rounded-md bg-destructive/10 px-3 py-2.5 text-[13px] text-destructive">{{ error }}</div>
    </section>

    <!-- 空态 -->
    <div v-if="orders.length === 0 && !loading" class="flex flex-col items-center gap-3 border-t py-16 text-center text-muted-foreground">
      <ClipboardList class="h-10 w-10 opacity-50" :stroke-width="1.5" />
      <p class="text-[14px]">{{ emptyMessage }}</p>
    </div>

    <!-- 列表：发丝分隔行 -->
    <div v-else class="border-t">
      <article
        v-for="order in orders"
        :key="order.order_no"
        class="flex flex-wrap items-center justify-between gap-x-6 gap-y-3 border-b py-5"
      >
        <div class="min-w-0">
          <div class="text-[13px] tabular-nums text-muted-foreground">{{ t('orders.orderNo') }}：{{ order.order_no }}</div>
          <div class="mt-1.5 flex items-baseline gap-3">
            <span class="text-[19px] font-semibold tabular-nums">{{ formatMoney(order.total_amount, order.currency) }}</span>
            <span class="text-[13px] text-muted-foreground">{{ formatDate(order.created_at) }}</span>
          </div>
          <div v-if="hasDiscount(order)" class="mt-1 flex flex-wrap gap-x-4 text-[12.5px] text-muted-foreground">
            <span v-if="hasDiscountAmount(order.discount_amount)">{{ t('orderDetail.couponDiscountLabel') }}：{{ formatDiscountMoney(order.discount_amount, order.currency) }}</span>
            <span v-if="hasDiscountAmount(order.promotion_discount_amount)">{{ t('orderDetail.promotionDiscountLabel') }}：{{ formatDiscountMoney(order.promotion_discount_amount, order.currency) }}</span>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2.5">
          <Badge :variant="statusVariant(order.status)">{{ statusLabel(order.status) }}</Badge>
          <RouterLink
            :to="{ name: 'guest-order-detail', params: { order_no: order.order_no } }"
            class="inline-flex h-9 items-center rounded-md border px-3.5 text-[13px] text-foreground transition-colors hover:border-hairline-strong"
          >{{ t('guestOrders.viewDetails') }}</RouterLink>
          <RouterLink
            v-if="order.status === 'pending_payment'"
            :to="`/pay?guest=1&order_no=${order.order_no}`"
            class="inline-flex h-9 items-center rounded-md bg-primary px-3.5 text-[13px] font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >{{ t('guestOrders.payNow') }}</RouterLink>
        </div>
      </article>

      <nav v-if="pagination.total_page > 1" class="mt-8 flex flex-wrap items-center justify-center gap-1.5">
        <button type="button" class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="loading || pagination.page <= 1" :aria-label="t('common.previousBanner')" @click="changePage(pagination.page - 1)"><ChevronLeft class="h-4 w-4" /></button>
        <span class="px-2.5 text-[13.5px] tabular-nums text-muted-foreground">{{ pagination.page }} / {{ pagination.total_page }}</span>
        <button type="button" class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="loading || pagination.page >= pagination.total_page" :aria-label="t('common.nextBanner')" @click="changePage(pagination.page + 1)"><ChevronRight class="h-4 w-4" /></button>
      </nav>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ChevronLeft, ChevronRight, ClipboardList, Search } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { useGuestOrders } from '../../composables/useGuestOrders'

const { t } = useI18n()

const {
  savedAuth, email, orderPassword, orderNo, loading, error, orders, pagination,
  hasSavedAuth, clearSaved, handleSearch, emptyMessage, changePage,
  statusLabel, statusVariant, formatMoney, formatDiscountMoney, hasDiscountAmount, hasDiscount, formatDate,
} = useGuestOrders()
</script>
