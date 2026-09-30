export const copyText = async (value: string, options?: { requireSuccess?: boolean }): Promise<void> => {
  if (navigator?.clipboard?.writeText) {
    await navigator.clipboard.writeText(value)
    return
  }
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  try {
    const copied = document.execCommand('copy')
    if (options?.requireSuccess && !copied) throw new Error('Copy failed')
  } finally {
    document.body.removeChild(textarea)
  }
}
