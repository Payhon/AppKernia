// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import axe from 'axe-core'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import { authSession, useAuthStore } from '../features/auth/store'
import { AnonymousPage } from '../app/route-boundaries'
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
  it('keeps the login form mounted while authentication is in progress', () => {
    useAuthStore.setState({ context: null, status: 'authenticating' })
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    render(<QueryClientProvider client={queryClient}><AnonymousPage><div>login form</div></AnonymousPage></QueryClientProvider>)

    expect(screen.getByText('login form')).toBeTruthy()
  })

  it('shows and submits the server-issued interactive challenge only after the third failed sign-in', async () => {
    const login = vi.fn().mockRejectedValueOnce(new ApiError(401, {
      error: { code: 'IAM.AUTH.INVALID_CREDENTIALS', message_key: 'errors.iam.auth.invalid_credentials', message: 'invalid' },
      request_id: 'failed-1',
    })).mockRejectedValueOnce(new ApiError(401, {
      error: { code: 'IAM.AUTH.INVALID_CREDENTIALS', message_key: 'errors.iam.auth.invalid_credentials', message: 'invalid' },
      request_id: 'failed-2',
    })).mockRejectedValueOnce(new ApiError(401, {
      error: { code: 'IAM.AUTH.CAPTCHA_REQUIRED', message_key: 'errors.iam.auth.captcha_required', message: 'captcha required' },
      request_id: 'failed-3',
    })).mockRejectedValueOnce(new ApiError(422, {
      error: { code: 'IAM.AUTH.CAPTCHA_INVALID', message_key: 'errors.iam.auth.captcha_invalid', message: 'captcha invalid' },
      request_id: 'failed-4',
    })).mockRejectedValueOnce(new ApiError(401, {
      error: { code: 'IAM.AUTH.INVALID_CREDENTIALS', message_key: 'errors.iam.auth.invalid_credentials', message: 'invalid' },
      request_id: 'failed-5',
    }))
    useAuthStore.setState({ login })
    vi.spyOn(authSession, 'publicConfig').mockResolvedValue({
      locale: 'zh-CN', default_locale: 'zh-CN', supported_locales: ['zh-CN', 'en-US'], feature_flags: {}, settings: {},
    })
    const captchaId = crypto.randomUUID()
    const captchaImage = { base64: 'iVBORw0KGgo=', height: 200, mime_type: 'image/png' as const, width: 300 }
    const createCaptcha = vi.spyOn(authSession, 'createLoginCaptcha')
      .mockResolvedValueOnce({
        captcha_id: captchaId,
        captcha_token: 'captcha-token-1',
        expires_in_seconds: 300,
        image: captchaImage,
        prompt_image: { ...captchaImage, height: 40, width: 80 },
        required_points: 1,
        type: 'click',
      })
      .mockResolvedValueOnce({
        captcha_id: crypto.randomUUID(),
        captcha_token: 'captcha-token-2',
        expires_in_seconds: 300,
        image: captchaImage,
        prompt_image: { ...captchaImage, height: 40, width: 80 },
        required_points: 1,
        type: 'click',
      })
      .mockResolvedValueOnce({
        captcha_id: crypto.randomUUID(),
        captcha_token: 'captcha-token-3',
        expires_in_seconds: 300,
        image: captchaImage,
        prompt_image: { ...captchaImage, height: 40, width: 80 },
        required_points: 1,
        type: 'click',
      })
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={queryClient}><LocaleProvider><LoginPage /></LocaleProvider></QueryClientProvider>)

    fireEvent.change(screen.getByLabelText(/账号|Account/), { target: { value: 'admin@example.test' } })
    fireEvent.change(screen.getByLabelText(/密码|Password/), { target: { value: 'wrong password' } })
    const submit = screen.getByRole('button', { name: /登录|Sign in/ })

    fireEvent.click(submit)
    await waitFor(() => { expect(login).toHaveBeenCalledTimes(1) })
    expect(screen.queryByRole('group', { name: /点选验证图片|Point-selection verification image/ })).toBeNull()

    fireEvent.click(submit)
    await waitFor(() => { expect(login).toHaveBeenCalledTimes(2) })
    expect(screen.queryByRole('group', { name: /点选验证图片|Point-selection verification image/ })).toBeNull()

    fireEvent.click(submit)
    await waitFor(() => { expect(login).toHaveBeenCalledTimes(3) })
    const board = await screen.findByRole(
      'group',
      { name: /点选验证图片|Point-selection verification image/ },
      { timeout: 10_000 },
    )
    vi.spyOn(board, 'getBoundingClientRect').mockReturnValue({
      bottom: 200, height: 200, left: 0, right: 300, toJSON: () => ({}), top: 0, width: 300, x: 0, y: 0,
    })
    fireEvent.pointerDown(board, { clientX: 120, clientY: 80, pointerId: 1 })
    await waitFor(() => { expect(screen.getByText(/互动验证已完成|Interactive verification completed/)).toBeTruthy() })

    fireEvent.click(submit)
    await waitFor(() => { expect(login).toHaveBeenCalledTimes(4) })
    expect(login).toHaveBeenLastCalledWith({
      email: 'admin@example.test',
      password: 'wrong password',
      captcha: {
        id: captchaId,
        response: { points: [{ x: 120, y: 80 }], type: 'click' },
        token: 'captcha-token-1',
      },
    })
    await waitFor(() => { expect(createCaptcha).toHaveBeenCalledTimes(2) })
    expect(await screen.findByText(/互动验证错误或已过期|interactive verification is incorrect or expired/i)).toBeTruthy()

    const refresh = screen.getByRole('button', { name: /刷新互动验证|Refresh interactive verification/ })
    expect(refresh).toBeTruthy()
    expect(createCaptcha).toHaveBeenCalledWith({ email: 'admin@example.test' })

    const accessibility = await axe.run(document.body)
    expect(accessibility.violations).toEqual([])

    fireEvent.click(refresh)
    await waitFor(() => { expect(createCaptcha).toHaveBeenCalledTimes(3) })

    fireEvent.change(screen.getByLabelText(/账号|Account/), { target: { value: 'another@example.test' } })
    expect(screen.queryByRole('group', { name: /点选验证图片|Point-selection verification image/ })).toBeNull()
    fireEvent.click(submit)
    await waitFor(() => { expect(login).toHaveBeenCalledTimes(5) })
    expect(login).toHaveBeenLastCalledWith({
      email: 'another@example.test',
      password: 'wrong password',
    })
    expect(createCaptcha).toHaveBeenCalledTimes(3)
  }, 15_000)
})
