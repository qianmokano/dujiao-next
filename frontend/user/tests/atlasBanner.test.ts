import test from 'node:test'
import assert from 'node:assert/strict'
import { createAtlasBannerCarousel, observeAtlasBannerEnvironment, resolveAtlasBannerTarget, selectAtlasBanners } from '../src/utils/atlasBanner.ts'

const fixture = (id = 1) => ({ id, title: { 'zh-CN': 'ChatGPT Plus 月度订阅' }, subtitle: { 'en-US': 'Make room for inspiration' }, image: '/banner.png', link_type: 'product', link_value: '/products/chatgpt-plus', open_in_new_tab: false })

test('banner selection keeps the API priority order, limits to five and does not mutate input', () => {
  const rows = [7, 3, 1, 2, 8, 4].map(fixture)
  assert.deepEqual(selectAtlasBanners(rows).map(b => b.id), [7, 3, 1, 2, 8])
  assert.equal(rows.length, 6)
  assert.notEqual(selectAtlasBanners(rows)[0], rows[0])
})

test('malformed lists and rows fall back; valid text-only banners and image-only banners survive', () => {
  for (const value of [undefined, null, {}, 'banners', []]) assert.deepEqual(selectAtlasBanners(value), [])
  const rows = [null, false, {}, { id: 0 }, { id: 1.5 }, { id: 1 },
    { ...fixture(2), image: ' ', title: { 'zh-CN': ' ' }, subtitle: [] },
    { ...fixture(3), image: null, title: { 'zh-CN': 'Text only', 'en-US': 23 }, subtitle: null, link_type: null, link_value: 42, open_in_new_tab: 'true' },
    { ...fixture(4), image: ' /image.png ', title: 'invalid', subtitle: {} }, fixture(3),
  ]
  assert.deepEqual(selectAtlasBanners(rows), [
    { id: 3, title: { 'zh-CN': 'Text only' }, subtitle: {}, image: '', link_type: 'none', link_value: '', open_in_new_tab: false },
    { ...fixture(4), title: {}, subtitle: {}, image: '/image.png' },
  ])
})

test('navigation preserves internal, external and new-window semantics and scrolls without a link', () => {
  assert.deepEqual(resolveAtlasBannerTarget(null), { type: 'plans' })
  const row = selectAtlasBanners([fixture()])[0]!
  assert.deepEqual(resolveAtlasBannerTarget({ ...row, link_type: 'none' }), { type: 'plans' })
  assert.deepEqual(resolveAtlasBannerTarget({ ...row, link_value: '' }), { type: 'plans' })
  assert.deepEqual(resolveAtlasBannerTarget(row), { type: 'router', value: '/products/chatgpt-plus' })
  assert.deepEqual(resolveAtlasBannerTarget({ ...row, open_in_new_tab: true }), { type: 'window', value: '/products/chatgpt-plus', target: '_blank' })
  assert.deepEqual(resolveAtlasBannerTarget({ ...row, link_type: 'external', link_value: 'https://example.test' }), { type: 'window', value: 'https://example.test', target: '_self' })
  assert.deepEqual(resolveAtlasBannerTarget({ ...row, link_value: 'HTTP://example.test', open_in_new_tab: true }), { type: 'window', value: 'HTTP://example.test', target: '_blank' })
})

test('empty and single-banner carousels never schedule autoplay', t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const updates: number[] = []
  const carousel = createAtlasBannerCarousel(state => updates.push(state.index))
  carousel.select(1)
  t.mock.timers.tick(20000)
  assert.deepEqual(updates, [])
  carousel.setCount(1)
  t.mock.timers.tick(20000)
  assert.deepEqual(updates, [0])
  carousel.next()
  carousel.previous()
  assert.equal(carousel.state.index, 0)
  carousel.setCount(0)
  t.mock.timers.tick(20000)
  assert.deepEqual(updates, [0, 0, 0, 0])
  carousel.dispose()
})

test('multiple banners advance every five seconds and loop', t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const carousel = createAtlasBannerCarousel(() => {})
  carousel.setCount(3)
  t.mock.timers.tick(4999)
  assert.equal(carousel.state.index, 0)
  for (const index of [1, 2, 0]) {
    t.mock.timers.tick(index === 1 ? 1 : 5000)
    assert.equal(carousel.state.index, index)
  }
  carousel.dispose()
})

test('manual selection, previous and next wrap and reset the full autoplay delay', t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const carousel = createAtlasBannerCarousel(() => {})
  carousel.setCount(3)
  t.mock.timers.tick(4000)
  carousel.previous()
  assert.equal(carousel.state.index, 2)
  t.mock.timers.tick(4999)
  assert.equal(carousel.state.index, 2)
  t.mock.timers.tick(1)
  assert.equal(carousel.state.index, 0)
  carousel.next()
  assert.equal(carousel.state.index, 1)
  carousel.select(8)
  assert.equal(carousel.state.index, 2)
  carousel.select(-7)
  assert.equal(carousel.state.index, 2)
  carousel.select(Number.NaN)
  assert.equal(carousel.state.index, 2)
  carousel.dispose()
})

