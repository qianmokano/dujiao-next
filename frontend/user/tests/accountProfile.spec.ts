import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, reactive, type App } from 'vue'
import ProfileAvatar from '../src/components/shared/ProfileAvatar.vue'
import ProfilePanel from '../src/views/personal/ProfilePanel.vue'

const mocks = vi.hoisted(() => ({ save: vi.fn(), notify: vi.fn(), app: null as any, profile: null as any }))
vi.mock('../src/stores/app', () => ({ useAppStore: () => mocks.app }))
vi.mock('../src/stores/userProfile', () => ({ useUserProfileStore: () => mocks.profile }))
vi.mock('../src/composables/useFeedback', () => ({ useFeedback: () => ({ success: mocks.notify }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/ui/select', () => ({
  Select: { props: ['modelValue'], emits: ['update:modelValue'], template: `<select :value="modelValue" @change="$emit('update:modelValue', $event.target.value)"><option value="en-US">English</option><option value="zh-CN">Chinese</option></select>` },
  SelectTrigger: { template: '<slot />' }, SelectValue: { template: '<span />' },
  SelectContent: { template: '<slot />' }, SelectItem: { template: '<slot />' },
}))

let app: App | undefined
let root: HTMLDivElement
const settle = async () => { await Promise.resolve(); await nextTick(); await nextTick() }
const mount = async (component: any, props = {}) => {
  root = document.createElement('div'); document.body.appendChild(root)
  app = createApp({ render: () => h(component, props) }); app.mount(root); await settle()
}
beforeEach(() => {
  vi.clearAllMocks()
  mocks.app = reactive({ identityPolicyReady: true, config: { oidc_auth: { only_enabled: true, account_url: 'https://auth.example/account' } } })
  mocks.profile = reactive({ profile: { email: 'original@example.com', nickname: 'Passport', locale: 'en-US' }, savingProfile: false, profileError: '', saveProfile: mocks.save })
  mocks.save.mockResolvedValue(true)
  mocks.notify.mockImplementation((_message: string, fn: () => void) => fn())
})
afterEach(() => { app?.unmount(); app = undefined; document.body.innerHTML = '' })

describe('shared profile avatar', () => {
  it('shows initials for missing, invalid and failed images, then retries a new URL', async () => {
    const props = reactive({ src: '', initial: 'K' })
    await mount(ProfileAvatar, props)
    expect(root.textContent).toBe('K')
    props.src = 'javascript:alert(1)'; await settle()
    expect(root.querySelector('img')).toBeNull()
    props.src = 'https://auth.example/avatar.png'; await settle()
    expect(root.querySelector('img')?.src).toBe(props.src)
    root.querySelector('img')!.dispatchEvent(new Event('error')); await settle()
    expect(root.textContent).toBe('K')
    props.src = 'https://auth.example/new.png'; await settle()
    expect(root.querySelector('img')?.src).toBe(props.src)
  })
})

describe('profile identity ownership', () => {
  it('disables the managed name, links the account center and saves only language', async () => {
    await mount(ProfilePanel)
    const inputs = root.querySelectorAll<HTMLInputElement>('input')
    expect(inputs[1]?.disabled).toBe(true)
    expect(root.querySelector('a')?.href).toBe('https://auth.example/account')
    root.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true })); await settle()
    expect(mocks.save).toHaveBeenCalledWith({ locale: 'en-US' })
    expect(mocks.notify).toHaveBeenCalled()
  })
  it.each([null, {}, { only_enabled: false }])('keeps local editing closed until policy loads: %j', async (config) => {
    mocks.app.identityPolicyReady = false
    mocks.app.config = config ? { oidc_auth: config } : null
    await mount(ProfilePanel)
    expect(root.querySelectorAll<HTMLInputElement>('input')[1]?.disabled).toBe(true)
    expect(root.querySelector('a')).toBeNull()
    root.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true })); await settle()
    expect(mocks.save).toHaveBeenCalledWith({ locale: 'en-US' })
  })
  it('restores local name editing when the current mode is local', async () => {
    mocks.app.config.oidc_auth.only_enabled = false
    await mount(ProfilePanel)
    expect(root.querySelectorAll<HTMLInputElement>('input')[1]?.disabled).toBe(false)
    expect(root.querySelector('a')).toBeNull()
    const name = root.querySelectorAll<HTMLInputElement>('input')[1]!
    name.value = 'Local'; name.dispatchEvent(new Event('input')); await settle()
    const locale = root.querySelector<HTMLSelectElement>('select')!
    locale.value = 'zh-CN'; locale.dispatchEvent(new Event('change')); await settle()
    root.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true })); await settle()
    expect(mocks.save).toHaveBeenCalledWith({ nickname: 'Local', locale: 'zh-CN' })
  })
  it('shows save failures and tolerates missing profile values', async () => {
    mocks.profile.profile = null
    mocks.save.mockResolvedValue(false)
    await mount(ProfilePanel)
    root.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true })); await settle()
    expect(root.textContent).toContain('personalCenter.common.saveFailed')
    mocks.profile.profileError = 'Saved language failed'
    mocks.profile.profile = { email: '', nickname: '', locale: '' }; await settle()
    root.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true })); await settle()
    expect(root.textContent).toContain('Saved language failed')
  })
})
