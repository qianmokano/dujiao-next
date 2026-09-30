<template>
  <div class="atlas-scope">
    <!-- 顶栏 -->
    <header class="atlas-topbar sticky top-0 z-50 border-b">
      <div class="mx-auto flex h-[60px] w-full max-w-[1120px] items-center gap-4 px-5 sm:gap-6 sm:px-6">
        <RouterLink class="inline-flex min-w-0 items-center gap-2 text-[16.5px] font-semibold tracking-[-0.01em] text-foreground" to="/" :title="brandName">
          <img v-if="brandLogo" :src="brandLogo" :alt="brandName" class="h-7 max-w-[110px] object-contain sm:max-w-[150px]" />
          <span v-else class="truncate">{{ brandName }}</span>
        </RouterLink>

        <nav class="flex items-center gap-5 max-[900px]:hidden">
          <template v-for="item in menuItems" :key="item.key">
            <RouterLink
              v-if="item.type === 'route'"
              :to="item.path"
              class="whitespace-nowrap text-[14.5px] text-muted-foreground transition-colors hover:text-foreground"
              active-class="!text-foreground font-medium"
            >{{ item.label }}</RouterLink>
            <a
              v-else
              :href="item.path"
              :target="item.target"
              rel="noopener noreferrer"
              class="whitespace-nowrap text-[14.5px] text-muted-foreground transition-colors hover:text-foreground"
            >{{ item.label }}</a>
          </template>
        </nav>

        <div class="ml-auto flex items-center gap-1">
          <RouterLink class="grid h-10 w-10 flex-none place-items-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" to="/products" :aria-label="t('nav.products')"><Search class="h-[18px] w-[18px]" /></RouterLink>
          <RouterLink v-if="!userAuthStore.isAuthenticated" class="grid h-10 w-10 flex-none place-items-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground max-[900px]:hidden" to="/guest/orders" :aria-label="t('navbar.guestOrders')" :title="t('navbar.guestOrders')"><ClipboardList class="h-[18px] w-[18px]" /></RouterLink>
          <button class="grid h-10 w-10 flex-none place-items-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" type="button" :aria-label="t('resellerConsole.common.toggleTheme')" @click="toggleTheme">
            <Sun v-if="theme === 'dark'" class="h-[18px] w-[18px]" />
            <Moon v-else class="h-[18px] w-[18px]" />
          </button>
          <RouterLink class="relative grid h-10 w-10 flex-none place-items-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" to="/cart" :aria-label="t('navbar.cart')">
            <ShoppingCart class="h-[18px] w-[18px]" />
            <span v-if="cartCount > 0" class="absolute -right-[2px] -top-[2px] grid h-[17px] min-w-[17px] place-items-center rounded-full bg-primary px-[4px] text-[10.5px] font-semibold text-primary-foreground">{{ cartCount }}</span>
          </RouterLink>

          <!-- 语言切换 -->
          <div class="relative max-[900px]:hidden" ref="langEl">
            <button class="grid h-10 w-10 flex-none place-items-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" type="button" :aria-label="t('navbar.selectLanguage')" @click="toggleLang">
              <Languages class="h-[18px] w-[18px]" />
            </button>
            <div v-if="langOpen" class="absolute right-0 top-[calc(100%+6px)] z-[60] flex min-w-[160px] flex-col gap-0.5 rounded-lg border bg-card p-1.5 shadow-[var(--shadow-overlay)]">
              <button v-for="lang in languages" :key="lang.code" class="flex w-full items-center justify-between gap-2.5 rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-secondary hover:text-foreground" :class="appStore.locale === lang.code ? 'text-foreground font-medium' : 'text-muted-foreground'" @click="changeLanguage(lang.code)">
                {{ lang.name }}
                <span v-if="appStore.locale === lang.code" class="h-[6px] w-[6px] rounded-full bg-primary"></span>
              </button>
            </div>
          </div>

          <!-- 登录 / 个人中心 / 退出（桌面） -->
          <template v-if="userAuthStore.isAuthenticated">
            <RouterLink class="ml-1 inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-[13.5px] font-medium text-foreground transition-colors hover:border-hairline-strong max-[900px]:hidden" to="/me"><User class="h-4 w-4" /> {{ t('navbar.personalCenter') }}</RouterLink>
            <button type="button" class="grid h-10 w-10 flex-none place-items-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-destructive max-[900px]:hidden" :aria-label="t('navbar.logout')" :title="t('navbar.logout')" @click="userAuthStore.logout()"><LogOut class="h-[18px] w-[18px]" /></button>
          </template>
          <RouterLink v-else class="ml-1 inline-flex items-center rounded-md bg-primary px-3.5 py-1.5 text-[13.5px] font-medium text-primary-foreground transition-colors hover:bg-primary/90 max-[900px]:hidden" to="/auth/login">{{ t('navbar.login') }}</RouterLink>

          <!-- 移动端：菜单 -->
          <div class="relative hidden max-[900px]:block" ref="moreEl">
            <button class="grid h-10 w-10 flex-none place-items-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" type="button" :aria-label="t('navbar.more')" @click="toggleMore">
              <Menu v-if="!moreOpen" class="pointer-events-none h-[18px] w-[18px]" />
              <X v-else class="pointer-events-none h-[18px] w-[18px]" />
            </button>
            <div v-if="moreOpen" class="absolute right-0 top-[calc(100%+6px)] z-[60] flex min-w-[180px] flex-col gap-0.5 rounded-lg border bg-card p-1.5 shadow-[var(--shadow-overlay)]">
              <template v-for="item in menuItems" :key="`m-${item.key}`">
                <RouterLink v-if="item.type === 'route'" :to="item.path" class="flex w-full items-center rounded-md px-3 py-2.5 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" @click="moreOpen = false">{{ item.label }}</RouterLink>
                <a v-else :href="item.path" :target="item.target" rel="noopener noreferrer" class="flex w-full items-center rounded-md px-3 py-2.5 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" @click="moreOpen = false">{{ item.label }}</a>
              </template>
              <RouterLink v-if="!userAuthStore.isAuthenticated" to="/guest/orders" class="flex w-full items-center rounded-md px-3 py-2.5 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" @click="moreOpen = false">{{ t('navbar.guestOrders') }}</RouterLink>
              <div class="my-1 border-t"></div>
              <RouterLink v-if="userAuthStore.isAuthenticated" to="/me" class="flex w-full items-center rounded-md px-3 py-2.5 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" @click="moreOpen = false">{{ t('navbar.personalCenter') }}</RouterLink>
              <RouterLink v-else to="/auth/login" class="flex w-full items-center rounded-md px-3 py-2.5 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" @click="moreOpen = false">{{ t('navbar.login') }}</RouterLink>
              <button v-if="userAuthStore.isAuthenticated" class="flex w-full items-center rounded-md px-3 py-2.5 text-left text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" @click="userAuthStore.logout(); moreOpen = false">{{ t('navbar.logout') }}</button>
              <div class="my-1 border-t"></div>
              <button v-for="lang in languages" :key="`ml-${lang.code}`" class="flex w-full items-center justify-between gap-2.5 rounded-md px-3 py-2.5 text-left text-sm transition-colors hover:bg-secondary hover:text-foreground" :class="appStore.locale === lang.code ? 'text-foreground font-medium' : 'text-muted-foreground'" @click="changeLanguage(lang.code); moreOpen = false">
                {{ lang.name }}
                <span v-if="appStore.locale === lang.code" class="h-[6px] w-[6px] rounded-full bg-primary"></span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </header>

    <!-- 页面内容 -->
    <main class="flex-1">
      <slot />
    </main>

    <!-- 页脚 -->
    <footer class="mt-[var(--gap-block)] border-t bg-[color:var(--bg-subtle)]">
      <div class="mx-auto grid w-full max-w-[1120px] grid-cols-2 gap-8 px-5 py-12 sm:px-6 md:grid-cols-[1.6fr_repeat(3,1fr)]">
        <div class="col-span-2 md:col-span-1">
          <RouterLink class="inline-flex items-center gap-2 text-[16.5px] font-semibold tracking-[-0.01em] text-foreground" to="/">
            <img v-if="brandLogo" :src="brandLogo" :alt="brandName" class="h-7 max-w-[150px] object-contain" />
            <span v-else>{{ brandName }}</span>
          </RouterLink>
          <p v-if="brandDescription" class="mt-3 max-w-[38ch] text-[13.5px] leading-relaxed text-muted-foreground">{{ brandDescription }}</p>
        </div>
        <div>
          <h4 class="mb-3 text-[13px] font-medium text-foreground">{{ t('atlas.footer.shop') }}</h4>
          <RouterLink v-if="!isListMode" to="/products" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('products.allCategories') }}</RouterLink>
          <RouterLink v-if="noticeEnabled" to="/notice" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.notice') }}</RouterLink>
          <RouterLink v-if="blogEnabled" to="/blog" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.blog') }}</RouterLink>
          <RouterLink to="/me" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('navbar.personalCenter') }}</RouterLink>
        </div>
        <div>
          <h4 class="mb-3 text-[13px] font-medium text-foreground">{{ t('atlas.footer.support') }}</h4>
          <RouterLink v-if="aboutEnabled" to="/about" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.about') }}</RouterLink>
          <RouterLink v-if="!userAuthStore.isAuthenticated" to="/guest/orders" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('navbar.guestOrders') }}</RouterLink>
          <a v-if="contact?.telegram" :href="contact.telegram" target="_blank" rel="noopener noreferrer" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">Telegram</a>
          <a v-if="contact?.whatsapp" :href="contact.whatsapp" target="_blank" rel="noopener noreferrer" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">WhatsApp</a>
        </div>
        <div>
          <h4 class="mb-3 text-[13px] font-medium text-foreground">{{ t('atlas.footer.legal') }}</h4>
          <RouterLink to="/terms" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('footer.terms') }}</RouterLink>
          <RouterLink to="/privacy" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ t('footer.privacy') }}</RouterLink>
          <a v-for="link in footerLinks" :key="link.name" :href="link.url || 'javascript:void(0)'" :target="link.url ? '_blank' : undefined" rel="noopener noreferrer" class="block py-1 text-[13.5px] text-muted-foreground transition-colors hover:text-foreground">{{ link.name }}</a>
        </div>
      </div>
      <div class="border-t">
        <div class="mx-auto flex w-full max-w-[1120px] flex-wrap items-center justify-between gap-2 px-5 py-5 text-[13px] text-muted-foreground sm:px-6">
          <span>© {{ year }} {{ brandName }}</span>
          <a href="https://github.com/dujiao-next" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1.5 transition-colors hover:text-foreground" aria-label="Dujiao-Next on GitHub">
            <Github class="h-[14px] w-[14px]" />
            <span>Dujiao-Next</span>
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Search, Moon, Sun, ShoppingCart, Languages, Menu, X, User, ClipboardList, LogOut, Github,
} from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { useCartStore } from '../../../stores/cart'
import { useUserAuthStore } from '../../../stores/userAuth'
import { useNavConfig } from '../../../composables/useNavConfig'
import { useTheme } from '../../../utils/theme'
import { getImageUrl } from '../../../utils/image'
import { getLocalizedText } from '../../../utils/resellerSiteConfig'
// 本地自托管字体（与 vault 相同，无 CDN）
import '@fontsource/rubik/latin-400.css'
import '@fontsource/rubik/latin-500.css'
import '@fontsource/rubik/latin-600.css'
import '@fontsource/nunito-sans/latin-400.css'
import '@fontsource/nunito-sans/latin-600.css'
import '@fontsource/nunito-sans/latin-700.css'
import '../styles/atlas.css'

