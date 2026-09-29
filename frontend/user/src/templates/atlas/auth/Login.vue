<template>
  <div class="mx-auto flex min-h-[72vh] w-full max-w-[420px] flex-col justify-center px-5 py-16 sm:px-6">
    <div class="mb-6 flex items-center justify-between">
      <RouterLink to="/" class="inline-flex items-center gap-1.5 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">
        <ArrowLeft class="h-4 w-4" /> {{ t('auth.login.backHome') }}
      </RouterLink>
      <span class="text-[13px] text-muted-foreground">{{ brandSiteName }}</span>
    </div>

    <div class="rounded-[10px] border bg-card p-6 sm:p-8">
      <h1 class="text-[22px] font-semibold tracking-[-0.01em]">{{ step === 'totp' ? t('auth.login.totp.title') : t('auth.login.title') }}</h1>
      <p class="mt-1.5 text-[14px] text-muted-foreground">{{ step === 'totp' ? t('auth.login.totp.subtitle') : t('auth.login.subtitle') }}</p>

      <!-- 2FA -->
      <form v-if="step === 'totp'" class="mt-6 grid gap-4" @submit.prevent="handleVerify2FA">
        <div class="rounded-md bg-secondary px-3.5 py-2.5 text-center text-[12.5px] text-muted-foreground">
          {{ t('auth.login.totp.countdown', { seconds: challengeRemainingSeconds }) }}
        </div>

        <FormField v-if="totpMode === 'code'" :label="t('auth.login.totp.codeLabel')">
          <template #default="{ id }">
            <Input :id="id" v-model="totpCode" inputmode="numeric" autocomplete="one-time-code" maxlength="6" class="h-11 text-center tracking-[0.4em]" :placeholder="t('auth.login.totp.codePlaceholder')" />
          </template>
        </FormField>
        <FormField v-else :label="t('auth.login.totp.recoveryLabel')">
          <template #default="{ id }">
            <Input :id="id" v-model="recoveryCode" autocomplete="off" class="h-11" :placeholder="t('auth.login.totp.recoveryPlaceholder')" />
          </template>
        </FormField>

        <div class="text-center">
          <button type="button" class="text-[12.5px] text-muted-foreground underline underline-offset-2 transition-colors hover:text-foreground" @click="totpMode = totpMode === 'code' ? 'recovery' : 'code'">
            {{ totpMode === 'code' ? t('auth.login.totp.useRecovery') : t('auth.login.totp.useCode') }}
          </button>
        </div>

        <div v-if="error" class="rounded-md bg-destructive/10 px-3.5 py-2.5 text-[13px] text-destructive">{{ error }}</div>

        <button type="submit" class="h-11 w-full rounded-md bg-primary text-[14.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50" :disabled="userAuthStore.loading">
          {{ userAuthStore.loading ? t('auth.login.totp.verifying') : t('auth.login.totp.submit') }}
        </button>

        <div class="text-center">
          <button type="button" class="text-[12.5px] text-muted-foreground underline underline-offset-2 transition-colors hover:text-foreground" @click="cancel2FA">
            {{ t('auth.login.totp.cancel') }}
          </button>
        </div>
      </form>

      <!-- 密码登录 -->
      <form v-else class="mt-6 grid gap-4" @submit.prevent="handleLogin">
        <FormField :label="t('auth.login.emailLabel')" :error="formValidation.getError('email')">
          <template #default="{ id, hasError, describedBy }">
            <Input
              :id="id"
              v-model="email"
              type="email"
              required
              class="h-11"
              :class="{ 'ring-2 ring-destructive/50': hasError }"
              :aria-invalid="hasError"
              :aria-describedby="describedBy"
              :placeholder="t('auth.login.emailPlaceholder')"
              @blur="formValidation.touchField('email', email)"
            />
          </template>
        </FormField>

        <FormField :label="t('auth.login.passwordLabel')" :error="formValidation.getError('password')">
          <template #default="{ id, hasError, describedBy }">
            <div class="relative">
              <Input
                :id="id"
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                required
                class="h-11 pr-10"
                :class="{ 'ring-2 ring-destructive/50': hasError }"
                :aria-invalid="hasError"
                :aria-describedby="describedBy"
                :placeholder="t('auth.login.passwordPlaceholder')"
                @blur="formValidation.touchField('password', password)"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                :aria-label="showPassword ? t('auth.common.hidePassword') : t('auth.common.showPassword')"
                @click="showPassword = !showPassword"
              >
                <EyeOff v-if="showPassword" class="h-4 w-4" aria-hidden="true" /><Eye v-else class="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
          </template>
        </FormField>

        <div v-if="loginCaptchaEnabled">
          <p class="mb-2 text-[13px] font-medium text-foreground">{{ t('auth.common.captchaLabel') }}</p>
          <ImageCaptcha
            v-if="captchaProvider === 'image'"
            ref="imageCaptchaRef"
            v-model="captchaPayload"
            :disabled="userAuthStore.loading"
            @config-stale="handleCaptchaConfigStale"
          />
          <TurnstileCaptcha
            v-else-if="captchaProvider === 'turnstile'"
            ref="turnstileRef"
            v-model="turnstileToken"
            :site-key="turnstileSiteKey"
          />
        </div>

        <div class="flex flex-wrap items-center justify-between gap-2 text-[13px] text-muted-foreground">
          <label class="inline-flex items-center gap-2">
            <input v-model="rememberMe" type="checkbox" class="h-4 w-4 accent-[var(--ui-accent)]" />
            {{ t('auth.login.rememberMe') }}
          </label>
          <RouterLink v-if="emailVerificationEnabled" to="/auth/forgot" class="text-muted-foreground underline underline-offset-2 transition-colors hover:text-foreground">
            {{ t('auth.login.forgot') }}
          </RouterLink>
        </div>

        <div v-if="info" class="rounded-md bg-success/10 px-3.5 py-2.5 text-[13px] text-success">{{ info }}</div>
        <div v-if="error" class="rounded-md bg-destructive/10 px-3.5 py-2.5 text-[13px] text-destructive">{{ error }}</div>

        <button type="submit" class="h-11 w-full rounded-md bg-primary text-[14.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50" :disabled="userAuthStore.loading">
          {{ userAuthStore.loading ? t('auth.login.submitting') : t('auth.login.submit') }}
        </button>

        <!-- 第三方登录 -->
        <div v-if="showThirdPartyLogin" class="grid gap-4 pt-1">
          <div class="flex items-center gap-3 text-[12px] text-muted-foreground">
            <span class="h-px flex-1 bg-border"></span><span>{{ t('auth.login.socialOr') }}</span><span class="h-px flex-1 bg-border"></span>
          </div>
          <div class="grid gap-3">
            <div v-if="showTelegramWidget" class="grid gap-2">
              <div ref="telegramWidgetRef" class="flex justify-center"></div>
              <p class="text-center text-[12.5px] text-muted-foreground">{{ t('auth.login.telegramHint') }}</p>
            </div>
            <div v-else-if="showTelegramOidc" class="grid gap-2">
              <button type="button" class="h-11 w-full rounded-md border text-[14px] font-medium text-foreground transition-colors hover:border-hairline-strong" @click="startTelegramOidc">{{ t('auth.login.telegramOidcButton') }}</button>
              <p class="text-center text-[12.5px] text-muted-foreground">{{ t('auth.login.telegramOidcHint') }}</p>
            </div>
            <div v-else-if="showMiniAppLoginHint" class="grid gap-2">
              <p class="text-center text-[12.5px] text-muted-foreground">{{ attemptingMiniAppLogin ? t('auth.login.telegramMiniAppLoggingIn') : t('auth.login.telegramMiniAppHint') }}</p>
            </div>
            <div v-if="showGoogleLogin" class="grid gap-2">
              <GoogleIdentityButton
                :client-id="googleClientID"
                :locale="googleButtonLocale"
                shape="rectangular"
                :ux-mode="googleIdentityUXMode"
                :login-uri="googleRedirectLoginURI"
                :prepare-redirect="prepareGoogleRedirectLogin"
                :disabled="userAuthStore.loading"
                :loading-label="t('auth.login.googleLoading')"
                @credential="handleGoogleCredential"
                @error="handleGoogleScriptError"
              />
              <p class="text-center text-[12.5px] text-muted-foreground">{{ t('auth.login.googleHint') }}</p>
            </div>
          </div>
        </div>
        <div v-if="showTelegramMiniAppEntry" class="grid gap-2 pt-1">
          <p class="text-center text-[12.5px] text-muted-foreground">{{ t('auth.login.telegramMiniAppEntryHint') }}</p>
          <button type="button" class="h-11 w-full rounded-md border text-[14px] font-medium text-foreground transition-colors hover:border-hairline-strong" @click="openTelegramMiniAppEntry">{{ t('auth.login.telegramMiniAppEntryAction') }}</button>
        </div>
      </form>
    </div>

    <div v-if="registrationEnabled" class="mt-5 text-center">
      <RouterLink to="/auth/register" class="text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('auth.login.noAccount') }}</RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Eye, EyeOff } from 'lucide-vue-next'
