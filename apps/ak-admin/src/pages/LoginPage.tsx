import { Button, Form, Input, Space, Typography } from 'antd'
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

interface LoginValues { email: string; password: string }

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
  const [submitError, setSubmitError] = useState<string | null>(null)
  const { control, handleSubmit, setError } = useForm<LoginValues>({
    defaultValues: { email: '', password: '' },
  })
  const publicConfig = useQuery({
    queryKey: ['auth', 'public-config', locale],
    queryFn: () => authSession.publicConfig(),
    retry: 1,
    staleTime: 60_000,
  })

  const submit = handleSubmit(async (values) => {
    setSubmitError(null)
    const parsed = loginSchema.safeParse(values)
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0]
        if (field === 'email' || field === 'password') setError(field, { message: t('validation.invalid', { name: t(field === 'email' ? 'auth.login.account' : 'auth.login.password') }) })
      }
      return
    }
    try {
      const context = await login(parsed.data.email, parsed.data.password)
      await setLocale(context.user.locale)
      const redirect = new URLSearchParams(window.location.search).get('redirect')
      await navigate({ to: isSafeInternalRedirect(redirect) ? redirect : '/dashboard', replace: true })
    } catch (error) {
      setSubmitError(error instanceof ApiError && error.status === 401
        ? t('auth.login.error.invalid')
        : t('auth.login.error.service'))
    }
  })

  return (
    <AuthFrame headingKey="auth.login.heading" descriptionKey="auth.login.description">
          {submitError ? <div className="ak-form-error" role="alert">{submitError}</div> : null}
          <Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}>
            <Controller control={control} name="email" render={({ field, fieldState }) => (
              <Form.Item htmlFor="login-email" label={t('auth.login.account')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
                <Input {...field} autoComplete="username" id="login-email" inputMode="email" size="large" />
              </Form.Item>
            )} />
            <Controller control={control} name="password" render={({ field, fieldState }) => (
              <Form.Item htmlFor="login-password" label={t('auth.login.password')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
                <Input.Password {...field} autoComplete="current-password" id="login-password" size="large" />
              </Form.Item>
            )} />
            {publicConfig.data?.feature_flags['password_recovery'] === true
              ? <a className="ak-forgot-link" href="/forgot-password">{t('auth.login.forgot')}</a>
              : null}
            <Button block htmlType="submit" loading={status === 'authenticating'} size="large" type="primary">{t('auth.login.submit')}</Button>
          </Form>
          {publicConfig.data?.feature_flags['admin_registration'] === true
            ? <Space className="ak-auth-secondary-action"><Typography.Text type="secondary">{t('auth.login.register_prompt')}</Typography.Text><a href="/register">{t('auth.login.register')}</a></Space>
            : null}
    </AuthFrame>
  )
}
