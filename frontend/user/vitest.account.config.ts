import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  test: {
    environment: 'jsdom',
    include: ['tests/account*.spec.ts'],
    coverage: {
      provider: 'v8',
      reportsDirectory: './node_modules/.tmp/coverage-account',
      include: ['src/components/shared/ProfileAvatar.vue', 'src/views/personal/ProfilePanel.vue', 'src/composables/useSSOCaptcha.ts', 'src/components/captcha/SSOCaptcha.vue'],
      reporter: ['text'],
      thresholds: { lines: 80, branches: 80, functions: 80, statements: 80 },
    },
  },
})
