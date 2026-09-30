<template>
  <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 pt-24 sm:px-6 sm:pt-28">
    <!-- 账户头部 -->
    <header class="mb-6 flex flex-wrap items-center justify-between gap-5 border-b pb-6">
      <div class="flex min-w-0 items-center gap-4">
        <div class="grid h-14 w-14 flex-none place-items-center rounded-full border text-[20px] font-semibold text-foreground">{{ displayInitial }}</div>
        <div class="min-w-0">
          <h1 class="truncate text-[24px] font-semibold tracking-[-0.01em] sm:text-[26px]">{{ userProfileStore.displayName }}</h1>
          <p class="mt-1 text-[13.5px] text-muted-foreground">{{ userProfileStore.profile?.email || t('personalCenter.subtitle') }}</p>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <span class="inline-flex items-center rounded-md px-2.5 py-1 text-[12.5px] font-medium" :class="emailVerifiedVariant === 'success' ? 'bg-success/10 text-success' : 'bg-warning/10 text-warning'">{{ emailVerifiedLabel }}</span>
        <span v-if="userProfileStore.currentLevel" class="inline-flex items-center gap-1.5 rounded-md bg-secondary px-2.5 py-1 text-[12.5px] font-medium text-foreground">
          <img v-if="isImagePath(userProfileStore.currentLevel?.icon)" :src="getImageUrl(userProfileStore.currentLevel!.icon)" class="h-3.5 w-3.5 object-contain" alt="" />
          <span v-else-if="userProfileStore.currentLevel?.icon">{{ userProfileStore.currentLevel.icon }}</span>
          {{ levelName(userProfileStore.currentLevel) }}
        </span>
      </div>
    </header>

    <div class="grid items-start gap-8 lg:grid-cols-[220px_1fr]">
      <!-- 侧栏 -->
      <aside class="min-w-0 max-[900px]:static max-[900px]:-mx-5 max-[900px]:px-5 lg:sticky lg:top-[84px]">
        <nav class="flex flex-col gap-0.5 border-t max-[900px]:flex-row max-[900px]:overflow-x-auto">
          <button
            v-for="item in visibleSectionItems"
            :key="item.key"
            type="button"
            class="flex w-full items-center gap-2.5 border-b py-3 text-left text-[14px] transition-colors max-[900px]:w-auto max-[900px]:whitespace-nowrap max-[900px]:border-b-0"
            :class="currentSection === item.key ? 'font-medium text-foreground' : 'text-muted-foreground hover:text-foreground'"
            @click="switchSection(item.key)"
          >
            <span class="h-[16px] w-[2px] flex-none rounded-full transition-colors" :class="currentSection === item.key ? 'bg-primary' : 'bg-transparent'"></span>
            <component :is="item.icon" class="h-4 w-4 flex-none" />
            <span>{{ t(item.label) }}</span>
          </button>
        </nav>
      </aside>

      <!-- 内容 -->
      <section class="grid min-w-0 gap-5">
        <PageFeedback v-if="globalAlert" :level="globalAlert.level" :message="globalAlert.message" />

        <!-- 概览 -->
        <template v-if="currentSection === 'overview'">
          <!-- 统计 -->
          <div class="grid grid-cols-2 gap-3.5 lg:grid-cols-4">
            <div class="rounded-[10px] border bg-card p-4">
              <div class="text-[12.5px] text-muted-foreground">{{ t('personalCenter.tabs.orders') }}</div>
              <div class="mt-1.5 text-[20px] font-semibold tabular-nums">{{ userProfileStore.loadingOrders ? '—' : userProfileStore.ordersTotal }}</div>
            </div>
            <div class="rounded-[10px] border bg-card p-4">
              <div class="text-[12.5px] text-muted-foreground">{{ t('personalCenter.memberLevel.currentLevel') }}</div>
              <div class="mt-1.5 flex items-center gap-1.5 truncate text-[15px] font-semibold">
                <img v-if="isImagePath(userProfileStore.currentLevel?.icon)" :src="getImageUrl(userProfileStore.currentLevel!.icon)" class="h-4 w-4 flex-none object-contain" alt="" />
                <span class="truncate">{{ levelName(userProfileStore.currentLevel) }}</span>
              </div>
            </div>
            <div class="rounded-[10px] border bg-card p-4">
              <div class="text-[12.5px] text-muted-foreground">{{ t('personalCenter.memberLevel.discountRate') }}</div>
              <div class="mt-1.5 text-[20px] font-semibold">{{ discountText }}</div>
            </div>
            <div class="rounded-[10px] border bg-card p-4">
              <div class="text-[12.5px] text-muted-foreground">{{ t('personalCenter.overview.accountLabel') }}</div>
              <div class="mt-1.5"><span class="inline-flex items-center rounded-md px-2 py-0.5 text-[12.5px] font-medium" :class="emailVerifiedVariant === 'success' ? 'bg-success/10 text-success' : 'bg-warning/10 text-warning'">{{ emailVerifiedLabel }}</span></div>
            </div>
          </div>

          <!-- 会员等级 -->
          <div v-if="userProfileStore.memberLevels.length > 0" class="rounded-[10px] border bg-card p-5 sm:p-6">
            <div class="flex flex-wrap items-center justify-between gap-4">
              <div class="flex items-center gap-3">
                <div class="grid h-10 w-10 flex-none place-items-center rounded-full bg-secondary text-[18px]">
                  <img v-if="isImagePath(userProfileStore.currentLevel?.icon)" :src="getImageUrl(userProfileStore.currentLevel!.icon)" class="h-6 w-6 object-contain" alt="" />
                  <span v-else>{{ userProfileStore.currentLevel?.icon || '★' }}</span>
                </div>
                <div>
                  <div class="text-[12.5px] text-muted-foreground">{{ t('personalCenter.memberLevel.currentLevel') }}</div>
                  <div class="mt-0.5 text-[15.5px] font-semibold">{{ levelName(userProfileStore.currentLevel) }}</div>
                </div>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center rounded-md bg-secondary px-2.5 py-1 text-[12.5px] font-medium text-foreground">
                  {{ t('personalCenter.memberLevel.discountRate') }}
                  {{ userProfileStore.currentLevel && userProfileStore.currentLevel.discount_rate < 100
                    ? t('personalCenter.memberLevel.discountOff', { n: userProfileStore.currentLevel.discount_rate / 10 })
                    : t('personalCenter.memberLevel.noDiscount') }}
                </span>
                <span v-if="!userProfileStore.nextLevel && userProfileStore.currentLevel" class="inline-flex items-center rounded-md bg-success/10 px-2.5 py-1 text-[12.5px] font-medium text-success">{{ t('personalCenter.memberLevel.highestLevel') }}</span>
              </div>
            </div>

            <div v-if="userProfileStore.nextLevel" class="mt-5 rounded-[10px] bg-secondary/50 p-4">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <div class="text-[12.5px] text-muted-foreground">{{ t('personalCenter.memberLevel.nextLevel') }}</div>
                  <div class="mt-0.5 flex items-center gap-2 text-[14px] font-medium">
                    <span>{{ levelName(userProfileStore.nextLevel) }}</span>
                    <span v-if="userProfileStore.nextLevel.discount_rate < 100" class="text-[12.5px] text-primary">{{ t('personalCenter.memberLevel.discountOff', { n: userProfileStore.nextLevel.discount_rate / 10 }) }}</span>
                  </div>
                </div>
              </div>

              <div v-if="userProfileStore.upgradeProgress" class="mt-3.5 grid gap-3">
                <div v-if="userProfileStore.upgradeProgress.rechargePercent !== null">
                  <div class="mb-1.5 flex justify-between text-[12.5px]">
                    <span class="text-muted-foreground">{{ t('personalCenter.memberLevel.rechargeProgress') }}</span>
                    <span class="tabular-nums text-muted-foreground">{{ userProfileStore.upgradeProgress.recharged.toFixed(2) }} / {{ userProfileStore.upgradeProgress.rechargeThreshold.toFixed(2) }}</span>
                  </div>
                  <div class="h-1 overflow-hidden rounded-full bg-border"><div class="h-full rounded-full bg-primary" :style="{ width: userProfileStore.upgradeProgress.rechargePercent + '%' }"></div></div>
                </div>
                <div v-if="userProfileStore.upgradeProgress.spendPercent !== null">
                  <div class="mb-1.5 flex justify-between text-[12.5px]">
                    <span class="text-muted-foreground">{{ t('personalCenter.memberLevel.spendProgress') }}</span>
                    <span class="tabular-nums text-muted-foreground">{{ userProfileStore.upgradeProgress.spent.toFixed(2) }} / {{ userProfileStore.upgradeProgress.spendThreshold.toFixed(2) }}</span>
                  </div>
                  <div class="h-1 overflow-hidden rounded-full bg-border"><div class="h-full rounded-full bg-primary" :style="{ width: userProfileStore.upgradeProgress.spendPercent + '%' }"></div></div>
                </div>
              </div>
            </div>
          </div>

          <!-- 最近订单 -->
          <div class="rounded-[10px] border bg-card p-5 sm:p-6">
            <div class="mb-4 flex items-center justify-between gap-3">
              <h3 class="text-[16px] font-semibold">{{ t('personalCenter.overview.recentOrdersTitle') }}</h3>
              <RouterLink to="/me/orders" class="text-[13px] text-muted-foreground transition-colors hover:text-foreground">{{ t('personalCenter.overview.viewAllOrders') }}</RouterLink>
            </div>
            <div v-if="userProfileStore.loadingOrders" class="grid gap-2.5">
              <div v-for="idx in 3" :key="idx" class="h-14 rounded-md bg-secondary"></div>
            </div>
            <div v-else-if="userProfileStore.recentOrders.length === 0" class="rounded-md border border-dashed p-4 text-[13px] text-muted-foreground">{{ t('personalCenter.overview.emptyOrders') }}</div>
            <div v-else class="divide-y border-t">
              <div v-for="order in userProfileStore.recentOrders" :key="order.order_no" class="flex flex-wrap items-center justify-between gap-x-6 gap-y-2.5 py-4">
                <div class="min-w-0">
                  <div class="text-[13px] tabular-nums text-muted-foreground">{{ t('orders.orderNo') }}：{{ order.order_no }}</div>
                  <div class="mt-1 flex items-baseline gap-3">
                    <span class="text-[17px] font-semibold tabular-nums">{{ formatMoney(order.total_amount, order.currency) }}</span>
                    <span class="text-[12.5px] text-muted-foreground">{{ formatDate(order.created_at) }}</span>
                  </div>
                </div>
                <div class="flex flex-wrap items-center gap-2.5">
                  <Badge :variant="statusVariant(order.status)">{{ statusLabel(order.status) }}</Badge>
                  <RouterLink :to="`/orders/${order.order_no}`" class="inline-flex h-9 items-center rounded-md border px-3.5 text-[13px] text-foreground transition-colors hover:border-hairline-strong">{{ t('orders.viewDetails') }}</RouterLink>
                  <RouterLink v-if="order.status === 'pending_payment'" :to="`/pay?order_no=${order.order_no}`" class="inline-flex h-9 items-center rounded-md bg-primary px-3.5 text-[13px] font-medium text-primary-foreground transition-colors hover:bg-primary/90">{{ t('orders.payNow') }}</RouterLink>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- 其余面板：复用 classic 组件（嵌于 atlas 令牌作用域内） -->
        <div v-else class="min-w-0">
          <ProfilePanel v-if="currentSection === 'profile'" />
          <SecurityPanel v-else-if="currentSection === 'security'" />
          <OrdersPanel v-else-if="currentSection === 'orders'" />
          <WalletPanel v-else-if="currentSection === 'wallet'" />
          <AffiliatePanel v-else-if="currentSection === 'affiliate'" />
          <div v-else-if="currentSection === 'reseller' && canAccessResellerConsole" class="rounded-[10px] border bg-card p-5 sm:p-6">
            <h2 class="text-[16px] font-semibold">{{ t('resellerConsole.title') }}</h2>
            <p class="mt-2 text-[14px] text-muted-foreground">{{ t('resellerConsole.dashboard.description') }}</p>
            <RouterLink to="/reseller" class="mt-4 inline-flex h-10 items-center rounded-md bg-primary px-4 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90">{{ t('resellerConsole.nav.dashboard') }}</RouterLink>
          </div>
          <GiftCardPanel v-else-if="currentSection === 'giftCard'" />
          <ApiPanel v-else-if="currentSection === 'api'" />
          <OrdersPanel v-else />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Badge } from '@/components/ui/badge'
import { getImageUrl } from '../../utils/image'
import ProfilePanel from '../../views/personal/ProfilePanel.vue'
import SecurityPanel from '../../views/personal/SecurityPanel.vue'
import OrdersPanel from '../../views/personal/OrdersPanel.vue'
import WalletPanel from '../../views/personal/WalletPanel.vue'
import GiftCardPanel from '../../views/personal/GiftCardPanel.vue'
import AffiliatePanel from '../../views/personal/AffiliatePanel.vue'
import ApiPanel from '../../views/personal/ApiPanel.vue'
import { usePersonalCenter, type PersonalSection } from '../../composables/usePersonalCenter'
import PageFeedback from '../../components/PageFeedback.vue'

const { t } = useI18n()

const props = withDefaults(defineProps<{ section?: PersonalSection }>(), {
  section: 'overview',
})

const {
  userProfileStore, canAccessResellerConsole, visibleSectionItems, currentSection, globalAlert,
  displayInitial, switchSection, statusLabel, statusVariant, formatMoney, formatDate,
  emailVerifiedLabel, emailVerifiedVariant, discountText, isImagePath, levelName,
} = usePersonalCenter(() => props.section)
</script>
