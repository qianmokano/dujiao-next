<template>
  <section
    class="mx-auto w-full max-w-[1120px] px-5 pt-6 sm:px-6 sm:pt-8"
    role="region"
    :aria-label="t('atlas.hero.bannerLabel')"
    :aria-roledescription="banners.length > 1 ? t('atlas.hero.carousel') : undefined"
    @mouseenter="carousel.setPaused('hover', true)"
    @mouseleave="carousel.setPaused('hover', false)"
    @focusin="carousel.setPaused('focus', true)"
    @focusout="onFocusOut"
  >
    <div class="atlas-banner-stage relative overflow-hidden rounded-[10px]" :aria-busy="loading">
      <div v-if="loading" class="atlas-banner-slide flex flex-col justify-center bg-secondary/40">
        <div class="h-9 w-2/3 max-w-[420px] rounded-md bg-secondary"></div>
        <div class="mt-4 h-4 w-1/2 max-w-[340px] rounded bg-secondary"></div>
        <div class="mt-6 h-11 w-32 rounded-full bg-secondary"></div>
      </div>
      <Transition v-else name="atlas-banner-fade" :css="!carouselState.reducedMotion">
        <div
          :key="banner?.id ?? 'fallback'"
          class="atlas-banner-slide flex flex-col justify-center"
          :class="imageUrl ? 'atlas-banner-with-image' : 'bg-background text-foreground'"
          :aria-live="carouselState.paused ? 'polite' : 'off'"
          aria-atomic="true"
        >
          <img
            v-if="imageUrl"
            :src="imageUrl"
            alt=""
            width="1920"
            height="720"
            fetchpriority="high"
            class="pointer-events-none absolute inset-0 h-full w-full object-cover"
            @error="onImageError"
          />
          <div class="atlas-banner-copy relative">
            <h1 class="line-clamp-2 break-words font-bold leading-[1.15] tracking-[-0.025em]">{{ title }}</h1>
            <p class="mt-3 line-clamp-2 break-words text-[15px] leading-relaxed" :class="imageUrl ? 'text-[#62656d]' : 'text-muted-foreground'">{{ subtitle }}</p>
            <button
              type="button"
              class="mt-6 inline-flex min-h-11 items-center justify-center gap-3 rounded-full px-6 py-2.5 text-[14.5px] font-semibold transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 motion-reduce:transition-none"
              :class="imageUrl ? 'bg-[#111111] text-white hover:bg-[#303030] focus-visible:outline-black' : 'bg-primary text-primary-foreground hover:bg-primary/90'"
              @click="onSubscribe"
            >
              {{ banner ? t('atlas.hero.subscribe') : t('atlas.hero.cta') }}
              <ChevronRight class="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        </div>
      </Transition>
    </div>

    <div v-if="banners.length > 1" class="mt-3 flex flex-wrap justify-center" role="group" :aria-label="t('atlas.hero.slides')">
      <button
        v-for="(item, index) in banners"
        :key="item.id"
        type="button"
        class="atlas-banner-control"
        :aria-label="t('common.switchBanner', { n: index + 1 })"
        :aria-current="index === carouselState.index ? 'true' : undefined"
        @click="carousel.select(index)"
      >
        <span class="h-1.5 rounded-full" :class="index === carouselState.index ? 'w-5 bg-foreground' : 'w-1.5 bg-muted-foreground/40'" aria-hidden="true"></span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ChevronRight } from 'lucide-vue-next'
import { bannerAPI } from '../../../api'
import { useLocalized } from '../../../composables/useProduct'
import { useAppStore } from '../../../stores/app'
import { getImageUrl } from '../../../utils/image'
import { createAtlasBannerCarousel, observeAtlasBannerEnvironment, resolveAtlasBannerTarget, selectAtlasBanners, type AtlasBanner } from '../../../utils/atlasBanner'

