<template>
  <span class="inline-flex shrink-0 items-center justify-center overflow-hidden">
    <img v-if="avatarURL && !failed" :src="avatarURL" alt="" class="h-full w-full object-cover" @error="failed = true" />
    <span v-else>{{ initial }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { safeIdentityURL } from '../../utils/unifiedAuth'

const props = defineProps<{ src?: string; initial: string }>()
const avatarURL = computed(() => safeIdentityURL(props.src))
const failed = ref(false)
watch(avatarURL, () => { failed.value = false })
</script>
