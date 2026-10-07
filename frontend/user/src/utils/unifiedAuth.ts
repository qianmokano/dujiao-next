export interface UnifiedAuthConfig {
  enabled?: boolean
  only_enabled?: boolean
  account_url?: string
  password_reset_url?: string
}

export function isUnifiedAuthOnly(config: UnifiedAuthConfig | null | undefined): boolean {
  return config?.only_enabled === true
}

export function useSSOCredentials(config: UnifiedAuthConfig | null | undefined, localQuery: unknown): boolean {
  return isUnifiedAuthOnly(config) || (config?.enabled === true && localQuery !== '1')
}

export function safeIdentityURL(value: unknown): string {
  if (typeof value !== 'string') return ''
  try {
    const url = new URL(value)
    return (url.protocol === 'https:' || url.protocol === 'http:') && !url.username && !url.password ? url.href : ''
  } catch {
    return ''
  }
}
