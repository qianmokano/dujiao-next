<template>
  <div class="mx-auto flex min-h-[72vh] w-full max-w-[420px] flex-col justify-center px-5 py-16 sm:px-6">
    <div class="mb-6 flex items-center justify-between">
      <RouterLink to="/" class="inline-flex items-center gap-1.5 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">
        <ArrowLeft class="h-4 w-4" /> {{ t('auth.login.backHome') }}
      </RouterLink>
      <span class="text-[13px] text-muted-foreground">{{ brandSiteName }}</span>
    </div>

    <div class="rounded-[10px] border bg-card p-6 sm:p-8">
      <h1 class="text-[22px] font-semibold tracking-[-0.01em]">{{ t('auth.forgot.title') }}</h1>
      <p class="mt-1.5 text-[14px] text-muted-foreground">{{ t('auth.forgot.subtitle') }}</p>

      <div v-if="!emailVerificationEnabled" class="mt-6">
        <PageFeedback level="error">
          <p class="font-medium">{{ t('auth.forgot.disabled') }}</p>
          <RouterLink to="/auth/login" class="mt-2.5 inline-block text-[13px] text-muted-foreground underline underline-offset-2 transition-colors hover:text-foreground">{{ t('auth.forgot.backLogin') }}</RouterLink>
        </PageFeedback>
      </div>

      <form v-else class="mt-6 grid gap-4" @submit.prevent="handleReset">
        <!-- 邮箱 -->
        <div>
          <label class="mb-2 block text-[13px] font-medium text-foreground">{{ t('auth.forgot.emailLabel') }}</label>
          <Input v-model="email" type="email" required class="h-11" :placeholder="t('auth.forgot.emailPlaceholder')" />
        </div>

        <!-- 图形验证 -->
        <div v-if="sendCodeCaptchaEnabled">
          <p class="mb-2 text-[13px] font-medium text-foreground">{{ t('auth.common.captchaLabel') }}</p>
          <ImageCaptcha
            v-if="captchaProvider === 'image'"
            ref="imageCaptchaRef"
            v-model="captchaPayload"
            :disabled="sending || countdown > 0"
            @config-stale="handleCaptchaConfigStale"
          />
          <TurnstileCaptcha
            v-else-if="captchaProvider === 'turnstile'"
            ref="turnstileRef"
            v-model="turnstileToken"
            :site-key="turnstileSiteKey"
          />
        </div>

        <!-- 验证码 -->
        <div>
          <label class="mb-2 block text-[13px] font-medium text-foreground">{{ t('auth.forgot.codeLabel') }}</label>
          <div class="flex gap-2.5">
            <Input v-model="code" type="text" required class="h-11 min-w-0 flex-1" :placeholder="t('auth.forgot.codePlaceholder')" />
            <button type="button" class="h-11 shrink-0 whitespace-nowrap rounded-md border px-4 text-[13.5px] font-medium text-foreground transition-colors hover:border-hairline-strong disabled:cursor-not-allowed disabled:opacity-50" :disabled="sending || countdown > 0" @click="handleSendCode">
              {{ countdown > 0 ? t('auth.common.countdown', { seconds: countdown }) : t('auth.common.sendCode') }}
            </button>
          </div>
        </div>

        <!-- 新密码 -->
        <div>
          <label class="mb-2 block text-[13px] font-medium text-foreground">{{ t('auth.forgot.newPasswordLabel') }}</label>
          <Input v-model="newPassword" type="password" required class="h-11" :placeholder="t('auth.forgot.newPasswordPlaceholder')" />
        </div>

        <PageFeedback v-if="error" level="error" :message="error" />

        <button type="submit" class="h-11 w-full rounded-md bg-primary text-[14.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50" :disabled="userAuthStore.loading">
          {{ userAuthStore.loading ? t('auth.forgot.submitting') : t('auth.forgot.submit') }}
        </button>
      </form>
    </div>

    <div class="mt-5 text-center">
      <RouterLink to="/auth/login" class="text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('auth.forgot.backLogin') }}</RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowLeft } from 'lucide-vue-next'
import ImageCaptcha from '../../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../../components/captcha/TurnstileCaptcha.vue'
import { Input } from '@/components/ui/input'
import { useForgot } from '../../../composables/useForgot'
import PageFeedback from '../../../components/PageFeedback.vue'

const { t } = useI18n()

const {
  userAuthStore, brandSiteName, emailVerificationEnabled,
  email, code, newPassword, error, sending, countdown,
  captchaPayload, turnstileToken, imageCaptchaRef, turnstileRef,
  captchaProvider, sendCodeCaptchaEnabled, turnstileSiteKey,
  handleCaptchaConfigStale, handleSendCode, handleReset,
} = useForgot()

// imageCaptchaRef / turnstileRef 仅通过字符串模板 ref 绑定，显式标记避免 noUnusedLocals 误报。
void imageCaptchaRef
void turnstileRef
</script>
