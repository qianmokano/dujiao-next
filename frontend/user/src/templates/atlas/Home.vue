<template>
  <div>
    <!-- ==================== 列表模式 ==================== -->
    <template v-if="isListMode">
      <section class="mx-auto w-full max-w-[1120px] px-5 pb-10 pt-24 sm:px-6 sm:pt-28">
        <h1 class="text-[26px] font-semibold tracking-[-0.01em] sm:text-[30px]">{{ t('nav.products') }}</h1>

        <div class="mt-8 grid items-start gap-8 lg:grid-cols-[220px_1fr]">
          <AtlasCategorySidebar
            :category-groups="categoryGroups"
            :selected-category="selectedCategory"
            :expanded-parent-ids="expandedParentIds"
            @select="selectCategory"
            @toggle="toggleParentCategory"
          />

          <main class="min-w-0">
            <!-- 搜索 -->
            <div class="relative mb-5">
              <Search class="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                v-model="searchQuery"
                class="h-11 w-full rounded-md border bg-card pl-10 pr-10 text-[14.5px] text-foreground transition-colors placeholder:text-muted-foreground focus:border-hairline-strong focus:outline-none"
                :placeholder="t('products.searchBoxPlaceholder')"
                :aria-label="t('products.searchLabel')"
                @keydown.enter="onSearch"
              />
              <button v-if="searchQuery" type="button" class="absolute right-2.5 top-1/2 grid h-6 w-6 -translate-y-1/2 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" :aria-label="t('blog.searchClear')" @click="clearSearch"><X class="h-3.5 w-3.5" /></button>
            </div>

            <!-- 骨架 -->
            <div v-if="listLoading" class="space-y-8">
              <div v-for="i in 3" :key="i">
                <div class="mb-3 h-4 w-32 rounded bg-secondary"></div>
                <div class="space-y-4 border-t py-2">
                  <div v-for="j in 3" :key="j" class="flex items-center gap-3.5 py-2">
                    <div class="h-11 w-11 flex-none rounded-md bg-secondary"></div>
                    <div class="flex-1 space-y-2">
                      <div class="h-3.5 w-1/3 rounded bg-secondary"></div>
                      <div class="h-3 w-1/4 rounded bg-secondary"></div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 分组列表 -->
            <div v-else-if="listProductGroups.length" class="space-y-8">
              <div v-for="group in listProductGroups" :key="group.categoryId ?? 'uncategorized'">
                <div class="mb-1 flex items-baseline gap-2">
                  <h2 class="text-[15px] font-semibold text-foreground">{{ group.categoryName }}</h2>
                  <span class="text-[12.5px] text-muted-foreground">{{ group.products.length }}</span>
                </div>
                <div class="border-t">
                  <AtlasProductListItem
                    v-for="product in group.products"
                    :key="product.id"
                    :product="product"
                    @quick-buy="openQuickBuy"
                  />
                </div>
              </div>

              <nav v-if="listTotalPages > 1" class="mt-8 flex flex-wrap items-center justify-center gap-1.5">
                <button class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="listCurrentPage <= 1" :aria-label="t('common.previousBanner')" @click="listChangePage(listCurrentPage - 1)"><ChevronLeft class="h-4 w-4" /></button>
                <button
                  v-for="p in listPageWindow"
                  :key="p"
                  class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-[13.5px] transition-colors"
                  :class="p === listCurrentPage ? 'border-primary font-medium text-primary' : 'text-muted-foreground hover:border-hairline-strong hover:text-foreground'"
                  @click="listChangePage(p)"
                >{{ p }}</button>
                <button class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="listCurrentPage >= listTotalPages" :aria-label="t('common.nextBanner')" @click="listChangePage(listCurrentPage + 1)"><ChevronRight class="h-4 w-4" /></button>
              </nav>
            </div>

            <!-- 空态 -->
            <div v-else class="flex flex-col items-center gap-3 border-t py-16 text-center text-muted-foreground">
              <component :is="(searchQuery || selectedCategory) ? SearchX : PackageOpen" class="h-11 w-11 opacity-50" :stroke-width="1.5" />
              <p class="text-[14px]">{{ (searchQuery || selectedCategory) ? t('products.emptyFiltered') : t('products.empty') }}</p>
              <button v-if="searchQuery || selectedCategory" type="button" class="mt-1 rounded-md border px-3.5 py-1.5 text-[13px] text-foreground transition-colors hover:border-hairline-strong" @click="resetFilters">{{ t('products.clearFilters') }}</button>
            </div>
          </main>
        </div>
      </section>
    </template>

    <!-- ==================== 默认：订阅方案首页 ==================== -->
    <template v-else>
      <AtlasBannerHero />

      <!-- 订阅方案 -->
      <section id="plans" class="mx-auto w-full max-w-[1120px] scroll-mt-20 px-5 py-16 sm:px-6 sm:py-20">
        <div class="mb-10 max-w-[60ch]">
          <h2 class="text-[24px] font-semibold tracking-[-0.01em] sm:text-[28px]">{{ t('atlas.plans.title') }}</h2>
          <p class="mt-2 text-[14px] leading-relaxed text-muted-foreground">{{ t('atlas.plans.description') }}</p>
        </div>

        <div v-if="productsLoading" class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          <div v-for="i in 3" :key="i" class="rounded-[10px] border p-6">
            <div class="h-3 w-20 rounded bg-secondary"></div>
            <div class="mt-3 h-5 w-2/3 rounded bg-secondary"></div>
            <div class="mt-2 h-3.5 w-full rounded bg-secondary"></div>
            <div class="mt-6 h-7 w-24 rounded bg-secondary"></div>
            <div class="mt-6 h-10 w-full rounded-md bg-secondary"></div>
          </div>
        </div>

        <div v-else-if="products.length" class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          <AtlasPlanCard
            v-for="product in products"
            :key="product.id"
            :product="product"
            @quick-buy="openQuickBuy"
          />
        </div>

        <div v-else class="rounded-[10px] border border-dashed py-16 text-center text-muted-foreground">
          <PackageOpen class="mx-auto mb-4 h-12 w-12 opacity-50" :stroke-width="1.5" />
          <p class="text-[14px]">{{ t('home.featured.empty') }}</p>
        </div>
      </section>

      <!-- 最新动态（博客/公告，受后台开关控制） -->
      <section v-if="latestSectionVisible" class="border-t">
        <div class="mx-auto w-full max-w-[1120px] px-5 py-16 sm:px-6 sm:py-20">
          <div class="mb-8 flex flex-wrap items-end justify-between gap-3">
            <h2 class="text-[20px] font-semibold tracking-[-0.01em]">{{ t('home.latest.title') }}</h2>
            <div class="flex items-center gap-4 text-[13.5px]">
              <RouterLink v-if="blogEnabled" to="/blog" class="text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.blog') }}</RouterLink>
              <RouterLink v-if="noticeEnabled" to="/notice" class="text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.notice') }}</RouterLink>
            </div>
          </div>

          <div v-if="posts.length" class="divide-y border-t">
            <article
              v-for="post in posts"
              :key="post.id"
              class="group flex cursor-pointer flex-wrap items-baseline gap-x-6 gap-y-1 py-5"
              @click="goToPost(post.slug)"
            >
              <time class="w-28 flex-none text-[13px] tabular-nums text-muted-foreground">{{ formatDate(post.published_at) }}</time>
              <div class="min-w-0 flex-1">
                <h3 class="text-[15px] font-medium text-foreground transition-colors group-hover:text-primary">{{ getLocalizedText(post.title) }}</h3>
                <p class="mt-1 line-clamp-1 text-[13.5px] text-muted-foreground">{{ getLocalizedText(post.summary) }}</p>
              </div>
            </article>
          </div>
          <div v-else class="border-t py-10 text-center text-[14px] text-muted-foreground">{{ t('blog.empty') }}</div>
        </div>
      </section>
    </template>

    <ProductQuickBuy
      v-if="quickBuyProduct"
      :product="quickBuyProduct"
      :visible="quickBuyVisible"
      @update:visible="quickBuyVisible = $event"
    />

    <AnnouncementModal
      v-if="activeAnnouncement"
      :announcement="activeAnnouncement"
      :visible="announcementVisible"
      @update:visible="announcementVisible = $event"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ChevronLeft, ChevronRight, PackageOpen, Search, SearchX, X } from 'lucide-vue-next'