const router = useRouter()
const { t } = useI18n()
const { getLocalizedText } = useLocalized()
const appStore = useAppStore()
const loading = ref(true)
const banners = ref<AtlasBanner[]>([])
const desktop = ref(false)
const failedImages = ref(new Set<string>())
const carousel = createAtlasBannerCarousel(state => { carouselState.value = state })
const carouselState = ref(carousel.state)
const banner = computed(() => banners.value[carouselState.value.index] || null)
const title = computed(() => getLocalizedText(banner.value?.title) || String(appStore.config?.brand?.site_name || '').trim() || 'Dujiao')
const subtitle = computed(() => getLocalizedText(banner.value?.subtitle) || t('atlas.hero.subtitle'))
const imageUrl = computed(() => {
  if (!desktop.value || !banner.value?.image) return ''
  const url = getImageUrl(banner.value.image)
  return failedImages.value.has(url) ? '' : url
})
const onImageError = (event: Event) => {
  failedImages.value.add((event.target as HTMLImageElement).getAttribute('src') || '')
}
const onFocusOut = (event: FocusEvent) => {
  if (!(event.currentTarget as HTMLElement).contains(event.relatedTarget as Node | null)) carousel.setPaused('focus', false)
}
const onSubscribe = () => {
  const target = resolveAtlasBannerTarget(banner.value)
  if (target.type === 'plans') {
    document.getElementById('plans')?.scrollIntoView({ behavior: carouselState.value.reducedMotion ? 'auto' : 'smooth' })
  } else if (target.type === 'window') {
    window.open(target.value, target.target)
  } else {
    void router.push(target.value)
  }
}

let disposed = false
let stopObserving: (() => void) | undefined
onMounted(async () => {
  stopObserving = observeAtlasBannerEnvironment({
    desktop: value => { desktop.value = value },
    reducedMotion: value => carousel.setPaused('reduced-motion', value),
    hidden: value => carousel.setPaused('hidden', value),
  })
  try {
    const response = await bannerAPI.list({ position: 'home_hero', limit: 5 })
    if (disposed) return
    banners.value = selectAtlasBanners(response.data.data)
  } catch {
    if (disposed) return
    banners.value = []
  } finally {
    if (!disposed) {
      loading.value = false
      carousel.setCount(banners.value.length)
    }
  }
})
onUnmounted(() => {
  disposed = true
  carousel.dispose()
  stopObserving?.()
})
</script>

<style scoped>
.atlas-banner-stage { display: grid; min-height: 260px; }
.atlas-banner-slide { grid-area: 1 / 1; padding: 40px 0; }
.atlas-banner-copy h1 { font-size: 30px; }
.atlas-banner-control {
  display: inline-grid;
  width: 44px;
  min-height: 44px;
  place-items: center;
  border-radius: 9999px;
  color: var(--muted-foreground);
}
.atlas-banner-control:hover { background: var(--secondary); color: var(--foreground); }
.atlas-banner-control:focus-visible { outline: 2px solid var(--foreground); outline-offset: 2px; }
@media (min-width: 768px) {
  .atlas-banner-stage { min-height: 0; aspect-ratio: 8 / 3; }
  .atlas-banner-slide { position: absolute; inset: 0; padding: 0 6%; }
  .atlas-banner-copy { width: 58%; }
  .atlas-banner-copy h1 { font-size: clamp(30px, 4.1vw, 48px); }
  .atlas-banner-with-image { background: #f4f5fa; color: #080b12; }
}
.atlas-banner-fade-enter-active, .atlas-banner-fade-leave-active { transition: opacity 180ms ease; }
.atlas-banner-fade-enter-from, .atlas-banner-fade-leave-to { opacity: 0; }
.atlas-banner-fade-leave-active { pointer-events: none; }
@media (prefers-reduced-motion: reduce) {
  .atlas-banner-fade-enter-active, .atlas-banner-fade-leave-active { transition: none; }
}
</style>
