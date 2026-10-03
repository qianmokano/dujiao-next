import test from 'node:test'
import assert from 'node:assert/strict'
import { isUnifiedAuthOnly, safeIdentityURL, useSSOCredentials } from '../src/utils/unifiedAuth.ts'

test('only unified authentication ignores customer local query even if the provider is unavailable', () => {
  for (const enabled of [true, false]) {
    for (const local of ['1', '0', undefined, ['1']]) {
      assert.equal(useSSOCredentials({ enabled, only_enabled: true }, local), true)
    }
  }
})

test('optional unified authentication preserves local display fallback', () => {
  assert.equal(useSSOCredentials({ enabled: true }, undefined), true)
  assert.equal(useSSOCredentials({ enabled: true }, '1'), false)
  assert.equal(useSSOCredentials({ enabled: false }, undefined), false)
  assert.equal(useSSOCredentials(null, undefined), false)
  assert.equal(isUnifiedAuthOnly(undefined), false)
  assert.equal(isUnifiedAuthOnly({ only_enabled: false }), false)
})

test('identity management links require absolute HTTP URLs without credentials', () => {
  assert.equal(safeIdentityURL('https://auth.example.com/login/kano'), 'https://auth.example.com/login/kano')
  assert.equal(safeIdentityURL('http://localhost:8000/forget/store'), 'http://localhost:8000/forget/store')
  for (const value of [undefined, null, 1, '/login', 'javascript:alert(1)', 'data:text/html,test', 'https://user:secret@example.com/']) {
    assert.equal(safeIdentityURL(value), '')
  }
})
