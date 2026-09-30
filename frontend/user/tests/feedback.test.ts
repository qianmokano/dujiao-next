import test from 'node:test'
import assert from 'node:assert/strict'
import {
  createActionFeedback,
  createCopyFeedback,
  feedbackIconClass,
  feedbackRole,
  resolveToastAppearance,
} from '../src/utils/feedback.ts'

test('only the Atlas storefront receives the Atlas toast appearance', () => {
  assert.equal(resolveToastAppearance('atlas', false), 'atlas')
  assert.equal(resolveToastAppearance('atlas', true), 'default')
  for (const template of ['classic', 'vault', 'unknown']) {
    assert.equal(resolveToastAppearance(template, false), 'default')
    assert.equal(resolveToastAppearance(template, true), 'default')
  }
})

test('feedback uses semantic icons and announces only errors as alerts', () => {
  assert.equal(feedbackIconClass('success'), 'text-success')
  assert.equal(feedbackIconClass('error'), 'text-destructive')
  assert.equal(feedbackIconClass('warning'), 'text-warning')
  assert.equal(feedbackIconClass('info'), 'text-muted-foreground')
  assert.equal(feedbackRole('error'), 'alert')
  for (const level of ['success', 'warning', 'info'] as const) {
    assert.equal(feedbackRole(level), 'status')
  }
})

test('Atlas action results produce a single toast and never execute inline fallbacks', () => {
  const results: Array<[string, string]> = []
  let inlineResults = 0
  const feedback = createActionFeedback({
    atlas: true,
    active: () => true,
    toast: (level, message) => { results.push([level, message]) },
  })
  feedback.success('Saved', () => { inlineResults += 1 })
  feedback.transientError('Copy failed', () => { inlineResults += 1 })
  assert.deepEqual(results, [['success', 'Saved'], ['error', 'Copy failed']])
  assert.equal(inlineResults, 0)
})

test('other templates keep their legacy feedback, including intentionally silent paths', () => {
  const results: string[] = []
  const feedback = createActionFeedback({
    atlas: false,
    active: () => true,
    toast: () => { assert.fail('legacy feedback must not create an Atlas toast') },
  })
  feedback.success('Saved', () => { results.push('inline success') })
  feedback.transientError('Copy failed', () => { results.push('legacy error') })
  feedback.success('No legacy notice')
  feedback.transientError('No legacy notice')
  assert.deepEqual(results, ['inline success', 'legacy error'])
})

test('late action results are ignored after the owning scope is disposed', () => {
  let active = true
  let results = 0
  const feedback = createActionFeedback({
    atlas: true,
    active: () => active,
    toast: () => { results += 1 },
  })
  feedback.success('Saved')
  active = false
  feedback.success('Stale success', () => { assert.fail('stale fallback') })
  feedback.transientError('Stale error', () => { assert.fail('stale fallback') })
  assert.equal(results, 1)
})

const copyFixture = (writeText: (value: string) => Promise<void> = async () => {}) => {
  let nextTimer = 0
  let copied = false
  let errors = 0
  const timers = new Map<number, { callback: () => void; delay: number }>()
  const feedback = createCopyFeedback({
    writeText,
    setCopied: (value) => { copied = value },
    onError: () => { errors += 1 },
    schedule: (callback, delay) => {
      const id = ++nextTimer
      timers.set(id, { callback, delay })
      return id as unknown as ReturnType<typeof setTimeout>
    },
    unschedule: (id) => { timers.delete(id as unknown as number) },
  })
  const expire = () => {
    const pending = [...timers.values()]
    timers.clear()
    pending.forEach(({ callback }) => callback())
  }
  return { feedback, timers, expire, copied: () => copied, errors: () => errors }
}

