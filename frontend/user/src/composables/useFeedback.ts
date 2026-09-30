import { inject, onScopeDispose, provide, type InjectionKey } from 'vue'
import { createActionFeedback } from '../utils/feedback'
import { toast } from './useToast'

const atlasFeedbackKey: InjectionKey<{ active: boolean }> = Symbol('atlas-feedback')

export function provideAtlasFeedback(): void {
  const context = { active: true }
  provide(atlasFeedbackKey, context)
  onScopeDispose(() => { context.active = false })
}

/** Shared panels opt into Atlas through their mounted layout, never a global flag. */
export function useFeedback() {
  const context = inject(atlasFeedbackKey, null)
  let active = true
  onScopeDispose(() => { active = false })

  return {
    isAtlas: context !== null,
    ...createActionFeedback({
      atlas: context !== null,
      active: () => active && (context?.active ?? true),
      toast: (level, message) => { toast[level](message) },
    }),
  }
}
