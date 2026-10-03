export function safeAdminURL(value: unknown): string {
  if (typeof value !== 'string') return ''
  try {
    const url = new URL(value)
    return (url.protocol === 'https:' || url.protocol === 'http:') && !url.username && !url.password ? url.href : ''
  } catch {
    return ''
  }
}

export function localIdentityEditable(data: unknown): boolean {
  return !!data && typeof data === 'object' && (data as { only_enabled?: unknown }).only_enabled === false
}
