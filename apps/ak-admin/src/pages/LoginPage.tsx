import { ReloadOutlined } from '@ant-design/icons'
import { Button, Form, Input, Space, Tooltip, Typography } from 'antd'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { isSafeInternalRedirect } from '../app/route-registry'
import { AuthFrame } from '../components/AuthFrame'
import { authSession, useAuthStore } from '../features/auth/store'
import { ApiError } from '../shared/api/error'
import { useLocale } from '../shared/i18n'

interface LoginValues { email: string; password: string; captcha_answer: string }

interface LoginCaptchaChallenge {
  captcha_id: string
  image_base64: string
  mime_type: string
}

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
  const { clearErrors, control, getValues, handleSubmit, resetField, setError, setFocus } = useForm<LoginValues>({
    defaultValues: { email: '', password: '', captcha_answer: '' },
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
    resetField('captcha_answer')
    clearErrors('captcha_answer')
    try {
      const next = await authSession.createLoginCaptcha({ email })
      setCaptcha(next)
      if (fieldErrorKey) setError('captcha_answer', { message: fieldErrorKey })
      window.requestAnimationFrame(() => { setFocus('captcha_answer') })
    } catch {
      setSubmitErrorKey('auth.login.captcha.error.load')
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
    if (captchaRequired && (!captcha || !/^[2-9]{6}$/.test(values.captcha_answer))) {
      if (!captcha) {
        await loadCaptcha(parsed.data.email)
      } else {
        setError('captcha_answer', { message: 'auth.login.captcha.error.invalid' })
        setFocus('captcha_answer')
      }
      return
    }
    try {
      const context = await login({
        email: parsed.data.email,
        password: parsed.data.password,
        ...(captcha ? { captcha_id: captcha.captcha_id, captcha_answer: values.captcha_answer } : {}),
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
                    resetField('captcha_answer')
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
              <Controller control={control} name="captcha_answer" render={({ field, fieldState }) => (
                <Form.Item
                  htmlFor="login-captcha"
                  label={t('auth.login.captcha.label')}
                  extra={t('auth.login.captcha.description')}
                  {...(fieldState.error ? { help: <span role="alert">{t(fieldState.error.message ?? 'auth.login.captcha.error.invalid')}</span>, validateStatus: 'error' as const } : {})}
                >
                  <div className="ak-login-captcha">
                    <Input
                      {...field}
                      aria-describedby="login-captcha-status"
                      autoComplete="one-time-code"
                      disabled={captchaLoading}
                      id="login-captcha"
                      inputMode="numeric"
                      maxLength={6}
                      placeholder={t('auth.login.captcha.placeholder')}
                      size="large"
                    />
                    <div className="ak-login-captcha-image" aria-live="polite" id="login-captcha-status">
                      {captcha ? <img alt={t('auth.login.captcha.alt')} height="56" src={`data:${captcha.mime_type};base64,${captcha.image_base64}`} width="176" /> : <span role="status">{t('auth.login.captcha.loading')}</span>}
                    </div>
                    <Tooltip title={t('auth.login.captcha.refresh')}>
                      <Button
                        aria-label={t('auth.login.captcha.refresh')}
                        disabled={captchaLoading}
                        icon={<ReloadOutlined />}
                        loading={captchaLoading}
                        onClick={() => { void loadCaptcha(getValues('email')) }}
                        size="large"
                        type="default"
                      />
                    </Tooltip>
                  </div>
                </Form.Item>
              )} />
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