test('copy success remains beside its button for two seconds', async () => {
  const writes: string[] = []
  const fixture = copyFixture(async (text) => { writes.push(text) })
  await fixture.feedback.copy('address')
  assert.deepEqual(writes, ['address'])
  assert.equal(fixture.copied(), true)
  assert.deepEqual([...fixture.timers.values()].map(({ delay }) => delay), [2000])
  fixture.expire()
  assert.equal(fixture.copied(), false)
  assert.equal(fixture.errors(), 0)
  fixture.feedback.dispose()
})

test('repeated copy replaces its timer instead of allowing the first timeout to clear the result', async () => {
  const fixture = copyFixture()
  await fixture.feedback.copy('first')
  const [firstTimer] = fixture.timers.keys()
  await fixture.feedback.copy('second')
  assert.equal(fixture.timers.size, 1)
  assert.equal(fixture.timers.has(firstTimer!), false)
  assert.equal(fixture.copied(), true)
  fixture.expire()
  assert.equal(fixture.copied(), false)
  fixture.feedback.dispose()
})

test('copy failure clears stale success and reports one error', async () => {
  let reject = false
  const fixture = copyFixture(async () => {
    if (reject) throw new Error('clipboard unavailable')
  })
  await fixture.feedback.copy('first')
  reject = true
  await fixture.feedback.copy('second')
  assert.equal(fixture.copied(), false)
  assert.equal(fixture.timers.size, 0)
  assert.equal(fixture.errors(), 1)
  fixture.feedback.dispose()
})

test('empty copy and copies after disposal do not access the clipboard', async () => {
  let writes = 0
  const fixture = copyFixture(async () => { writes += 1 })
  await fixture.feedback.copy('')
  fixture.feedback.dispose()
  await fixture.feedback.copy('after disposal')
  assert.equal(writes, 0)
  assert.equal(fixture.errors(), 0)
})

test('disposing copy feedback clears the timer and displayed state', async () => {
  const fixture = copyFixture()
  await fixture.feedback.copy('value')
  fixture.feedback.dispose()
  assert.equal(fixture.copied(), false)
  assert.equal(fixture.timers.size, 0)
})

test('an older overlapping copy cannot replace the result of the latest request', async () => {
  const requests: Array<() => void> = []
  const fixture = copyFixture(() => new Promise<void>((resolve) => { requests.push(resolve) }))
  const first = fixture.feedback.copy('first')
  const second = fixture.feedback.copy('second')
  requests[1]!()
  await second
  const [latestTimer] = fixture.timers.keys()
  requests[0]!()
  await first
  assert.equal(fixture.copied(), true)
  assert.deepEqual([...fixture.timers.keys()], [latestTimer])
  fixture.feedback.dispose()
})

test('late clipboard rejection is ignored after disposal', async () => {
  let reject!: (reason: Error) => void
  const fixture = copyFixture(() => new Promise<void>((_resolve, rejectPromise) => { reject = rejectPromise }))
  const pending = fixture.feedback.copy('value')
  fixture.feedback.dispose()
  reject(new Error('late error'))
  await pending
  assert.equal(fixture.copied(), false)
  assert.equal(fixture.errors(), 0)
  assert.equal(fixture.timers.size, 0)
})

test('a superseded clipboard rejection does not erase a newer success', async () => {
  let reject!: (reason: Error) => void
  let requests = 0
  const fixture = copyFixture(() => {
    requests += 1
    return requests === 1
      ? new Promise<void>((_resolve, rejectPromise) => { reject = rejectPromise })
      : Promise.resolve()
  })
  const first = fixture.feedback.copy('first')
  await fixture.feedback.copy('second')
  reject(new Error('older error'))
  await first
  assert.equal(fixture.copied(), true)
  assert.equal(fixture.errors(), 0)
  fixture.feedback.dispose()
})

test('the default copy timer is also cancelled on disposal', async () => {
  const copied: boolean[] = []
  const feedback = createCopyFeedback({
    writeText: async () => {},
    setCopied: (value) => { copied.push(value) },
    onError: () => { assert.fail('unexpected copy error') },
  })
  await feedback.copy('value')
  feedback.dispose()
  assert.deepEqual(copied, [false, true, false])
})
