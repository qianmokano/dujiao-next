<template>
  <aside class="min-w-0 lg:sticky lg:top-[84px]">
    <!-- 移动端横向 chips / 桌面竖向列表 -->
    <div class="flex gap-1.5 overflow-x-auto pb-1 lg:grid lg:gap-0.5 lg:overflow-visible lg:pb-0">
      <button
        type="button"
        class="flex-none whitespace-nowrap rounded-md px-3 py-2 text-[13.5px] transition-colors lg:w-full lg:text-left"
        :class="selectedCategory === null
          ? 'bg-secondary font-medium text-foreground'
          : 'text-muted-foreground hover:bg-secondary hover:text-foreground'"
        @click="$emit('select', null)"
      >
        {{ t('products.allCategories') }}
      </button>
      <template v-for="grp in categoryGroups" :key="grp.id">
        <div class="flex flex-none items-center gap-0.5 lg:w-full lg:min-w-0">
          <button
            type="button"
            class="flex min-w-0 max-w-[65vw] flex-none items-center gap-2 rounded-md px-3 py-2 text-[13.5px] transition-colors lg:w-full lg:max-w-none lg:flex-1 lg:text-left"
            :class="selectedCategory === grp.id
              ? 'bg-secondary font-medium text-foreground'
              : 'text-muted-foreground hover:bg-secondary hover:text-foreground'"
            @click="$emit('select', grp.id)"
          >
            <img v-if="grp.icon" :src="getImageUrl(grp.icon)" :alt="catName(grp)" loading="lazy" class="h-4.5 w-4.5 flex-none rounded object-cover" />
            <span class="min-w-0 truncate">{{ catName(grp) }}</span>
          </button>
          <button
            v-if="grp.children.length"
            type="button"
            class="hidden h-8 w-8 flex-none place-items-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground lg:grid"
            :aria-expanded="expandedParentIds.includes(grp.id)"
            :aria-label="catName(grp)"
            @click="$emit('toggle', grp.id)"
          >
            <ChevronDown class="h-4 w-4 transition-transform" :class="{ 'rotate-180': expandedParentIds.includes(grp.id) }" />
          </button>
        </div>
        <template v-if="grp.children.length && expandedParentIds.includes(grp.id)">
          <button
            v-for="child in grp.children"
            :key="child.id"
            type="button"
            class="hidden w-full min-w-0 truncate rounded-md py-1.5 pl-8 pr-3 text-left text-[13px] transition-colors lg:block"
            :class="selectedCategory === child.id ? 'bg-secondary font-medium text-foreground' : 'text-muted-foreground hover:bg-secondary hover:text-foreground'"
            @click="$emit('select', child.id)"
          >
            {{ catName(child) }}
          </button>
        </template>
      </template>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ChevronDown } from 'lucide-vue-next'
import { getImageUrl } from '../../../utils/image'
import { useLocalized } from '../../../composables/useProduct'
import type { PublicCategory } from '../../../utils/category'

defineProps<{
  categoryGroups: (PublicCategory & { children: PublicCategory[] })[]
  selectedCategory: number | null
  expandedParentIds: number[]
}>()

defineEmits<{ select: [id: number | null]; toggle: [id: number] }>()

const { t } = useI18n()
const { getLocalizedText } = useLocalized()
const catName = (cat: PublicCategory) => getLocalizedText(cat.name) || cat.slug || ''
</script>
