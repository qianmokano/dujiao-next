import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, reactive, type App } from 'vue'
import TurnstileCaptcha from '../src/components/captcha/TurnstileCaptcha.vue'

let app: App | undefined
const scriptSelector = 'script[data-turnstile="1"]'
const flush = async () => { await Promise.resolve(); await Promise.resolve(); await nextTick() }
afterEach(() => {
  app?.unmount(); app = undefined
  document.body.innerHTML = ''
  document.querySelectorAll(scriptSelector).forEach(script => script.remove())
  delete window.turnstile
})

describe('site and Passport Turnstile widgets', () => {
  it('shares the SDK while keeping tokens, reset and removal separate', async () => {
    const values = reactive({site:'', passport:''})
    const widgets: any[] = []
    const root = document.createElement('div'); document.body.appendChild(root)
    app = createApp({render: () => h('div', [
      h(TurnstileCaptcha, {siteKey:'site-key', ref:(value: any) => {widgets[0]=value}, 'onUpdate:modelValue':(value: string) => {values.site=value}}),
      h(TurnstileCaptcha, {siteKey:'passport-key', ref:(value: any) => {widgets[1]=value}, 'onUpdate:modelValue':(value: string) => {values.passport=value}}),
    ])})
    app.mount(root)
    expect(document.querySelectorAll(scriptSelector)).toHaveLength(1)
    const render = vi.fn().mockReturnValueOnce('site-widget').mockReturnValueOnce('passport-widget')
    const reset = vi.fn(); const remove = vi.fn()
    window.turnstile = {render, reset, remove}
    document.querySelector(scriptSelector)!.dispatchEvent(new Event('load'))
    await flush()
    expect(render).toHaveBeenCalledTimes(2)
    render.mock.calls[0]![1].callback('site-proof')
    render.mock.calls[1]![1].callback('passport-proof')
    expect(values).toEqual({site:'site-proof',passport:'passport-proof'})
    widgets[1].reset()
    expect(reset).toHaveBeenCalledWith('passport-widget')
    expect(values).toEqual({site:'site-proof',passport:''})
    render.mock.calls[0]![1]['expired-callback']()
    expect(values.site).toBe('')
    app.unmount(); app = undefined
    expect(remove).toHaveBeenCalledWith('site-widget')
    expect(remove).toHaveBeenCalledWith('passport-widget')
  })

  it('removes a failed SDK script so a fresh challenge can retry', async () => {
    const root = document.createElement('div'); document.body.appendChild(root)
    const answer = vi.fn()
    app = createApp({render:() => h(TurnstileCaptcha,{siteKey:'passport-key','onUpdate:modelValue':answer})})
    app.mount(root)
    document.querySelector(scriptSelector)!.dispatchEvent(new Event('error'))
    await flush()
    expect(answer).toHaveBeenCalledWith('')
    expect(document.querySelector(scriptSelector)).toBeNull()
  })
})
