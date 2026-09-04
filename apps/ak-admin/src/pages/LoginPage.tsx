import { Button, Form, Input, Space, Typography } from 'antd'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { lazy, Suspense, useRef, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { isSafeInternalRedirect } from '../app/route-registry'
import { AuthFrame } from '../components/AuthFrame'
import type {
  AkInteractiveCaptchaHandle,
  AkInteractiveCaptchaResponse,
} from '../components/AkInteractiveCaptcha'
import { authSession, useAuthStore } from '../features/auth/store'
import type { AdminLoginCaptchaResponse } from '../generated/api/types.gen'
import { ApiError } from '../shared/api/error'
import { useLocale } from '../shared/i18n'

const loadInteractiveCaptcha = () => import('../components/AkInteractiveCaptcha')
const AkInteractiveCaptcha = lazy(async () =>
  loadInteractiveCaptcha().then((module) => ({
    default: module.AkInteractiveCaptcha,
  })),
)

interface LoginValues { email: string; password: string }
type LoginCaptchaChallenge = AdminLoginCaptchaResponse['data']

const loginSchema = z.object({
  email: z.email(),
  password: z.string().min(1).max(512),
})

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { locale, setLocale } = useLocale()
  const login = useAuthStore((state) => state.login)
  const status = useAuthStore((state) => state.status)
  const [submitErrorKey, setSubmitErrorKey] = useState<string | null>(null)
  const [captchaRequired, setCaptchaRequired] = useState(false)
  const [captchaLoading, setCaptchaLoading] = useState(false)
  const [captcha, setCaptcha] = useState<LoginCaptchaChallenge | null>(null)
  const [captchaResponse, setCaptchaResponse] = useState<AkInteractiveCaptchaResponse | null>(null)
  const [captchaErrorKey, setCaptchaErrorKey] = useState<string | null>(null)
  const captchaRef = useRef<AkInteractiveCaptchaHandle>(null)
  const { control, getValues, handleSubmit, setError } = useForm<LoginValues>({
    defaultValues: { email: '', password: '' },
  })
  const publicConfig = useQuery({
    queryKey: ['auth', 'public-config', locale],
    queryFn: () => authSession.publicConfig(),
    retry: 1,
    staleTime: 60_000,
  })

  const loadCaptcha = async (email: string, fieldErrorKey?: string) => {
    setCaptchaLoading(true)
    setCaptcha(null)
    setCaptchaResponse(null)
    setCaptchaErrorKey(null)
    const componentRequest = loadInteractiveCaptcha()
    try {
      const [next] = await Promise.all([
        authSession.createLoginCaptcha({ email }),
        componentRequest,
      ])
      setCaptcha(next)
      if (fieldErrorKey) setCaptchaErrorKey(fieldErrorKey)
    } catch {
      const componentLoaded = await componentRequest.then(
        () => true,
        () => false,
      )
      setCaptchaErrorKey(componentLoaded
        ? 'auth.login.captcha.error.load'
        : 'auth.login.error.service')
    } finally {
      setCaptchaLoading(false)
    }
  }

  const submit = handleSubmit(async (values) => {
    setSubmitErrorKey(null)
    const parsed = loginSchema.safeParse(values)
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0]
        if (field === 'email' || field === 'password') setError(field, { message: t('validation.invalid', { name: t(field === 'email' ? 'auth.login.account' : 'auth.login.password') }) })
      }
      return
    }
    if (captchaRequired && (!captcha || !captchaResponse)) {
      if (!captcha) {
        await loadCaptcha(parsed.data.email)
      } else {
        setCaptchaErrorKey('auth.login.captcha.error.required')
        captchaRef.current?.focus()
      }
      return
    }
    try {
      const context = await login({
        email: parsed.data.email,
        password: parsed.data.password,
        ...(captcha && captchaResponse ? {
          captcha: {
            id: captcha.captcha_id,
            response: captchaResponse,
            token: captcha.captcha_token,
          },
        } : {}),
      })
      await setLocale(context.user.locale)
      const redirect = new URLSearchParams(window.location.search).get('redirect')
      await navigate({ to: isSafeInternalRedirect(redirect) ? redirect : '/dashboard', replace: true })
    } catch (error) {
      if (error instanceof ApiError && error.code === 'IAM.AUTH.CAPTCHA_REQUIRED') {
        setCaptchaRequired(true)
        setSubmitErrorKey('auth.login.error.invalid')
        await loadCaptcha(parsed.data.email)
        return
      }
      if (error instanceof ApiError && error.code === 'IAM.AUTH.CAPTCHA_INVALID') {
        setCaptchaRequired(true)
        setSubmitErrorKey(null)
        await loadCaptcha(parsed.data.email, 'auth.login.captcha.error.invalid')
        return
      }
      setSubmitErrorKey(error instanceof ApiError && error.status === 401
        ? 'auth.login.error.invalid'
        : 'auth.login.error.service')
    }
  })

  return (
    <AuthFrame headingKey="auth.login.heading" descriptionKey="auth.login.description">
          {submitErrorKey ? <div className="ak-form-error" role="alert">{t(submitErrorKey)}</div> : null}
          <Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}>
            <Controller control={control} name="email" render={({ field, fieldState }) => (
              <Form.Item htmlFor="login-email" label={t('auth.login.account')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
                <Input {...field} autoComplete="username" id="login-email" inputMode="email" size="large" onChange={(event) => {
                  field.onChange(event)
                  if (captchaRequired) {
                    setCaptcha(null)
                    setCaptchaResponse(null)
                    setCaptchaErrorKey(null)
                    setCaptchaRequired(false)
                  }
                }} />
              </Form.Item>
            )} />
            <Controller control={control} name="password" render={({ field, fieldState }) => (
              <Form.Item htmlFor="login-password" label={t('auth.login.password')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
                <Input.Password {...field} autoComplete="current-password" id="login-password" size="large" />
              </Form.Item>
            )} />
            {captchaRequired ? (
              <Form.Item>
                {captcha ? (
                  <Suspense fallback={<div className="ak-captcha-loading" role="status">{t('common.states.loading')}</div>}>
                    <AkInteractiveCaptcha
                      autoFocus
                      challenge={captcha}
                      disabled={captchaLoading || status === 'authenticating'}
                      error={captchaErrorKey ? t(captchaErrorKey) : undefined}
                      ref={captchaRef}
                      value={captchaResponse}
                      onChange={(response) => {
                        setCaptchaResponse(response)
                        if (response) setCaptchaErrorKey(null)
                      }}
                      onRefresh={() => { void loadCaptcha(getValues('email')) }}
                    />
                  </Suspense>
                ) : captchaLoading ? (
                  <div className="ak-captcha-loading" role="status">{t('common.states.loading')}</div>
                ) : captchaErrorKey ? (
                  <div className="ak-captcha-load-error">
                    <div role="alert">{t(captchaErrorKey)}</div>
                    <Button onClick={() => { void loadCaptcha(getValues('email')) }}>
                      {t('common.actions.retry')}
                    </Button>
                  </div>
                ) : null}
              </Form.Item>
            ) : null}
            {publicConfig.data?.feature_flags['password_recovery'] === true
              ? <a className="ak-forgot-link" href="/forgot-password">{t('auth.login.forgot')}</a>
              : null}
            <Button block disabled={captchaLoading} htmlType="submit" loading={status === 'authenticating'} size="large" type="primary">{t('auth.login.submit')}</Button>
          </Form>
          {publicConfig.data?.feature_flags['admin_registration'] === true
            ? <Space className="ak-auth-secondary-action"><Typography.Text type="secondary">{t('auth.login.register_prompt')}</Typography.Text><a href="/register">{t('auth.login.register')}</a></Space>
            : null}
    </AuthFrame>
  )
}
