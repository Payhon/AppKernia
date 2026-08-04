import { Button, Form, Input, Typography } from 'antd'
import { useEffect, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { AuthFrame } from '../components/AuthFrame'
import { authSession } from '../features/auth/store'
import { ApiError } from '../shared/api/error'

interface ResetValues { confirmPassword: string; newPassword: string }
const resetSchema = z.object({ confirmPassword: z.string().min(12).max(256), newPassword: z.string().min(12).max(256) }).refine((values) => values.newPassword === values.confirmPassword, { path: ['confirmPassword'] })

export function ResetPasswordPage() {
  const { t } = useTranslation()
  const [token] = useState(() => new URLSearchParams(window.location.search).get('token') ?? '')
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [completed, setCompleted] = useState(false)
  const { control, handleSubmit, setError } = useForm<ResetValues>({ defaultValues: { confirmPassword: '', newPassword: '' } })
  useEffect(() => {
    if (window.location.search) window.history.replaceState(window.history.state, '', '/reset-password')
  }, [])
  const submit = handleSubmit(async (values) => {
    setSubmitError(null)
    const parsed = resetSchema.safeParse(values)
    if (!parsed.success) {
      if (parsed.error.issues.some((issue) => issue.path[0] === 'confirmPassword')) setError('confirmPassword', { message: t('auth.reset.mismatch') })
      if (parsed.error.issues.some((issue) => issue.path[0] === 'newPassword')) setError('newPassword', { message: t('validation.invalid', { name: t('auth.reset.new_password') }) })
      return
    }
    setSubmitting(true)
    try {
      await authSession.resetPassword({ token, new_password: parsed.data.newPassword })
      setCompleted(true)
    } catch (error) {
      setSubmitError(error instanceof ApiError && error.code === 'IAM.PASSWORD.RESET_TOKEN_INVALID' ? t('auth.reset.invalid') : t('auth.reset.error'))
    } finally {
      setSubmitting(false)
    }
  })
  if (!token) return <AuthFrame headingKey="auth.reset.heading" descriptionKey="auth.reset.invalid"><a href="/forgot-password">{t('auth.forgot.heading')}</a></AuthFrame>
  if (completed) return <AuthFrame headingKey="auth.reset.success" descriptionKey="auth.reset.success_description"><a href="/login">{t('auth.common.back_to_login')}</a></AuthFrame>
  return (
    <AuthFrame headingKey="auth.reset.heading" descriptionKey="auth.reset.description">
      {submitError ? <div className="ak-form-error" role="alert">{submitError}</div> : null}
      <Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}>
        <Controller control={control} name="newPassword" render={({ field, fieldState }) => <Form.Item htmlFor="reset-password" label={t('auth.reset.new_password')} extra={t('auth.common.password_guidance')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input.Password {...field} autoComplete="new-password" id="reset-password" size="large" /></Form.Item>} />
        <Controller control={control} name="confirmPassword" render={({ field, fieldState }) => <Form.Item htmlFor="reset-confirm-password" label={t('auth.reset.confirm_password')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input.Password {...field} autoComplete="new-password" id="reset-confirm-password" size="large" /></Form.Item>} />
        <Button block htmlType="submit" loading={submitting} size="large" type="primary">{t('auth.reset.submit')}</Button>
      </Form>
      <Typography.Paragraph className="ak-auth-back-link"><a href="/login">{t('auth.common.back_to_login')}</a></Typography.Paragraph>
    </AuthFrame>
  )
}