const { t } = useI18n()
const appStore = useAppStore()
const cartStore = useCartStore()
const userAuthStore = useUserAuthStore()
const { theme, toggleTheme } = useTheme()

const langOpen = ref(false)
const moreOpen = ref(false)
const langEl = ref<HTMLElement | null>(null)
const moreEl = ref<HTMLElement | null>(null)

const year = new Date().getFullYear()

const brandName = computed(() => String(appStore.config?.brand?.site_name || '').trim() || 'Dujiao')
const brandLogo = computed(() => {
  const raw = String(appStore.config?.brand?.site_logo || '').trim()
  return raw ? getImageUrl(raw) : ''
})
const brandDescription = computed(() => {
  const desc = appStore.config?.brand?.site_description
  if (desc && typeof desc === 'object') {
    const val = (desc as Record<string, string>)[appStore.locale] || (desc as Record<string, string>)['zh-CN'] || ''
    return typeof val === 'string' ? val.trim() : ''
  }
  return ''
})

const { isListMode, blogEnabled, noticeEnabled, aboutEnabled, primaryNavItems } = useNavConfig()

const menuItems = primaryNavItems

/** 后台配置的自定义页脚链接 */
const footerLinks = computed(() => {
  const links = appStore.config?.footer_links
  if (!Array.isArray(links)) return []
  return links
    .map((item: { name?: unknown; url?: unknown }) => ({
      name: typeof item?.name === 'string' ? item.name.trim() : getLocalizedText(item?.name as Record<string, string>, appStore.locale),
      url: String(item?.url || '').trim(),
    }))
    .filter((item) => item.name)
})

