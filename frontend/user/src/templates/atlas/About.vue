<template>
  <div class="mx-auto w-full max-w-[760px] px-5 pb-10 sm:px-6">
    <header class="pb-2 pt-24 sm:pt-28">
      <h1 class="text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ heroTitle }}</h1>
      <p v-if="heroSubtitle" class="mt-2 text-[14px] leading-relaxed text-muted-foreground">{{ heroSubtitle }}</p>
    </header>

    <!-- 站长配置的介绍 / 服务 / 联系（仅渲染真实配置内容） -->
    <div class="mt-6 space-y-10">
      <div v-if="hasIntroduction">
        <p class="whitespace-pre-line text-[15px] leading-[1.8] text-foreground">{{ introductionText }}</p>
      </div>

      <div v-if="hasServices">
        <h2 v-if="servicesTitle" class="mb-4 text-[16px] font-semibold">{{ servicesTitle }}</h2>
        <div class="divide-y border-t">
          <div v-for="(service, index) in serviceItems" :key="`about-service-${index}`" class="flex items-start gap-3 py-3.5">
            <Check class="mt-1 h-4 w-4 flex-none text-success" :stroke-width="2" />
            <span class="text-[14.5px] leading-relaxed text-foreground">{{ service }}</span>
          </div>
        </div>
      </div>

      <div v-if="hasContact">
        <h2 v-if="contactTitle" class="mb-4 text-[16px] font-semibold">{{ contactTitle }}</h2>
        <p v-if="contactText" class="mb-5 whitespace-pre-line text-[14.5px] leading-relaxed text-muted-foreground">{{ contactText }}</p>
        <div v-if="hasContactLinks" class="flex flex-wrap gap-3">
          <a v-if="contactConfig?.telegram" :href="contactConfig.telegram" target="_blank" rel="noopener noreferrer" class="inline-flex h-11 items-center gap-2 rounded-md border bg-card px-4 text-[13.5px] text-foreground transition-colors hover:border-hairline-strong">
            <Send class="h-4 w-4" /> Telegram
          </a>
          <a v-if="contactConfig?.whatsapp" :href="contactConfig.whatsapp" target="_blank" rel="noopener noreferrer" class="inline-flex h-11 items-center gap-2 rounded-md border bg-card px-4 text-[13.5px] text-foreground transition-colors hover:border-hairline-strong">
            <MessageCircle class="h-4 w-4" /> WhatsApp
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Check, MessageCircle, Send } from 'lucide-vue-next'
import { useAbout } from '../../composables/useAbout'

const {
  contactConfig, heroTitle, heroSubtitle, introductionText, servicesTitle, contactTitle, contactText,
  serviceItems, hasIntroduction, hasServices, hasContactLinks, hasContact,
} = useAbout()
</script>
