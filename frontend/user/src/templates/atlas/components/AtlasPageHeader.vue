<template>
  <!-- 内页统一页头：标题 + 可选副标题 + 发丝分隔 -->
  <div class="border-b">
    <div class="mx-auto w-full max-w-[1120px] px-5 pb-8 pt-24 sm:px-6 sm:pt-28">
      <h1 class="text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ title }}</h1>
      <p v-if="subtitle" class="mt-2 max-w-[60ch] text-[14px] leading-relaxed text-muted-foreground">{{ subtitle }}</p>
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ title: string; subtitle?: string }>()
</script>
