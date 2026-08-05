// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import { useAuthStore, type AuthContext } from '../features/auth/store'
import { LocaleProvider } from '../shared/i18n'
import { LocaleSwitcher } from './LocaleSwitcher'

const context: AuthContext = {
  user: { id: '019fc551-5b28-78ca-9f5c-422acfc92983', email: 'admin@example.test', display_name: 'Admin', avatar_url: null, locale: 'zh-CN', time_zone: 'UTC' },
  active_tenant: { id: '019fc551-5b27-71d5-92e6-544b0105d3d6', code: 'test', name: 'Test' },
  available_tenants: [{ id: '019fc551-5b27-71d5-92e6-544b0105d3d6', code: 'test', name: 'Test', status: 'active' }],
  roles: [], permissions: [], menus: [], feature_flags: {}, menu_revision: 1, permission_revision: 1,
  server_time: '2026-08-03T00:00:00Z',
}

beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
  class TestResizeObserver {
    observe() { return undefined }
    disconnect() { return undefined }
    unobserve() { return undefined }
  }
  Object.defineProperty(globalThis, 'ResizeObserver', {
    configurable: true,
    value: TestResizeObserver,
  })
})

afterEach(() => {
  cleanup()
  useAuthStore.setState({ context: null, status: 'anonymous' })
})

describe('LocaleSwitcher', () => {
  it('opens an accessible icon menu and changes the anonymous locale', async () => {
    render(<LocaleProvider><LocaleSwitcher variant="icon" /></LocaleProvider>)
    const trigger = screen.getByRole('button', { name: /显示语言|Display language/ })
    expect(trigger.querySelector('.anticon-translation')).not.toBeNull()
    fireEvent.click(trigger)

    const current = document.documentElement.lang
    const nextLocale = current === 'zh-CN' ? 'en-US' : 'zh-CN'
    const nextLabel = nextLocale === 'zh-CN' ? '简体中文' : 'English'
    fireEvent.click(await screen.findByRole('menuitem', { name: nextLabel }))

    await waitFor(() => { expect(document.documentElement.lang).toBe(nextLocale) })
  })

  it('rolls back and announces an authenticated persistence failure', async () => {
    const updateLocale = vi.fn<() => Promise<void>>().mockRejectedValue(new Error('network'))
    useAuthStore.setState({ context, status: 'authenticated', updateLocale })
    render(<LocaleProvider><LocaleSwitcher /></LocaleProvider>)
    const select = screen.getByRole<HTMLSelectElement>('combobox')
    await waitFor(() => { expect(document.documentElement.lang).toBe(select.value) })
    const original = select.value
    const next = original === 'zh-CN' ? 'en-US' : 'zh-CN'

    fireEvent.change(select, { target: { value: next } })

    await waitFor(() => { expect(updateLocale).toHaveBeenCalledWith(next) })
    await waitFor(() => {
      expect(select.value).toBe(original)
      expect(document.documentElement.lang).toBe(original)
    })
    await waitFor(() => {
      expect(screen.getByRole('alert').textContent).toBe(original === 'zh-CN' ? '未保存，请重试' : 'Not saved. Try again.')
    })
  })
})
