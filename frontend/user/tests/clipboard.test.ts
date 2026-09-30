import test from 'node:test'
import assert from 'node:assert/strict'
import { copyText } from '../src/utils/clipboard.ts'

test('clipboard API success and rejection are reported without using the fallback', async (context) => {
  const navigatorDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'navigator')
  const writes: string[] = []
  let reject = false
  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: { clipboard: { writeText: async (value: string) => {
      if (reject) throw new Error('permission denied')
      writes.push(value)
    } } },
  })
  context.after(() => {
    if (navigatorDescriptor) Object.defineProperty(globalThis, 'navigator', navigatorDescriptor)
    else Reflect.deleteProperty(globalThis, 'navigator')
  })
  await copyText('value', { requireSuccess: true })
  assert.deepEqual(writes, ['value'])
  reject = true
  await assert.rejects(copyText('value', { requireSuccess: true }), /permission denied/)
})

test('strict fallback rejects a failed copy and always removes its temporary element', async (context) => {
  const navigatorDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'navigator')
  const documentDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'document')
  let result = true
  let throws = false
  let appended = 0
  let removed = 0
  let focused = 0
  let selected = 0
  const textarea = { value: '', style: {}, focus: () => { focused += 1 }, select: () => { selected += 1 } }
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: {} })
  Object.defineProperty(globalThis, 'document', { configurable: true, value: {
    createElement: () => textarea,
    body: { appendChild: () => { appended += 1 }, removeChild: () => { removed += 1 } },
    execCommand: (command: string) => {
      assert.equal(command, 'copy')
      if (throws) throw new Error('copy unavailable')
      return result
    },
  } })
  context.after(() => {
    if (navigatorDescriptor) Object.defineProperty(globalThis, 'navigator', navigatorDescriptor)
    else Reflect.deleteProperty(globalThis, 'navigator')
    if (documentDescriptor) Object.defineProperty(globalThis, 'document', documentDescriptor)
    else Reflect.deleteProperty(globalThis, 'document')
  })
  await copyText('first', { requireSuccess: true })
  assert.equal(textarea.value, 'first')
  result = false
  await assert.rejects(copyText('second', { requireSuccess: true }), /Copy failed/)
  // Legacy callers preserve their current fallback behavior.
  await copyText('legacy')
  throws = true
  await assert.rejects(copyText('third', { requireSuccess: true }), /copy unavailable/)
  assert.equal(appended, 4)
  assert.equal(removed, 4)
  assert.equal(focused, 4)
  assert.equal(selected, 4)
})
