import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAppStore } from '../src/stores/app'

const mocks = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('../src/api', () => ({ configAPI: { get: mocks.get } }))
vi.mock('@unhead/vue', () => ({ useHead: vi.fn() }))
vi.mock('../src/utils/customScripts', () => ({ applyCustomScripts: vi.fn() }))
vi.mock('../src/i18n', () => ({ detectLocale: () => 'en-US', setI18nLocale: vi.fn() }))

beforeEach(() => { setActivePinia(createPinia()); vi.clearAllMocks() })
describe('configuration identity policy state', () => {
  it('closes identity editing on initial and later failed loads', async () => {
    const store = useAppStore()
    expect(store.identityPolicyReady).toBe(false)
    mocks.get.mockResolvedValueOnce({ data: { data: { sso_auth: { only_enabled: false } } } })
    await store.loadConfig()
    expect(store.identityPolicyReady).toBe(true)
    mocks.get.mockRejectedValueOnce(new Error('offline'))
    await store.loadConfig(true)
    expect(store.identityPolicyReady).toBe(false)
  })
  it.each([null, {}, { only_enabled: 'false' }])('does not accept an incomplete policy: %j', async (sso_auth) => {
    mocks.get.mockResolvedValue({ data: { data: { sso_auth } } })
    const store = useAppStore(); await store.loadConfig()
    expect(store.identityPolicyReady).toBe(false)
  })
})