import { postAPI, productAPI } from '../../api'
import { useLocalized } from '../../composables/useProduct'
import { useProductList } from '../../composables/useProductList'
import { useProductListGroups } from '../../composables/useProductListGroups'
import { usePageSeo } from '../../composables/usePageSeo'
import { useAppStore } from '../../stores/app'
import AtlasPlanCard from './components/AtlasPlanCard.vue'
import AtlasProductListItem from './components/AtlasProductListItem.vue'
import AtlasCategorySidebar from './components/AtlasCategorySidebar.vue'
import AtlasBannerHero from './components/AtlasBannerHero.vue'
import ProductQuickBuy from '../../components/ProductQuickBuy.vue'
import AnnouncementModal from '../../components/AnnouncementModal.vue'
import { useAnnouncement, type HomeAnnouncement } from '../../composables/useAnnouncement'

const router = useRouter()
const { t } = useI18n()
const { getLocalizedText } = useLocalized()
const appStore = useAppStore()

const isListMode = computed(() => appStore.config?.template_mode === 'list')
const navBuiltin = computed(() => (appStore.config?.nav_config as { builtin?: Record<string, boolean> } | undefined)?.builtin)
const blogEnabled = computed(() => navBuiltin.value?.blog !== false)
const noticeEnabled = computed(() => navBuiltin.value?.notice !== false)
const latestSectionVisible = computed(() => blogEnabled.value || noticeEnabled.value)