import ImageCaptcha from '../../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../../components/captcha/TurnstileCaptcha.vue'
import FormField from '../../../components/FormField.vue'
import GoogleIdentityButton from '../../../components/auth/GoogleIdentityButton.vue'
import { Input } from '@/components/ui/input'
import { useLogin } from '../../../composables/useLogin'

const { t } = useI18n()

const {
  userAuthStore, brandSiteName,
  email, password, showPassword, rememberMe,
  step, totpMode, totpCode, recoveryCode, challengeRemainingSeconds, handleVerify2FA, cancel2FA,
  error, info, formValidation,
  loginCaptchaEnabled, captchaProvider, captchaPayload, turnstileToken, turnstileSiteKey,
  imageCaptchaRef, turnstileRef, handleCaptchaConfigStale,
  registrationEnabled, emailVerificationEnabled,
  showTelegramWidget, telegramWidgetRef, showTelegramOidc, startTelegramOidc,
  showMiniAppLoginHint, attemptingMiniAppLogin, showTelegramMiniAppEntry, openTelegramMiniAppEntry,
  googleClientID, googleButtonLocale, googleIdentityUXMode, googleRedirectLoginURI,
  prepareGoogleRedirectLogin, showGoogleLogin, showThirdPartyLogin,
  handleGoogleCredential, handleGoogleScriptError,
  handleLogin,
} = useLogin()

// 以下引用仅通过模板字符串 ref 绑定（逻辑在 composable 内），显式标记避免 noUnusedLocals 误报。
void imageCaptchaRef
void turnstileRef
void telegramWidgetRef
</script>
