import test from 'node:test'
import assert from 'node:assert/strict'
import { registerHooks } from 'node:module'
import { createRenderer, defineComponent, h } from 'vue'

// Match Vite's extensionless TypeScript resolution for the composables under test.
const resolver = registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.startsWith('.') && !/\.[a-z]+$/i.test(specifier)) {
      return nextResolve(`${specifier}.ts`, context)
    }
    return nextResolve(specifier, context)
  },
})
const { provideAtlasFeedback, useFeedback } = await import('../src/composables/useFeedback.ts')
const { useCopyFeedback } = await import('../src/composables/useCopyFeedback.ts')
const { useToast } = await import('../src/composables/useToast.ts')
resolver.deregister()

// A DOM-free renderer exercises real Vue provide/inject and scope disposal.
const renderer = createRenderer({
  patchProp: () => {},
  insert: () => {},
  remove: () => {},
  createElement: () => ({}),
  createText: () => ({}),
  createComment: () => ({}),
  setText: () => {},
  setElementText: () => {},
  parentNode: () => null,
  nextSibling: () => null,
})

const mountFeedback = (atlas: boolean) => {
  let feedback!: ReturnType<typeof useFeedback>
  let copy!: ReturnType<typeof useCopyFeedback>
  const Owner = defineComponent({ setup() {
    feedback = useFeedback()
    copy = useCopyFeedback(() => 'Copy failed')
    return () => h('div')
  } })
  const Layout = defineComponent({ setup() {
    if (atlas) provideAtlasFeedback()
    return () => h(Owner)
  } })
  const app = renderer.createApp(Layout)
  app.mount({})
  return { feedback, copy, unmount: () => { app.unmount() } }
}

test('Atlas layout context reaches shared descendants without affecting sibling roots', (context) => {
  context.mock.timers.enable({ apis: ['setTimeout'] })
  const atlas = mountFeedback(true)
  const legacy = mountFeedback(false)
  const { toasts, removeToast } = useToast()
  context.after(() => {
    atlas.unmount()
    legacy.unmount()
    toasts.value.forEach(({ id }) => removeToast(id))
  })
  assert.equal(atlas.feedback.isAtlas, true)
  assert.equal(legacy.feedback.isAtlas, false)
  let legacyNotices = 0
  atlas.feedback.success('Saved', () => { assert.fail('Atlas inline success') })
  legacy.feedback.success('Saved', () => { legacyNotices += 1 })
  assert.equal(legacyNotices, 1)
  assert.deepEqual(toasts.value.map(({ type, message }) => [type, message]), [['success', 'Saved']])
  context.mock.timers.tick(3000)
  assert.equal(toasts.value.length, 0)
  atlas.unmount()
  atlas.feedback.success('Late result')
  assert.equal(toasts.value.length, 0)
})

test('scoped copy uses local success, resets its timer, and reports failures only through Toast', async (context) => {
  context.mock.timers.enable({ apis: ['setTimeout'] })
  const navigatorDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'navigator')
  let writes = 0
  let reject = false
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { clipboard: {
    writeText: async () => { writes += 1; if (reject) throw new Error('denied') },
  } } })
  const atlas = mountFeedback(true)
  const legacy = mountFeedback(false)
  const { toasts, removeToast } = useToast()
  context.after(() => {
    atlas.unmount()
    legacy.unmount()
    toasts.value.forEach(({ id }) => removeToast(id))
    if (navigatorDescriptor) Object.defineProperty(globalThis, 'navigator', navigatorDescriptor)
    else Reflect.deleteProperty(globalThis, 'navigator')
  })
  assert.equal(await legacy.copy.copy('legacy'), false)
  assert.equal(writes, 0, 'legacy caller must still own its clipboard operation')
  assert.equal(await atlas.copy.copy('address'), true)
  assert.equal(atlas.copy.copied.value, true)
  assert.equal(toasts.value.length, 0)
  context.mock.timers.tick(1000)
  await atlas.copy.copy('address')
  context.mock.timers.tick(1000)
  assert.equal(atlas.copy.copied.value, true)
  context.mock.timers.tick(1000)
  assert.equal(atlas.copy.copied.value, false)
  reject = true
  await atlas.copy.copy('address')
  assert.equal(atlas.copy.copied.value, false)
  assert.deepEqual(toasts.value.map(({ type, message }) => [type, message]), [['error', 'Copy failed']])
  reject = false
  await atlas.copy.copy('address')
  atlas.unmount()
  assert.equal(atlas.copy.copied.value, false)
  context.mock.timers.tick(3000)
  assert.equal(toasts.value.length, 0)
})
