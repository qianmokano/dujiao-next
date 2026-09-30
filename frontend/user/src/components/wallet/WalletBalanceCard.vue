<template>
  <div>
    <PanelHeading :title="t('personalCenter.wallet.title')" :description="t('personalCenter.wallet.subtitle')" :icon="Wallet">
      <template #actions>
        <Badge variant="accent" size="sm">{{ t('personalCenter.tabs.wallet') }}</Badge>
      </template>
    </PanelHeading>

    <PageFeedback v-if="alert" class="mb-5" :level="alert.level" :message="alert.message" />

    <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
      <StatCard :label="t('personalCenter.wallet.balanceLabel')" :value="balanceDisplay" :icon="Banknote" tone="accent" mono />
      <StatCard :label="t('personalCenter.wallet.transactionsLabel')" :value="totalTransactions" :icon="ReceiptText" tone="info" mono />
      <StatCard
        :label="t('personalCenter.wallet.currentPageLabel')"
        :value="t('orders.pageInfo', { page: currentPage, total: totalPages })"
        :icon="Layers"
        tone="neutral"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Wallet, Banknote, ReceiptText, Layers } from 'lucide-vue-next'
import type { PageAlert } from '../../utils/alerts'
import PanelHeading from '../shared/PanelHeading.vue'
import StatCard from '../shared/StatCard.vue'
import PageFeedback from '../PageFeedback.vue'
import { Badge } from '@/components/ui/badge'

defineProps<{
  alert: PageAlert | null
  balanceDisplay: string
  totalTransactions: number
  currentPage: number
  totalPages: number
}>()

const { t } = useI18n()
</script>
