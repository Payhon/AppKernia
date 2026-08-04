import { Button, Form, Input, Typography } from 'antd'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { AuthFrame } from '../components/AuthFrame'
import { authSession } from '../features/auth/store'

interface ForgotValues { email: string }
const forgotSchema = z.object({ email: z.email().max(254) })

export function ForgotPasswordPage() {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [retryAfter, setRetryAfter] = useState<number | null>(null)
  const { control, handleSubmit, setError } = useForm<ForgotValues>({ defaultValues: { email: '' } })
  const submit = handleSubmit(async (values) => {
    setSubmitError(null)
    const parsed = forgotSchema.safeParse(values)
    if (!parsed.success) {
      setError('email', { message: t('validation.invalid', { name: t('auth.common.email') }) })
      return
    }
    setSubmitting(true)
    try {
      const result = await authSession.forgotPassword(parsed.data)
      setRetryAfter(result.retry_after_seconds)
    } catch {
      setSubmitError(t('auth.forgot.error'))
    } finally {
      setSubmitting(false)
    }
  })
  if (retryAfter !== null) {
    return <AuthFrame headingKey="auth.forgot.success" descriptionKey="auth.forgot.success_description"><Typography.Paragraph aria-live="polite">{t('auth.forgot.cooldown', { seconds: retryAfter })}</Typography.Paragraph><a href="/login">{t('auth.common.back_to_login')}</a></AuthFrame>
  }
  return (
    <AuthFrame headingKey="auth.forgot.heading" descriptionKey="auth.forgot.description">
      {submitError ? <div className="ak-form-error" role="alert">{submitError}</div> : null}
      <Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}>
        <Controller control={control} name="email" render={({ field, fieldState }) => <Form.Item htmlFor="forgot-email" label={t('auth.common.email')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input {...field} autoComplete="email" id="forgot-email" inputMode="email" size="large" /></Form.Item>} />
        <Button block htmlType="submit" loading={submitting} size="large" type="primary">{t('auth.forgot.submit')}</Button>
      </Form>
      <Typography.Paragraph className="ak-auth-back-link"><a href="/login">{t('auth.common.back_to_login')}</a></Typography.Paragraph>
    </AuthFrame>
  )
}
