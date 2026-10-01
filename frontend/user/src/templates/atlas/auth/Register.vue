<template>
  <div class="mx-auto flex min-h-[72vh] w-full max-w-[440px] flex-col justify-center px-5 py-16 sm:px-6">
    <div class="mb-6 flex items-center justify-between">
      <RouterLink to="/" class="inline-flex items-center gap-1.5 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">
        <ArrowLeft class="h-4 w-4" /> {{ t('auth.login.backHome') }}
      </RouterLink>
      <span class="text-[13px] text-muted-foreground">{{ brandSiteName }}</span>
    </div>

    <div class="rounded-[10px] border bg-card p-6 sm:p-8">
      <div v-if="!registrationEnabled && !ssoOnlyMode" class="py-6 text-center">
        <p class="text-[14px] text-muted-foreground">{{ t('auth.register.registrationDisabled') }}</p>
        <RouterLink to="/auth/login" class="mt-4 inline-block text-[13.5px] text-primary underline underline-offset-2">
          {{ t('auth.register.hasAccount') }}
        </RouterLink>
      </div>

      <template v-else>
        <h1 class="text-[22px] font-semibold tracking-[-0.01em]">{{ t('auth.register.title') }}</h1>
        <p class="mt-1.5 text-[14px] text-muted-foreground">{{ t('auth.register.subtitle') }}</p>

        <form class="mt-6 grid gap-4" @submit.prevent="handleRegister">
          <!-- 邮箱 -->
          <div>
            <label class="mb-2 block text-[13px] font-medium text-foreground">{{ t('auth.register.emailLabel') }}</label>
            <div v-if="emailDomainSelectionRequired" class="grid gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(9rem,auto)]">
              <Input
                v-model="emailLocalPart"
                type="text"
                required
                autocomplete="username"
                class="h-11"
                :class="{ 'ring-2 ring-destructive/50': formValidation.hasError('email') }"
                :placeholder="t('auth.register.emailLocalPlaceholder')"
                @blur="touchRegistrationEmail"
              />
              <Select v-model="selectedEmailDomain" @update:model-value="touchRegistrationEmail">
                <SelectTrigger class="h-11 w-full" :class="{ 'ring-2 ring-destructive/50': formValidation.hasError('email') }">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="domain in allowedEmailDomains" :key="domain" :value="domain">@{{ domain }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <Input
              v-else
              v-model="email"
              type="email"
              required
              class="h-11"
              :class="{ 'ring-2 ring-destructive/50': formValidation.hasError('email') }"
              :placeholder="t('auth.register.emailPlaceholder')"
              @blur="touchRegistrationEmail"
            />
            <p v-if="formValidation.hasError('email')" class="mt-1.5 text-[12.5px] text-destructive">{{ formValidation.getError('email') }}</p>
            <p v-else-if="emailDomainSelectionRequired" class="mt-1.5 text-[12.5px] text-muted-foreground">{{ t('auth.register.emailDomainSelectHint') }}</p>
            <p v-else-if="emailDomainAllowlistEnabled" class="mt-1.5 text-[12.5px] text-muted-foreground">
              {{ allowedEmailDomains.length > 0
                ? t('auth.register.allowedEmailDomainsHint', { domains: allowedEmailDomainsText })
                : t('auth.register.noAllowedEmailDomainsHint') }}
            </p>
          </div>

          <!-- 密码 -->
          <div>
            <label class="mb-2 block text-[13px] font-medium text-foreground">{{ t('auth.register.passwordLabel') }}</label>
            <div class="relative">
              <Input
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                required
                class="h-11 pr-10"
                :class="{ 'ring-2 ring-destructive/50': formValidation.hasError('password') }"
                :placeholder="t('auth.register.passwordPlaceholder')"
                @blur="formValidation.touchField('password', password)"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                :aria-label="showPassword ? t('auth.common.hidePassword') : t('auth.common.showPassword')"
                @click="showPassword = !showPassword"
              >
                <EyeOff v-if="showPassword" class="h-4 w-4" /><Eye v-else class="h-4 w-4" />
              </button>
            </div>
            <p v-if="formValidation.hasError('password')" class="mt-1.5 text-[12.5px] text-destructive">{{ formValidation.getError('password') }}</p>
            <div v-if="password && !formValidation.hasError('password')" class="mt-2.5 flex items-center gap-2.5">
              <div class="flex flex-1 gap-1.5">
                <div class="h-[3px] flex-1 rounded-full transition-colors" :class="passwordStrength === 'weak' ? 'bg-destructive' : passwordStrength === 'medium' ? 'bg-warning' : 'bg-success'"></div>
                <div class="h-[3px] flex-1 rounded-full transition-colors" :class="passwordStrength === 'medium' ? 'bg-warning' : passwordStrength === 'strong' ? 'bg-success' : 'bg-border'"></div>
                <div class="h-[3px] flex-1 rounded-full transition-colors" :class="passwordStrength === 'strong' ? 'bg-success' : 'bg-border'"></div>
              </div>
              <span class="text-[12px] font-medium" :class="passwordStrength === 'weak' ? 'text-destructive' : passwordStrength === 'medium' ? 'text-warning' : 'text-success'">
                {{ t(`formValidation.passwordStrength.${passwordStrength}`) }}
              </span>
            </div>
          </div>

          <!-- 图形验证 -->
          <div v-if="emailVerificationEnabled && sendCodeCaptchaEnabled && !ssoOnlyMode">
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

          <!-- 邮箱验证码 -->
          <div v-if="emailVerificationEnabled">
            <label class="mb-2 block text-[13px] font-medium text-foreground">{{ t('auth.register.codeLabel') }}</label>
            <div class="flex gap-2.5">
              <Input
                v-model="code"
                type="text"
                required
                class="h-11 min-w-0 flex-1"
                :placeholder="t('auth.register.codePlaceholder')"
              />
              <button type="button" class="h-11 shrink-0 whitespace-nowrap rounded-md border px-4 text-[13.5px] font-medium text-foreground transition-colors hover:border-hairline-strong disabled:cursor-not-allowed disabled:opacity-50" :disabled="sending || countdown > 0" @click="handleSendCode">
                {{ countdown > 0 ? t('auth.common.countdown', { seconds: countdown }) : t('auth.common.sendCode') }}
              </button>
            </div>
          </div>

          <!-- 协议 -->
          <label class="flex items-start gap-3 rounded-md bg-secondary px-3.5 py-3 text-[13px] leading-6 text-muted-foreground">
            <input v-model="agreed" type="checkbox" class="mt-1 h-4 w-4 flex-none accent-[var(--ui-accent)]" />
            <span>
              {{ t('auth.register.agreementPrefix') }}
              <RouterLink to="/privacy" target="_blank" rel="noopener noreferrer" class="text-primary underline underline-offset-2">{{ t('footer.privacy') }}</RouterLink>
              {{ t('auth.register.agreementAnd') }}
              <RouterLink to="/terms" target="_blank" rel="noopener noreferrer" class="text-primary underline underline-offset-2">{{ t('footer.terms') }}</RouterLink>
            </span>
          </label>

          <PageFeedback v-if="error" level="error" :message="error" />

          <button type="submit" class="h-11 w-full rounded-md bg-primary text-[14.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50" :disabled="userAuthStore.loading || !agreed">
            {{ userAuthStore.loading ? t('auth.register.creating') : t('auth.register.create') }}
          </button>
        </form>
      </template>
    </div>

    <div class="mt-5 text-center">
      <RouterLink to="/auth/login" class="text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('auth.register.hasAccount') }}</RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Eye, EyeOff } from 'lucide-vue-next'
import ImageCaptcha from '../../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../../components/captcha/TurnstileCaptcha.vue'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useRegister } from '../../../composables/useRegister'
import PageFeedback from '../../../components/PageFeedback.vue'

const { t } = useI18n()

const {
  userAuthStore, brandSiteName,
  email, emailLocalPart, selectedEmailDomain, password, showPassword, code, agreed,
  passwordStrength, error, sending, countdown,
  captchaPayload, turnstileToken, imageCaptchaRef, turnstileRef,
  captchaProvider, sendCodeCaptchaEnabled, turnstileSiteKey,
  ssoOnlyMode,
  registrationEnabled, emailVerificationEnabled,
  emailDomainAllowlistEnabled, allowedEmailDomains, allowedEmailDomainsText, emailDomainSelectionRequired,
  touchRegistrationEmail, formValidation, handleCaptchaConfigStale, handleSendCode, handleRegister,
} = useRegister()

// imageCaptchaRef / turnstileRef 仅通过字符串模板 ref 绑定，显式标记避免 noUnusedLocals 误报。
void imageCaptchaRef
void turnstileRef
</script>
