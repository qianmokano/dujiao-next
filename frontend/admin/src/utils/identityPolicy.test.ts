import { describe, expect, it } from 'vitest'
import { localIdentityEditable, safeAdminURL } from './identityPolicy'

describe('identity settings policy', () => {
  it('opens local editing only after an explicit local-mode response', () => {
    expect(localIdentityEditable({ only_enabled: false })).toBe(true)
    for (const data of [null, undefined, {}, { only_enabled: true }, { only_enabled: 'false' }, 1]) {
      expect(localIdentityEditable(data)).toBe(false)
    }
  })
  it('shows only absolute browser URLs without embedded credentials', () => {
    expect(safeAdminURL('https://auth.example/login/built-in')).toBe('https://auth.example/login/built-in')
    expect(safeAdminURL('http://localhost:8000/login/built-in')).toBe('http://localhost:8000/login/built-in')
    for (const url of [undefined, null, 1, '/account', 'javascript:alert(1)', 'https://user:secret@auth.example']) {
      expect(safeAdminURL(url)).toBe('')
    }
  })
})
