import { Alert, Button, Card, Checkbox, Drawer, Form, Input, Select, Skeleton, Space, Tag, Typography } from 'antd'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import type { AdminStepUpRequestWritable } from '../generated/api/types.gen'
import { useAuthStore } from '../features/auth/store'
import { useMfaMutations, useSelfMfa } from '../features/identity-security/hooks'

const codeSchema = z.object({ code: z.string().regex(/^\d{6}$/) })
const stepUpSchema = z.object({ method: z.enum(['password', 'totp']), proof: z.string().min(6).max(256) })
type CodeValues = z.infer<typeof codeSchema>
type StepUpValues = z.infer<typeof stepUpSchema>

export function MfaSecurityCard() {
  const { t } = useTranslation()
  const enabled = useAuthStore((state) => state.context?.feature_flags['mfa'] === true)
  const permitted = useAuthStore((state) => state.context?.permissions.includes('iam.mfa.manage_self') ?? false)
  const status = useSelfMfa()
  const mutations = useMfaMutations()
  const [action, setAction] = useState<'disable' | 'rotate' | null>(null)
  const [acknowledged, setAcknowledged] = useState(false)
  const codeForm = useForm<CodeValues>({ defaultValues: { code: '' } })
  const stepUpForm = useForm<StepUpValues>({ defaultValues: { method: 'password', proof: '' } })
  if (!enabled || !permitted) return null

  const recoveryCodes = mutations.verify.data?.codes ?? mutations.rotate.data?.codes
  const closeDisclosure = () => {
    if (!acknowledged) return
    mutations.verify.reset()
    mutations.rotate.reset()
    setAcknowledged(false)
  }
  const verify = codeForm.handleSubmit(async (values) => {
    const parsed = codeSchema.safeParse(values)
    if (!parsed.success) { codeForm.setError('code', {}); return }
    try {
      await mutations.verify.mutateAsync(parsed.data)
      mutations.enroll.reset()
      codeForm.reset()
      setAcknowledged(false)
    } catch {
      // The mutation state drives translated, non-sensitive feedback.
    }
  })
  const stepUp = stepUpForm.handleSubmit(async (values) => {
    const parsed = stepUpSchema.safeParse(values)
    if (!parsed.success) { stepUpForm.setError('proof', {}); return }
    const input = parsed.data satisfies AdminStepUpRequestWritable
    try {
      if (action === 'disable') await mutations.disable.mutateAsync(input)
      if (action === 'rotate') await mutations.rotate.mutateAsync(input)
      setAction(null)
      stepUpForm.reset()
      setAcknowledged(false)
    } catch {
      // Keep the drawer open so the user can correct the step-up proof.
    }
  })

  return (
    <section aria-labelledby="mfa-title" className="ak-security-section">
      <Typography.Title id="mfa-title" level={2}>{t('profile.mfa.title')}</Typography.Title>
      <Typography.Paragraph type="secondary">{t('profile.mfa.description')}</Typography.Paragraph>
      {status.isPending ? <Card><Skeleton active paragraph={{ rows: 3 }} /></Card> : null}
      {status.isError ? <Alert action={<Button onClick={() => { void status.refetch() }} size="small">{t('common.actions.retry')}</Button>} role="alert" showIcon title={t('profile.mfa.load_error')} type="error" /> : null}
      {status.data ? (
        <Card>
          <Space orientation="vertical" size="middle">
            <Space wrap>
              <Typography.Text strong>{t('profile.mfa.authenticator')}</Typography.Text>
              <Tag {...(status.data.totp_enabled ? { className: 'ak-status-tag-success' } : {})}>{t(status.data.totp_enabled ? 'profile.mfa.enabled' : 'profile.mfa.disabled')}</Tag>
            </Space>
            <Typography.Text type="secondary">{t('profile.mfa.recovery_remaining', { count: status.data.recovery_codes_remaining })}</Typography.Text>
            {status.data.totp_enabled ? (
              <Space wrap>
                <Button onClick={() => { setAction('rotate') }}>{t('profile.mfa.rotate_codes')}</Button>
                <Button danger onClick={() => { setAction('disable') }}>{t('profile.mfa.disable')}</Button>
              </Space>
            ) : <Button loading={mutations.enroll.isPending} onClick={() => { mutations.enroll.mutate() }} type="primary">{t('profile.mfa.enroll')}</Button>}
          </Space>
        </Card>
      ) : null}
      {mutations.enroll.isError || mutations.verify.isError || mutations.disable.isError || mutations.rotate.isError ? <Alert className="ak-session-feedback" role="alert" showIcon title={t('profile.mfa.action_error')} type="error" /> : null}
      {mutations.disable.isSuccess ? <Alert className="ak-session-feedback" role="status" showIcon title={t('profile.mfa.disable_success')} type="success" /> : null}

      <Drawer destroyOnHidden open={Boolean(mutations.enroll.data) && !recoveryCodes} onClose={() => { mutations.enroll.reset(); codeForm.reset() }} size="large" title={t('profile.mfa.enrollment_title')}>
        {mutations.enroll.data ? (
          <Space orientation="vertical" size="large" style={{ width: '100%' }}>
            <Alert showIcon title={t('profile.mfa.secret_once')} type="warning" />
            <div><Typography.Text strong>{t('profile.mfa.secret_label')}</Typography.Text><Typography.Paragraph copyable={{ text: mutations.enroll.data.secret }}><code>{mutations.enroll.data.secret}</code></Typography.Paragraph></div>
            <Typography.Paragraph copyable={{ text: mutations.enroll.data.otpauth_uri }} ellipsis={{ rows: 2, expandable: true }}><code>{mutations.enroll.data.otpauth_uri}</code></Typography.Paragraph>
            <Form layout="vertical" onFinish={() => { void verify() }}>
              <Controller control={codeForm.control} name="code" render={({ field, fieldState }) => <Form.Item {...(fieldState.error ? { help: t('profile.mfa.code_invalid'), validateStatus: 'error' as const } : {})} label={t('profile.mfa.code_label')}><Input {...field} aria-label={t('profile.mfa.code_label')} autoComplete="one-time-code" inputMode="numeric" maxLength={6} /></Form.Item>} />
              <Button htmlType="submit" loading={mutations.verify.isPending} type="primary">{t('profile.mfa.verify')}</Button>
            </Form>
          </Space>
        ) : null}
      </Drawer>

      <Drawer destroyOnHidden extra={<Button disabled={!acknowledged} onClick={closeDisclosure} type="primary">{t('profile.mfa.saved_codes')}</Button>} mask={{ closable: false }} onClose={closeDisclosure} open={Boolean(recoveryCodes)} size="large" title={t('profile.mfa.recovery_title')}>
        <Alert showIcon title={t('profile.mfa.recovery_once')} type="warning" />
        <ul className="ak-recovery-code-list">{recoveryCodes?.map((code) => <li key={code}><code>{code}</code></li>)}</ul>
        <Checkbox checked={acknowledged} onChange={(event) => { setAcknowledged(event.target.checked) }}>{t('profile.mfa.recovery_ack')}</Checkbox>
      </Drawer>

      <Drawer destroyOnHidden onClose={() => { setAction(null); stepUpForm.reset() }} open={action !== null} size={480} title={t(action === 'disable' ? 'profile.mfa.disable_title' : 'profile.mfa.rotate_title')}>
        <Alert showIcon title={t('profile.mfa.step_up_notice')} type={action === 'disable' ? 'warning' : 'info'} />
        <Form layout="vertical" onFinish={() => { void stepUp() }}>
          <Controller control={stepUpForm.control} name="method" render={({ field }) => <Form.Item label={t('profile.mfa.step_up_method')}><Select {...field} aria-label={t('profile.mfa.step_up_method')} options={(['password', 'totp'] as const).map((value) => ({ value, label: t(`profile.mfa.step_up.${value}`) }))} /></Form.Item>} />
          <Controller control={stepUpForm.control} name="proof" render={({ field, fieldState }) => <Form.Item {...(fieldState.error ? { help: t('profile.mfa.step_up_invalid'), validateStatus: 'error' as const } : {})} label={t('profile.mfa.step_up_proof')}><Input {...field} aria-label={t('profile.mfa.step_up_proof')} autoComplete={stepUpForm.watch('method') === 'password' ? 'current-password' : 'one-time-code'} {...(stepUpForm.watch('method') === 'totp' ? { inputMode: 'numeric' as const } : {})} type={stepUpForm.watch('method') === 'password' ? 'password' : 'text'} /></Form.Item>} />
          <Button danger={action === 'disable'} htmlType="submit" loading={mutations.disable.isPending || mutations.rotate.isPending} type="primary">{t('common.actions.confirm')}</Button>
        </Form>
      </Drawer>
    </section>
  )
}