// ==================== 订阅方案 ====================
const products = ref<any[]>([])
const productsLoading = ref(true)
const posts = ref<any[]>([])
const quickBuyProduct = ref<any>(null)
const quickBuyVisible = ref(false)

const { shouldShow } = useAnnouncement()
const activeAnnouncement = ref<HomeAnnouncement | null>(null)
const announcementVisible = ref(false)

const openQuickBuy = (product: any) => {
  quickBuyProduct.value = product
  quickBuyVisible.value = true
}

const loadPlans = async () => {
  productsLoading.value = true
  try {
    const response = await productAPI.list({ page: 1, page_size: 12 })
    products.value = response.data.data || []
  } catch (error) {
    console.error('Failed to load products:', error)
  } finally {
    productsLoading.value = false
  }
}

const loadLatestPosts = async () => {
  if (!latestSectionVisible.value) return
  try {
    const params: Record<string, unknown> = { page: 1, page_size: 4 }
    if (blogEnabled.value && !noticeEnabled.value) params.type = 'blog'
    if (!blogEnabled.value && noticeEnabled.value) params.type = 'notice'
    const response = await postAPI.list(params)
    posts.value = response.data.data || []
  } catch (error) {
    console.error('Failed to load posts:', error)
  }
}

const formatDate = (dateString: string) => {
  if (!dateString) return ''
  return new Date(dateString).toLocaleDateString()
}

const goToPost = (slug: string) => {
  router.push(`/blog/${slug}`)
}

// ==================== 列表模式 ====================
const {
  loading: listLoading,
  products: listProducts,
  selectedCategory,
  searchQuery,
  currentPage: listCurrentPage,
  totalPages: listTotalPages,
  expandedParentIds,
  categoryGroups,
  categoryMap,
  selectCategory,
  toggleParentCategory,
  changePage: listChangePage,
  clearSearch,
  onSearch,
  initialize: listInitialize,
  cleanup: listCleanup,
} = useProductList({ pageSize: 20, homeRouteName: 'home' })

const resetFilters = () => {
  clearSearch()
  selectCategory(null)
}

const listProductGroups = useProductListGroups(listProducts, categoryMap)

const listPageWindow = computed(() => {
  const total = listTotalPages.value
  const current = listCurrentPage.value
  const start = Math.max(1, Math.min(current - 2, total - 4))
  const end = Math.min(total, start + 4)
  const pages: number[] = []
  for (let p = start; p <= end; p++) pages.push(p)
  return pages
})

// ==================== SEO ====================
const route = useRoute()
const seoCategoryName = computed(() => {
  if (!selectedCategory.value) return ''
  const cat = categoryMap.value.get(selectedCategory.value)
  return cat ? getLocalizedText(cat.name) : ''
})
usePageSeo({
  canonicalPath: () => route.path,
  title: () => {
    if (route.name === 'category-products') {
      return seoCategoryName.value || t('nav.products')
    }
    if (route.name === 'products') return t('nav.products')
    return undefined
  },
})

// ==================== Lifecycle ====================
const showAnnouncementIfNeeded = () => {
  const announcement = appStore.config?.announcement as HomeAnnouncement | undefined
  if (announcement && shouldShow(announcement)) {
    activeAnnouncement.value = announcement
    announcementVisible.value = true
  }
}

onMounted(async () => {
  await appStore.loadConfig()
  if (isListMode.value) {
    await listInitialize()
  } else {
    await Promise.all([loadPlans(), loadLatestPosts()])
  }
  showAnnouncementIfNeeded()
})

onUnmounted(() => {
  listCleanup()
})
</script>
