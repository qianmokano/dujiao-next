export interface AtlasBanner {
  id: number
  title: Record<string, string>
  subtitle: Record<string, string>
  image: string
  link_type: string
  link_value: string
  open_in_new_tab: boolean
}

const localizedStrings = (value: unknown): Record<string, string> => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return Object.fromEntries(Object.entries(value).filter((entry): entry is [string, string] => typeof entry[1] === 'string'))
}

// The public API already filters active dates and sorts by merchant priority.
export function selectAtlasBanners(value: unknown): AtlasBanner[] {
  if (!Array.isArray(value)) return []
  const banners: AtlasBanner[] = []
  const ids = new Set<number>()
  for (const row of value) {
    if (!row || typeof row !== 'object' || !Number.isInteger(row.id) || row.id <= 0 || ids.has(row.id)) continue
    const title = localizedStrings(row.title)
    const subtitle = localizedStrings(row.subtitle)
    const image = typeof row.image === 'string' ? row.image.trim() : ''
    if (!image && !Object.values(title).some(text => text.trim()) && !Object.values(subtitle).some(text => text.trim())) continue
    banners.push({
      id: row.id, title, subtitle, image,
      link_type: typeof row.link_type === 'string' ? row.link_type : 'none',
      link_value: typeof row.link_value === 'string' ? row.link_value.trim() : '',
      open_in_new_tab: row.open_in_new_tab === true,
    })
    ids.add(row.id)
    if (banners.length === 5) break
  }
  return banners
}

export function resolveAtlasBannerTarget(banner: AtlasBanner | null) {
  const value = banner?.link_type === 'none' ? '' : banner?.link_value || ''
  if (!value) return { type: 'plans' as const }
  if (/^https?:\/\//i.test(value) || banner?.open_in_new_tab) {
    return { type: 'window' as const, value, target: banner?.open_in_new_tab ? '_blank' : '_self' }
  }
  return { type: 'router' as const, value }
}

type PauseReason = 'hover' | 'focus' | 'hidden' | 'manual' | 'reduced-motion'
export interface AtlasCarouselState {
  index: number
  paused: boolean
  manualPaused: boolean
  reducedMotion: boolean
}

export function createAtlasBannerCarousel(onChange: (state: AtlasCarouselState) => void) {
  let index = 0
  let count = 0
  let disposed = false
  let timer: ReturnType<typeof setTimeout> | undefined
  const pauses = new Set<PauseReason>()
  const state = (): AtlasCarouselState => ({
    index, paused: pauses.size > 0,
    manualPaused: pauses.has('manual'), reducedMotion: pauses.has('reduced-motion'),
  })
  const restart = () => {
    clearTimeout(timer)
    timer = undefined
    if (disposed || count <= 1 || pauses.size) return
    timer = setTimeout(() => {
      index = (index + 1) % count
      onChange(state())
      restart()
    }, 5000)
  }
  const select = (next: number) => {
    if (disposed || !count || !Number.isFinite(next)) return
    index = ((Math.trunc(next) % count) + count) % count
    onChange(state())
    restart()
  }
  return {
    get state() { return state() },
    setCount(next: number) {
      if (disposed) return
      count = Number.isFinite(next) ? Math.max(0, Math.trunc(next)) : 0
      index = 0
      onChange(state())
      restart()
    },
    select,
    next: () => select(index + 1),
    previous: () => select(index - 1),
    setPaused(reason: PauseReason, paused: boolean) {
      if (disposed || pauses.has(reason) === paused) return
      if (paused) pauses.add(reason)
      else pauses.delete(reason)
      onChange(state())
      restart()
    },
    dispose() {
      disposed = true
      clearTimeout(timer)
      timer = undefined
    },
  }
}

export function observeAtlasBannerEnvironment(callbacks: {
  desktop: (value: boolean) => void
  reducedMotion: (value: boolean) => void
  hidden: (value: boolean) => void
}) {
  const desktop = window.matchMedia('(min-width: 768px)')
  const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
  const updateDesktop = () => callbacks.desktop(desktop.matches)
  const updateMotion = () => callbacks.reducedMotion(reducedMotion.matches)
  const updateVisibility = () => callbacks.hidden(document.hidden)
  desktop.addEventListener('change', updateDesktop)
  reducedMotion.addEventListener('change', updateMotion)
  document.addEventListener('visibilitychange', updateVisibility)
  updateDesktop()
  updateMotion()
  updateVisibility()
  return () => {
    desktop.removeEventListener('change', updateDesktop)
    reducedMotion.removeEventListener('change', updateMotion)
    document.removeEventListener('visibilitychange', updateVisibility)
  }
}
