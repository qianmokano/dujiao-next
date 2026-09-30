export type FeedbackLevel = 'success' | 'error' | 'warning' | 'info'

export const feedbackIconClass = (level: FeedbackLevel): string => {
  switch (level) {
    case 'success': return 'text-success'
    case 'error': return 'text-destructive'
    case 'warning': return 'text-warning'
    default: return 'text-muted-foreground'
  }
}

export const feedbackRole = (level: FeedbackLevel): 'alert' | 'status' =>
  level === 'error' ? 'alert' : 'status'

export const resolveToastAppearance = (template: string, resellerConsole: boolean): 'atlas' | 'default' =>
  template === 'atlas' && !resellerConsole ? 'atlas' : 'default'

/** Explicit action results; persistent page/form feedback stays with its owner. */
export const createActionFeedback = (options: {
  atlas: boolean
  active: () => boolean
  toast: (level: 'success' | 'error', message: string) => void
}) => {
  const notify = (level: 'success' | 'error', message: string, legacy?: () => void) => {
    if (!options.active()) return
    if (options.atlas) options.toast(level, message)
    else legacy?.()
  }

  return {
    success: (message: string, legacy?: () => void) => notify('success', message, legacy),
    transientError: (message: string, legacy?: () => void) => notify('error', message, legacy),
  }
}

/** A copy result belongs to its button, including repeated or overlapping requests. */
export const createCopyFeedback = (options: {
  writeText: (text: string) => Promise<void>
  setCopied: (copied: boolean) => void
  onError: () => void
  schedule?: (callback: () => void, delay: number) => ReturnType<typeof setTimeout>
  unschedule?: (timer: ReturnType<typeof setTimeout>) => void
}) => {
  const schedule = options.schedule ?? setTimeout
  const unschedule = options.unschedule ?? clearTimeout
  let timer: ReturnType<typeof setTimeout> | null = null
  let disposed = false
  let sequence = 0

  const reset = () => {
    if (timer !== null) unschedule(timer)
    timer = null
    options.setCopied(false)
  }

  const copy = async (text: string): Promise<void> => {
    if (!text || disposed) return
    const request = ++sequence
    try {
      await options.writeText(text)
      if (disposed || request !== sequence) return
      reset()
      options.setCopied(true)
      timer = schedule(() => {
        timer = null
        options.setCopied(false)
      }, 2000)
    } catch {
      if (disposed || request !== sequence) return
      reset()
      options.onError()
    }
  }

  const dispose = () => {
    disposed = true
    sequence += 1
    reset()
  }

  return { copy, dispose }
}
