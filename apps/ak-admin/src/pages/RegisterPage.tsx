import { Button, Checkbox, Form, Input, Typography } from 'antd'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { AuthFrame } from '../components/AuthFrame'
import { authSession } from '../features/auth/store'
import { useLocale } from '../shared/i18n'

interface RegisterValues {
  acceptTerms: boolean
  confirmPassword: string
  displayName: string
  email: string
  password: string
}

const registerSchema = z.object({
  acceptTerms: z.literal(true),
  confirmPassword: z.string().min(12).max(256),
  displayName: z.string().trim().min(2).max(120),
  email: z.email().max(254),
  password: z.string().min(12).max(256),
}).refine((values) => values.password === values.confirmPassword, { path: ['confirmPassword'] })

export function RegisterPage() {
  const { t } = useTranslation()
  const { locale } = useLocale()
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [accepted, setAccepted] = useState(false)
  const { control, handleSubmit, setError } = useForm<RegisterValues>({
    defaultValues: { acceptTerms: false, confirmPassword: '', displayName: '', email: '', password: '' },
  })

  const submit = handleSubmit(async (values) => {
    setSubmitError(null)
    const parsed = registerSchema.safeParse(values)
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0]
        if (field === 'acceptTerms') setError(field, { message: t('auth.register.terms_required') })
        else if (field === 'confirmPassword') setError(field, { message: t('auth.register.mismatch') })
        else if (field === 'displayName' || field === 'email' || field === 'password') setError(field, { message: t('validation.invalid', { name: t(`auth.register.${field === 'displayName' ? 'display_name' : field}`) }) })
      }
      return
    }
    setSubmitting(true)
    try {
      await authSession.register({
        accept_terms: true, display_name: parsed.data.displayName, email: parsed.data.email,
        locale, password: parsed.data.password,
      })
      setAccepted(true)
    } catch {
      setSubmitError(t('auth.register.error'))
    } finally {
      setSubmitting(false)
    }
  })

  if (accepted) {
    return <AuthFrame headingKey="auth.register.success" descriptionKey="auth.register.success_description"><Link to="/login">{t('auth.common.back_to_login')}</Link></AuthFrame>
  }
  return (
    <AuthFrame headingKey="auth.register.heading" descriptionKey="auth.register.description">
      {submitError ? <div className="ak-form-error" role="alert">{submitError}</div> : null}
      <Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}>
        <Controller control={control} name="displayName" render={({ field, fieldState }) => <Form.Item htmlFor="register-display-name" label={t('auth.register.display_name')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input {...field} autoComplete="name" id="register-display-name" size="large" /></Form.Item>} />
        <Controller control={control} name="email" render={({ field, fieldState }) => <Form.Item htmlFor="register-email" label={t('auth.common.email')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input {...field} autoComplete="email" id="register-email" inputMode="email" size="large" /></Form.Item>} />
        <Controller control={control} name="password" render={({ field, fieldState }) => <Form.Item htmlFor="register-password" label={t('auth.register.password')} extra={t('auth.common.password_guidance')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input.Password {...field} autoComplete="new-password" id="register-password" size="large" /></Form.Item>} />
        <Controller control={control} name="confirmPassword" render={({ field, fieldState }) => <Form.Item htmlFor="register-confirm-password" label={t('auth.register.confirm_password')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input.Password {...field} autoComplete="new-password" id="register-confirm-password" size="large" /></Form.Item>} />
        <Controller control={control} name="acceptTerms" render={({ field: { value, onChange }, fieldState }) => <Form.Item {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Checkbox checked={value} onChange={(event) => { onChange(event.target.checked) }}>{t('auth.register.terms')}</Checkbox></Form.Item>} />
        <Button block htmlType="submit" loading={submitting} size="large" type="primary">{t('auth.register.submit')}</Button>
      </Form>
      <Typography.Paragraph className="ak-auth-back-link"><Link to="/login">{t('auth.common.back_to_login')}</Link></Typography.Paragraph>
    </AuthFrame>
  )
}
