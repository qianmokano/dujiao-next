import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick, type App } from 'vue'
import AtlasBannerHero from '../src/templates/atlas/components/AtlasBannerHero.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(), push: vi.fn(),
  store: { locale: 'zh-CN', config: { brand: { site_name: 'Kano Store' } } },
}))
vi.mock('../src/api', () => ({ bannerAPI: { list: mocks.list } }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, params?: { n: number }) => params ? `${key}:${params.n}` : key }) }))
vi.mock('../src/stores/app', () => ({ useAppStore: () => mocks.store }))
vi.mock('../src/composables/useProduct', () => ({ useLocalized: () => ({ getLocalizedText: (value: Record<string, string> | undefined) => value?.[mocks.store.locale] || value?.['zh-CN'] || '' }) }))

const banner = (id: number) => ({ id, title: { 'zh-CN': `订阅 ${id}`, 'en-US': `Plan ${id}` }, subtitle: { 'zh-CN': `说明 ${id}` }, image: `/banner-${id}.png`, link_type: 'product', link_value: `/products/plan-${id}`, open_in_new_tab: false })
let app: App | undefined
let root: HTMLDivElement
let desktop: MediaQueryList
let motion: MediaQueryList
const settle = async () => { await Promise.resolve(); await nextTick(); await nextTick() }
const mount = async () => {
  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp(AtlasBannerHero)
  app.mount(root)
  await settle()
}
const control = (key: string) => root.querySelector<HTMLButtonElement>(`button[aria-label="${key}"]`)!
const cta = () => root.querySelector<HTMLButtonElement>('.atlas-banner-copy button')!
const setMatch = async (media: MediaQueryList, matches: boolean) => {
  Object.defineProperty(media, 'matches', { configurable: true, value: matches })
  media.dispatchEvent(new Event('change'))
  await settle()
}
const visibility = async (hidden: boolean) => {
  Object.defineProperty(document, 'hidden', { configurable: true, value: hidden })
  document.dispatchEvent(new Event('visibilitychange'))
  await settle()
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.clearAllMocks()
  mocks.store.locale = 'zh-CN'
  mocks.list.mockResolvedValue({ data: { data: [banner(1), banner(2), banner(3)] } })
  desktop = Object.assign(new EventTarget(), { matches: true }) as MediaQueryList
  motion = Object.assign(new EventTarget(), { matches: false }) as MediaQueryList
  vi.stubGlobal('matchMedia', vi.fn((query: string) => query.includes('min-width') ? desktop : motion))
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
})
afterEach(() => {
  app?.unmount()
  app = undefined
  document.body.innerHTML = ''
  vi.useRealTimers()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('Atlas Banner component', () => {
  it('loads the existing API, renders configured copy and navigates internally', async () => {
    await mount()
    expect(mocks.list).toHaveBeenCalledWith({ position: 'home_hero', limit: 5 })
    expect(root.querySelector('h1')?.textContent).toBe('订阅 1')
    expect(cta().textContent).toContain('atlas.hero.subscribe')
    expect(root.querySelector('img')?.getAttribute('src')).toBe('/banner-1.png')
    expect(control('common.switchBanner:1').getAttribute('aria-current')).toBe('true')
    cta().click()
    expect(mocks.push).toHaveBeenCalledWith('/products/plan-1')
  })

  it('never mounts mobile images, and responds to desktop breakpoint changes', async () => {
    Object.defineProperty(desktop, 'matches', { configurable: true, value: false })
    await mount()
    expect(root.querySelector('img')).toBeNull()
    await setMatch(desktop, true)
    expect(root.querySelector('img')).not.toBeNull()
    await setMatch(desktop, false)
    expect(root.querySelector('img')).toBeNull()
  })

  it('keeps copy and controls after an image fails and shows other images', async () => {
    await mount()
    root.querySelector('img')!.dispatchEvent(new Event('error'))
    await settle()
    expect(root.querySelector('img')).toBeNull()
    expect(cta().textContent).toContain('atlas.hero.subscribe')
    await setMatch(motion, true)
    control('common.nextBanner').click()
    await settle()
    expect(root.querySelector('img')?.getAttribute('src')).toBe('/banner-2.png')
    control('common.previousBanner').click()
    await settle()
    expect(root.querySelector('img')).toBeNull()
    control('common.switchBanner:3').click()
    await settle()
    expect(root.querySelector('h1')?.textContent).toBe('订阅 3')
  })

  it('uses the site name, default description and plans entry on empty data and API failure', async () => {
    const plans = document.createElement('div')
    plans.id = 'plans'
    const scroll = vi.fn()
    plans.scrollIntoView = scroll
    document.body.appendChild(plans)
    for (const fails of [false, true]) {
      await setMatch(motion, false)
      mocks.list.mockReset()
      if (fails) mocks.list.mockRejectedValue(new Error('offline'))
      else mocks.list.mockResolvedValue({ data: { data: [] } })
      await mount()
      expect(root.querySelector('h1')?.textContent).toBe('Kano Store')
      expect(root.querySelector('p')?.textContent).toBe('atlas.hero.subtitle')
      expect(cta().textContent).toContain('atlas.hero.cta')
      expect(root.querySelector('.atlas-banner-control')).toBeNull()
      cta().click()
      expect(scroll).toHaveBeenLastCalledWith({ behavior: 'smooth' })
      await setMatch(motion, true)
      cta().click()
      expect(scroll).toHaveBeenLastCalledWith({ behavior: 'auto' })
      app!.unmount()
      app = undefined
      root.remove()
    }
  })

  it('keeps a single banner static, renders localized title, and preserves external/new-tab links', async () => {
    mocks.store.locale = 'en-US'
    mocks.list.mockResolvedValue({ data: { data: [{ ...banner(1), link_value: 'https://example.test', open_in_new_tab: true }] } })
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)
    await mount()
    expect(root.querySelector('h1')?.textContent).toBe('Plan 1')
    expect(root.querySelector('.atlas-banner-control')).toBeNull()
    cta().click()
    expect(open).toHaveBeenCalledWith('https://example.test', '_blank')
    await vi.advanceTimersByTimeAsync(20000)
    expect(root.querySelector('h1')?.textContent).toBe('Plan 1')
    open.mockRestore()
  })

  it('autoplays, stacks hover/focus/visibility pauses and preserves manual pause', async () => {
    await mount()
    await vi.advanceTimersByTimeAsync(5300)
    expect(root.querySelector('h1')?.textContent).toBe('订阅 2')
    const region = root.querySelector('section')!
    region.dispatchEvent(new Event('mouseenter'))
    cta().dispatchEvent(new FocusEvent('focusin', { bubbles: true }))
    await visibility(true)
    region.dispatchEvent(new Event('mouseleave'))
    cta().dispatchEvent(new FocusEvent('focusout', { bubbles: true, relatedTarget: control('common.nextBanner') }))
    await vi.advanceTimersByTimeAsync(10000)
    expect(root.querySelector('h1')?.textContent).toBe('订阅 2')
    await visibility(false)
    await vi.advanceTimersByTimeAsync(10000)
    expect(root.querySelector('h1')?.textContent).toBe('订阅 2')
    region.dispatchEvent(new FocusEvent('focusout', { bubbles: true }))
    await vi.advanceTimersByTimeAsync(5300)
    expect(root.querySelector('h1')?.textContent).toBe('订阅 3')
    control('atlas.hero.pause').click()
    await settle()
    expect(control('atlas.hero.resume').getAttribute('aria-pressed')).toBe('true')
    await visibility(true)
    await visibility(false)
    await vi.advanceTimersByTimeAsync(10000)
    expect(root.querySelector('h1')?.textContent).toBe('订阅 3')
    control('atlas.hero.resume').click()
    await vi.advanceTimersByTimeAsync(5300)
    expect(root.querySelector('h1')?.textContent).toBe('订阅 1')
  })

  it('reduced motion removes autoplay and animation while retaining manual controls', async () => {
    Object.defineProperty(motion, 'matches', { configurable: true, value: true })
    await mount()
    expect(root.querySelector('button[aria-label="atlas.hero.pause"]')).toBeNull()
    await vi.advanceTimersByTimeAsync(10000)
    expect(root.querySelector('h1')?.textContent).toBe('订阅 1')
    control('common.nextBanner').click()
    await settle()
    expect(root.querySelector('h1')?.textContent).toBe('订阅 2')
    expect(root.querySelector('[class*="fade-enter"]')).toBeNull()
  })

  it('cleans up timers and media/visibility listeners on unmount', async () => {
    const removeDesktop = vi.spyOn(desktop, 'removeEventListener')
    const removeMotion = vi.spyOn(motion, 'removeEventListener')
    const removeDocument = vi.spyOn(document, 'removeEventListener')
    await mount()
    app!.unmount()
    app = undefined
    expect(removeDesktop).toHaveBeenCalledWith('change', expect.any(Function))
    expect(removeMotion).toHaveBeenCalledWith('change', expect.any(Function))
    expect(removeDocument).toHaveBeenCalledWith('visibilitychange', expect.any(Function))
    await vi.advanceTimersByTimeAsync(20000)
    expect(root.children.length).toBe(0)
  })

  it.each([false, true])('ignores late API resolution/rejection after unmount (reject=%s)', async rejects => {
    let resolve!: (value: unknown) => void
    let reject!: (value: unknown) => void
    mocks.list.mockReturnValue(new Promise((yes, no) => { resolve = yes; reject = no }))
    await mount()
    expect(root.querySelector('[aria-busy="true"]')).not.toBeNull()
    app!.unmount()
    app = undefined
    if (rejects) reject(new Error('offline'))
    else resolve({ data: { data: [banner(1), banner(2)] } })
    await settle()
    expect(vi.getTimerCount()).toBe(0)
  })
})
