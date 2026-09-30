<template>
  <div v-if="isAtlas" class="atlas-feedback" :role="feedbackRole(level)">
    <component :is="icons[level]" class="atlas-feedback-icon" :class="feedbackIconClass(level)" aria-hidden="true" />
    <div class="atlas-feedback-message"><slot>{{ message }}</slot></div>
  </div>
  <Alert v-else :variant="pageAlertVariant(level)" :class="pageAlertToneClass(level)">
    <AlertDescription><slot>{{ message }}</slot></AlertDescription>
  </Alert>
</template>

<script setup lang="ts">
import { Check, CircleX, Info, TriangleAlert } from 'lucide-vue-next'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { useFeedback } from '../composables/useFeedback'
import { pageAlertVariant, pageAlertToneClass } from '../utils/alerts'
import { feedbackIconClass, feedbackRole, type FeedbackLevel } from '../utils/feedback'

defineProps<{ level: FeedbackLevel; message?: string }>()

const { isAtlas } = useFeedback()
const icons = { success: Check, error: CircleX, warning: TriangleAlert, info: Info }
</script>
