<template>
  <div>
    <AtlasPageHeader :title="pageTitle" />

    <div class="mx-auto w-full max-w-[1120px] px-5 pb-10 sm:px-6">
      <div class="grid items-start gap-8 py-8 lg:grid-cols-[220px_1fr]">
        <!-- 筛选侧栏：搜索 + 分类 -->
        <div class="grid min-w-0 gap-5 lg:sticky lg:top-[84px]">
          <div class="relative">
            <Search class="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input
              v-model="searchQuery"
              class="h-11 w-full rounded-md border bg-card pl-10 pr-10 text-[14.5px] text-foreground transition-colors placeholder:text-muted-foreground focus:border-hairline-strong focus:outline-none"
              :placeholder="t('products.searchBoxPlaceholder')"
              :aria-label="t('products.searchLabel')"
            />
            <button v-if="searchQuery" type="button" class="absolute right-2.5 top-1/2 grid h-6 w-6 -translate-y-1/2 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" :aria-label="t('blog.searchClear')" @click="clearSearch"><X class="h-3.5 w-3.5" /></button>
          </div>

          <AtlasCategorySidebar
            :category-groups="categoryGroups"
            :selected-category="selectedCategory"
            :expanded-parent-ids="expandedParentIds"
            @select="selectCategory"
            @toggle="toggleParentCategory"
          />
        </div>

        <!-- 商品区 -->
        <section class="min-w-0">
          <div v-if="loading" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            <div v-for="i in 6" :key="i" class="rounded-[10px] border p-5">
              <div class="h-3 w-20 rounded bg-secondary"></div>
              <div class="mt-3 h-5 w-2/3 rounded bg-secondary"></div>
              <div class="mt-6 h-6 w-24 rounded bg-secondary"></div>
            </div>
          </div>

          <template v-else-if="products.length">
            <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              <AtlasProductCard
                v-for="(product, idx) in products"
                :key="product.id"
                :product="product"
                :index="idx"
                @quick-buy="openQuickBuy"
              />
            </div>

            <nav v-if="totalPages > 1" class="mt-8 flex flex-wrap justify-center gap-1.5">
              <button class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="currentPage <= 1" @click="changePage(currentPage - 1)"><ChevronLeft class="h-4 w-4" /></button>
              <button
                v-for="p in pageWindow"
                :key="p"
                class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-[13.5px] transition-colors"
                :class="p === currentPage ? 'border-primary font-medium text-primary' : 'text-muted-foreground hover:border-hairline-strong hover:text-foreground'"
                @click="changePage(p)"
              >{{ p }}</button>
              <button class="grid h-9 min-w-[36px] place-items-center rounded-md border px-2.5 text-muted-foreground transition-colors hover:border-hairline-strong hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40" :disabled="currentPage >= totalPages" @click="changePage(currentPage + 1)"><ChevronRight class="h-4 w-4" /></button>
            </nav>
          </template>

          <div v-else class="flex flex-col items-center gap-3 rounded-[10px] border border-dashed py-16 text-center text-muted-foreground">
            <component :is="searchQuery || selectedCategory ? SearchX : PackageOpen" class="h-11 w-11 opacity-50" :stroke-width="1.5" />
            <p class="text-[14px]">{{ (searchQuery || selectedCategory) ? t('products.emptyFiltered') : t('products.empty') }}</p>
            <button v-if="searchQuery || selectedCategory" type="button" class="mt-1 rounded-md border px-3.5 py-1.5 text-[13px] text-foreground transition-colors hover:border-hairline-strong" @click="resetFilters">{{ t('products.clearFilters') }}</button>
          </div>
        </section>
      </div>
    </div>

    <ProductQuickBuy
      v-if="quickBuyProduct"
      :product="quickBuyProduct"
      :visible="quickBuyVisible"
      @update:visible="quickBuyVisible = $event"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ChevronLeft, ChevronRight, PackageOpen, Search, SearchX, X } from 'lucide-vue-next'
import { useProductList } from '../../composables/useProductList'
import { usePageSeo } from '../../composables/usePageSeo'
import { useLocalized } from '../../composables/useProduct'
import type { PublicCategory } from '../../utils/category'
import AtlasPageHeader from './components/AtlasPageHeader.vue'
import AtlasProductCard from './components/AtlasProductCard.vue'
import AtlasCategorySidebar from './components/AtlasCategorySidebar.vue'
import ProductQuickBuy from '../../components/ProductQuickBuy.vue'

const { t } = useI18n()
const route = useRoute()
const { getLocalizedText } = useLocalized()

const quickBuyProduct = ref<any>(null)
const quickBuyVisible = ref(false)
const openQuickBuy = (product: any) => {
  quickBuyProduct.value = product
  quickBuyVisible.value = true
}

const {
  loading,
  products,
  selectedCategory,
  searchQuery,
  currentPage,
  totalPages,
  expandedParentIds,
  categoryGroups,
  categoryMap,
  selectCategory,
  toggleParentCategory,
  changePage,
  clearSearch,
  initialize,
  cleanup,
} = useProductList({ pageSize: 12, homeRouteName: 'products' })

const catName = (cat: PublicCategory) => getLocalizedText(cat.name) || cat.slug || ''

const selectedCategoryName = computed(() => {
  if (!selectedCategory.value) return ''
  const cat = categoryMap.value.get(selectedCategory.value)
  return cat ? catName(cat) : ''
})

const pageTitle = computed(() => {
  if (route.name === 'category-products') return selectedCategoryName.value || t('nav.products')
  return t('nav.products')
})

// 分页窗口：当前页前后各 2 页
const pageWindow = computed(() => {
  const total = totalPages.value
  const cur = currentPage.value
  const start = Math.max(1, cur - 2)
  const end = Math.min(total, start + 4)
  const realStart = Math.max(1, end - 4)
  const pages: number[] = []
  for (let p = realStart; p <= end; p++) pages.push(p)
  return pages
})

const resetFilters = () => {
  clearSearch()
  selectCategory(null)
}

usePageSeo({
  canonicalPath: () => route.path,
  title: () => pageTitle.value,
})

onMounted(() => { void initialize() })
onUnmounted(() => cleanup())
</script>
