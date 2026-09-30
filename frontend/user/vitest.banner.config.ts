import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'jsdom',
    include: ['tests/atlasBannerHero.spec.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/templates/atlas/components/AtlasBannerHero.vue'],
      reporter: ['text'],
      thresholds: { lines: 80, branches: 80, functions: 80, statements: 80 },
    },
  },
})
