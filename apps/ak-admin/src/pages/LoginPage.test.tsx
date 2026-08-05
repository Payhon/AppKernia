// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import axe from 'axe-core'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import { authSession, useAuthStore } from '../features/auth/store'
import { ApiError } from '../shared/api/error'
import { LocaleProvider } from '../shared/i18n'
import { LoginPage } from './LoginPage'

vi.mock('@tanstack/react-router', () => ({ useNavigate: () => vi.fn() }))
vi.mock('../components/AuthFrame', () => ({
  AuthFrame: ({ children }: { children: React.ReactNode }) => <main>{children}</main>,
}))

beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false, media: query, onchange: null,
      addListener: vi.fn(), removeListener: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn(),
    })),
  })
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  useAuthStore.setState({ context: null, status: 'anonymous' })
})

describe('LoginPage CAPTCHA progression', () => {
  it('shows the server-issued image challenge only after the third failed sign-in', async () => {
    const login = vi.fn().mockRejectedValueOnce(new ApiError(401, {
      error: { code: 'IAM.AUTH.INVALID_CREDENTIALS', message_key: 'errors.iam.auth.invalid_credentials', message: 'invalid' },
      request_id: 'failed-1',
    })).mockRejectedValueOnce(new ApiError(401, {
      error: { code: 'IAM.AUTH.INVALID_CREDENTIALS', message_key: 'errors.iam.auth.invalid_credentials', message: 'invalid' },
      request_id: 'failed-2',
    })).mockRejectedValueOnce(new ApiError(401, {
      error: { code: 'IAM.AUTH.CAPTCHA_REQUIRED', message_key: 'errors.iam.auth.captcha_required', message: 'captcha required' },
      request_id: 'failed-3',
    }))
    useAuthStore.setState({ login })
    vi.spyOn(authSession, 'publicConfig').mockResolvedValue({
      locale: 'zh-CN', default_locale: 'zh-CN', supported_locales: ['zh-CN', 'en-US'], feature_flags: {}, settings: {},
    })
    const captchaId = crypto.randomUUID()
    const createCaptcha = vi.spyOn(authSession, 'createLoginCaptcha').mockResolvedValue({
      captcha_id: captchaId, image_base64: 'iVBORw0KGgo=', mime_type: 'image/png', expires_in_seconds: 300,
    })
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={queryClient}><LocaleProvider><LoginPage /></LocaleProvider></QueryClientProvider>)

    fireEvent.change(screen.getByLabelText(/账号|Account/), { target: { value: 'admin@example.test' } })
    fireEvent.change(screen.getByLabelText(/密码|Password/), { target: { value: 'wrong password' } })
    const submit = screen.getByRole('button', { name: /登录|Sign in/ })

    fireEvent.click(submit)
    await waitFor(() => { expect(login).toHaveBeenCalledTimes(1) })
    expect(screen.queryByLabelText(/图形验证码|Image verification/)).toBeNull()

    fireEvent.click(submit)
    await waitFor(() => { expect(login).toHaveBeenCalledTimes(2) })
    expect(screen.queryByLabelText(/图形验证码|Image verification/)).toBeNull()

    fireEvent.click(submit)
    await waitFor(() => { expect(login).toHaveBeenCalledTimes(3) })
    await waitFor(() => { expect(screen.getByLabelText(/图形验证码|Image verification/)).toBeTruthy() })
    expect(screen.getByAltText(/登录图形验证码|Image verification code for sign-in/).getAttribute('src')).toBe('data:image/png;base64,iVBORw0KGgo=')
    const refresh = screen.getByRole('button', { name: /刷新验证码|Refresh image verification/ })
    expect(refresh).toBeTruthy()
    expect(createCaptcha).toHaveBeenCalledWith({ email: 'admin@example.test' })

    const accessibility = await axe.run(document.body)
    expect(accessibility.violations).toEqual([])

    fireEvent.click(refresh)
    await waitFor(() => { expect(createCaptcha).toHaveBeenCalledTimes(2) })
  })
})