const contact = computed(() => appStore.config?.contact as { telegram?: string; whatsapp?: string } | undefined)

const cartCount = computed(() => cartStore.totalItems)

const languages = [
  { code: 'zh-CN', name: '简体中文' },
  { code: 'zh-TW', name: '繁體中文' },
  { code: 'en-US', name: 'English' },
]

const changeLanguage = (code: string) => {
  appStore.setLocale(code)
  langOpen.value = false
}

const toggleLang = () => { langOpen.value = !langOpen.value; moreOpen.value = false }
const toggleMore = () => { moreOpen.value = !moreOpen.value; langOpen.value = false }

const onDocClick = (e: MouseEvent) => {
  const target = e.target as Node
  if (langOpen.value && langEl.value && !langEl.value.contains(target)) langOpen.value = false
  if (moreOpen.value && moreEl.value && !moreEl.value.contains(target)) moreOpen.value = false
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
  // Teleport 到 body 的浮层（Toast / ConfirmDialog / 公告弹窗 / QuickBuy）在
  // .atlas-scope 之外，靠 body 上的这个 class 拿到 atlas 中性色板，详见 styles/atlas.css
  document.body.classList.add('atlas-tokens')
})
onUnmounted(() => {
  document.removeEventListener('click', onDocClick)
  document.body.classList.remove('atlas-tokens')
})
</script>
