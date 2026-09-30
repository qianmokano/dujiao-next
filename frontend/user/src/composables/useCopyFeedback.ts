import { onScopeDispose, ref, type Ref } from 'vue'
import { copyText } from '../utils/clipboard'
import { createCopyFeedback } from '../utils/feedback'
import { useFeedback } from './useFeedback'

/** Returns true when Atlas handled the request; other templates keep their old path. */
export function useCopyFeedback(errorMessage: () => string, copied: Ref<boolean> = ref(false)) {
  const { isAtlas, transientError } = useFeedback()
  const controller = createCopyFeedback({
    writeText: (text) => copyText(text, { requireSuccess: true }),
    setCopied: (value) => { copied.value = value },
    onError: () => { transientError(errorMessage()) },
  })
  onScopeDispose(controller.dispose)

  const copy = async (text: string): Promise<boolean> => {
    if (!isAtlas) return false
    await controller.copy(text)
    return true
  }

  return { copied, copy, isAtlas }
}
