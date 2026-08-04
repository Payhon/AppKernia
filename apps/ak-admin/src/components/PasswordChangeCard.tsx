import { Button, Card, Form, Input, Typography } from 'antd'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { authSession } from '../features/auth/store'
import { ApiError } from '../shared/api/error'

interface PasswordValues {
  current_password: string
  new_password: string
  confirm_password: string
}

const passwordSchema = z.object({
  current_password: z.string().min(12).max(256),
  new_password: z.string().min(12).max(256),
  confirm_password: z.string().min(12).max(256),
}).refine((value) => value.new_password === value.confirm_password, {
  path: ['confirm_password'],
  message: 'mismatch',
})

export function PasswordChangeCard() {
  const { t } = useTranslation()
  const [feedback, setFeedback] = useState<'success' | 'current' | 'reused' | 'error' | null>(null)
  const { control, handleSubmit, reset, setError, formState } = useForm<PasswordValues>({
    defaultValues: { current_password: '', new_password: '', confirm_password: '' },
  })
  const submit = handleSubmit(async (values) => {
    setFeedback(null)
    const parsed = passwordSchema.safeParse(values)
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0]
        if (field === 'current_password' || field === 'new_password' || field === 'confirm_password') {
          setError(field, {
            message: issue.message === 'mismatch'
              ? t('profile.password.mismatch')
              : t('validation.invalid', { name: t(`profile.password.${field === 'current_password' ? 'current' : field === 'new_password' ? 'new' : 'confirm'}`) }),
          })
        }
      }
      return
    }
    try {
      await authSession.changeSelfPassword({
        current_password: parsed.data.current_password,
        new_password: parsed.data.new_password,
      })
      reset()
      setFeedback('success')
    } catch (error) {
      if (error instanceof ApiError && error.code === 'IAM.PASSWORD.CURRENT_INVALID') setFeedback('current')
      else if (error instanceof ApiError && error.code === 'IAM.PASSWORD.REUSED') setFeedback('reused')
      else setFeedback('error')
    }
  })

  return (
    <section aria-labelledby="change-password-title" className="ak-password-section">
      <Typography.Title id="change-password-title" level={2}>{t('profile.password.title')}</Typography.Title>
      <Card className="ak-password-card">
        <Typography.Paragraph type="secondary">{t('profile.password.description')}</Typography.Paragraph>
        <Typography.Paragraph className="ak-password-requirements">{t('profile.password.requirements')}</Typography.Paragraph>
        {feedback === 'success' ? <div className="ak-form-success" role="status">{t('profile.password.success')}</div> : null}
        {feedback === 'current' ? <div className="ak-form-error" role="alert">{t('profile.password.current_invalid')}</div> : null}
        {feedback === 'reused' ? <div className="ak-form-error" role="alert">{t('profile.password.reused')}</div> : null}
        {feedback === 'error' ? <div className="ak-form-error" role="alert">{t('profile.password.error')}</div> : null}
        <Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}>
          <Controller control={control} name="current_password" render={({ field, fieldState }) => (
            <Form.Item htmlFor="change-current-password" label={t('profile.password.current')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
              <Input.Password {...field} autoComplete="current-password" id="change-current-password" maxLength={256} />
            </Form.Item>
          )} />
          <Controller control={control} name="new_password" render={({ field, fieldState }) => (
            <Form.Item htmlFor="change-new-password" label={t('profile.password.new')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
              <Input.Password {...field} autoComplete="new-password" id="change-new-password" maxLength={256} />
            </Form.Item>
          )} />
          <Controller control={control} name="confirm_password" render={({ field, fieldState }) => (
            <Form.Item htmlFor="change-confirm-password" label={t('profile.password.confirm')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
              <Input.Password {...field} autoComplete="new-password" id="change-confirm-password" maxLength={256} />
            </Form.Item>
          )} />
          <Button htmlType="submit" loading={formState.isSubmitting} type="primary">{t('profile.password.submit')}</Button>
        </Form>
      </Card>
    </section>
  )
}