test('overlapping hover, focus and hidden pauses resume only after all conditions clear', t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const carousel = createAtlasBannerCarousel(() => {})
  carousel.setCount(3)
  for (const reason of ['hover', 'focus', 'hidden'] as const) carousel.setPaused(reason, true)
  carousel.setPaused('hover', true)
  carousel.setPaused('hover', false)
  carousel.setPaused('focus', false)
  t.mock.timers.tick(20000)
  assert.equal(carousel.state.index, 0)
  assert.equal(carousel.state.paused, true)
  carousel.setPaused('hidden', false)
  t.mock.timers.tick(4999)
  assert.equal(carousel.state.index, 0)
  t.mock.timers.tick(1)
  assert.equal(carousel.state.index, 1)
  carousel.dispose()
})

test('manual pause survives navigation and environmental changes; resume still respects focus', t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const carousel = createAtlasBannerCarousel(() => {})
  carousel.setCount(2)
  carousel.setPaused('manual', true)
  carousel.next()
  carousel.setPaused('hidden', true)
  carousel.setPaused('hidden', false)
  t.mock.timers.tick(10000)
  assert.equal(carousel.state.index, 1)
  assert.equal(carousel.state.manualPaused, true)
  carousel.setPaused('focus', true)
  carousel.setPaused('manual', false)
  t.mock.timers.tick(10000)
  assert.equal(carousel.state.index, 1)
  carousel.setPaused('focus', false)
  t.mock.timers.tick(5000)
  assert.equal(carousel.state.index, 0)
  carousel.dispose()
})

test('reduced motion permits manual navigation and never autoplays', t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const carousel = createAtlasBannerCarousel(() => {})
  carousel.setPaused('reduced-motion', true)
  carousel.setCount(2)
  t.mock.timers.tick(10000)
  assert.equal(carousel.state.reducedMotion, true)
  assert.equal(carousel.state.index, 0)
  carousel.next()
  t.mock.timers.tick(10000)
  assert.equal(carousel.state.index, 1)
  carousel.setPaused('reduced-motion', false)
  t.mock.timers.tick(5000)
  assert.equal(carousel.state.index, 0)
  carousel.dispose()
})

test('reload resets the index; disposal cancels timers and ignores late changes', t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  let updates = 0
  const carousel = createAtlasBannerCarousel(() => { updates += 1 })
  carousel.setCount(3)
  carousel.select(2)
  carousel.setCount(2)
  assert.equal(carousel.state.index, 0)
  carousel.setCount(-1)
  carousel.setCount(Number.NaN)
  carousel.setCount(2)
  carousel.dispose()
  const before = updates
  t.mock.timers.tick(30000)
  carousel.setCount(5)
  carousel.select(1)
  carousel.setPaused('manual', true)
  carousel.dispose()
  assert.equal(updates, before)
})

test('environment observes initial viewport, motion and visibility and cleans up every listener', t => {
  const desktop = Object.assign(new EventTarget(), { matches: false })
  const motion = Object.assign(new EventTarget(), { matches: false })
  const document = Object.assign(new EventTarget(), { hidden: false })
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window')
  const originalDocument = Object.getOwnPropertyDescriptor(globalThis, 'document')
  const queries: string[] = []
  Object.defineProperty(globalThis, 'window', { configurable: true, value: { matchMedia(query: string) {
    queries.push(query)
    return query.includes('min-width') ? desktop : motion
  } } })
  Object.defineProperty(globalThis, 'document', { configurable: true, value: document })
  t.after(() => {
    if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow)
    else Reflect.deleteProperty(globalThis, 'window')
    if (originalDocument) Object.defineProperty(globalThis, 'document', originalDocument)
    else Reflect.deleteProperty(globalThis, 'document')
  })
  const states: unknown[] = []
  const stop = observeAtlasBannerEnvironment({
    desktop: value => states.push(['desktop', value]),
    reducedMotion: value => states.push(['motion', value]),
    hidden: value => states.push(['hidden', value]),
  })
  assert.deepEqual(queries, ['(min-width: 768px)', '(prefers-reduced-motion: reduce)'])
  assert.deepEqual(states, [['desktop', false], ['motion', false], ['hidden', false]])
  desktop.matches = true
  desktop.dispatchEvent(new Event('change'))
  motion.matches = true
  motion.dispatchEvent(new Event('change'))
  document.hidden = true
  document.dispatchEvent(new Event('visibilitychange'))
  assert.deepEqual(states.slice(3), [['desktop', true], ['motion', true], ['hidden', true]])
  stop()
  stop()
  desktop.dispatchEvent(new Event('change'))
  motion.dispatchEvent(new Event('change'))
  document.dispatchEvent(new Event('visibilitychange'))
  assert.equal(states.length, 6)
})
