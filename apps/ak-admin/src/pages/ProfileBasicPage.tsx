import { Avatar, Button, Card, Form, Input, Progress, Select, Typography } from 'antd'
import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import type { AdminUpdateMeRequest, SupportedLocale } from '../generated/api/types.gen'
import { useAuthStore } from '../features/auth/store'
import { selfAvatarQueryOptions, selfProfileQueryKey, useSelfAvatarQuery, useSelfProfileQuery } from '../features/profile/hooks'
import { useLocale } from '../shared/i18n'
import { ProfileNavigation } from '../components/ProfileNavigation'

interface ProfileValues {
  display_name: string
  locale: SupportedLocale
  time_zone: string
}

const profileSchema = z.object({
  display_name: z.string().trim().min(1).max(160),
  locale: z.enum(['zh-CN', 'en-US']),
  time_zone: z.string().trim().min(1).max(64).refine(isTimeZone),
})

function isTimeZone(value: string): boolean {
  try {
    Intl.DateTimeFormat(undefined, { timeZone: value })
    return true
  } catch {
    return false
  }
}

function timeZones(): string[] {
  if ('supportedValuesOf' in Intl) return Intl.supportedValuesOf('timeZone')
  return ['UTC', 'Asia/Shanghai', 'America/New_York', 'Europe/London']
}

export function ProfileBasicPage() {
  const { t } = useTranslation()
  const { setLocale } = useLocale()
  const queryClient = useQueryClient()
  const profile = useSelfProfileQuery()
  const updateProfile = useAuthStore((state) => state.updateProfile)
  const uploadAvatar = useAuthStore((state) => state.uploadAvatar)
  const avatarEnabled = useAuthStore((state) => state.context?.feature_flags['avatar_upload'] === true)
  const [feedback, setFeedback] = useState<'success' | 'error' | null>(null)
  const [avatarFeedback, setAvatarFeedback] = useState<'success' | 'invalid' | 'error' | null>(null)
  const [avatarFile, setAvatarFile] = useState<File | null>(null)
  const [avatarProgress, setAvatarProgress] = useState(0)
  const [avatarPreviewURL, setAvatarPreviewURL] = useState<string | null>(null)
  const [storedAvatarURL, setStoredAvatarURL] = useState<string | null>(null)
  const avatarInput = useRef<HTMLInputElement>(null)
  const avatar = useSelfAvatarQuery(profile.data?.avatar_url ?? null)
  const zones = useMemo(() => timeZones().map((value) => ({ label: value, value })), [])
  const { control, handleSubmit, reset, setError, formState } = useForm<ProfileValues>({
    defaultValues: { display_name: '', locale: 'zh-CN', time_zone: 'UTC' },
  })

  useEffect(() => {
    if (profile.data) reset(profile.data)
  }, [profile.data, reset])

  useEffect(() => {
    if (!avatar.data) {
      setStoredAvatarURL(null)
      return
    }
    const next = URL.createObjectURL(avatar.data)
    setStoredAvatarURL(next)
    return () => { URL.revokeObjectURL(next) }
  }, [avatar.data])

  useEffect(() => {
    if (!avatarFile) {
      setAvatarPreviewURL(null)
      return
    }
    const next = URL.createObjectURL(avatarFile)
    setAvatarPreviewURL(next)
    return () => { URL.revokeObjectURL(next) }
  }, [avatarFile])

  const chooseAvatar = (file: File | null) => {
    setAvatarFeedback(null)
    setAvatarProgress(0)
    if (!file) {
      setAvatarFile(null)
      return
    }
    const validType = file.type === 'image/jpeg' || file.type === 'image/png' || file.type === 'image/webp'
    if (!validType || file.size <= 0 || file.size > 5 * 1024 * 1024) {
      setAvatarFile(null)
      setAvatarFeedback('invalid')
      return
    }
    setAvatarFile(file)
  }

  const submitAvatar = async () => {
    if (!avatarFile) return
    setAvatarFeedback(null)
    try {
      const updated = await uploadAvatar(avatarFile, setAvatarProgress)
      queryClient.setQueryData(selfProfileQueryKey, updated)
      if (updated.avatar_url) await queryClient.fetchQuery(selfAvatarQueryOptions(updated.avatar_url))
      setAvatarFile(null)
      setAvatarFeedback('success')
    } catch {
      setAvatarProgress(0)
      setAvatarFeedback('error')
    }
  }

  const submit = handleSubmit(async (values) => {
    setFeedback(null)
    const parsed = profileSchema.safeParse(values)
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0]
        if (field === 'display_name' || field === 'locale' || field === 'time_zone') {
          setError(field, { message: t('validation.invalid', { name: t(`profile.fields.${field}`) }) })
        }
      }
      return
    }
    try {
      const updated = await updateProfile(parsed.data satisfies AdminUpdateMeRequest)
      queryClient.setQueryData(selfProfileQueryKey, updated)
      await setLocale(updated.locale)
      reset(updated)
      setFeedback('success')
    } catch {
      setFeedback('error')
    }
  })

  if (profile.isPending) return <div className="ak-centered-state" aria-live="polite"><span className="ak-loading-indicator" /><span>{t('common.states.loading')}</span></div>
  if (profile.isError) return <div className="ak-page-container"><div className="ak-form-error" role="alert">{t('profile.load_error')}</div><Button onClick={() => { void profile.refetch() }}>{t('common.actions.retry')}</Button></div>

  return (
    <div className="ak-page-container">
      <header className="ak-page-heading">
        <Typography.Title level={1}>{t('routes.profile.basic.title')}</Typography.Title>
        <Typography.Paragraph type="secondary">{t('profile.basic.description')}</Typography.Paragraph>
      </header>
      <ProfileNavigation active="basic" />
      <Card className="ak-settings-card">
        {avatarEnabled ? (
          <section className="ak-avatar-editor" aria-labelledby="profile-avatar-label">
            <div className="ak-avatar-preview">
              <Avatar
                alt={t('profile.avatar.alt', { name: profile.data.display_name })}
                size={88}
                src={avatarPreviewURL ?? storedAvatarURL ?? undefined}
              >
                {profile.data.display_name.slice(0, 1).toUpperCase()}
              </Avatar>
              <div>
                <Typography.Title id="profile-avatar-label" level={2}>{t('profile.avatar.label')}</Typography.Title>
                <Typography.Paragraph type="secondary">{t('profile.avatar.description')}</Typography.Paragraph>
              </div>
            </div>
            <div className="ak-avatar-actions">
              <Button onClick={() => { avatarInput.current?.click() }}>{t('profile.avatar.change')}</Button>
              <input
                accept="image/jpeg,image/png,image/webp"
                hidden
                id="profile-avatar-file"
                onChange={(event) => {
                  chooseAvatar(event.currentTarget.files?.[0] ?? null)
                  event.currentTarget.value = ''
                }}
                ref={avatarInput}
                type="file"
              />
              <Button disabled={!avatarFile} loading={avatarProgress > 0 && avatarProgress < 100} onClick={() => { void submitAvatar() }} type="primary">
                {t('profile.avatar.submit')}
              </Button>
            </div>
            {avatarProgress > 0 ? <Progress aria-label={t('profile.avatar.progress', { percent: avatarProgress })} percent={avatarProgress} size="small" /> : null}
            {avatarFeedback === 'success' ? <div className="ak-form-success" role="status">{t('profile.avatar.success')}</div> : null}
            {avatarFeedback === 'invalid' ? <div className="ak-form-error" role="alert">{t('profile.avatar.invalid')}</div> : null}
            {avatarFeedback === 'error' ? <div className="ak-form-error" role="alert">{t('profile.avatar.upload_error')}</div> : null}
          </section>
        ) : null}
        {feedback === 'success' ? <div className="ak-form-success" role="status">{t('profile.save_success')}</div> : null}
        {feedback === 'error' ? <div className="ak-form-error" role="alert">{t('profile.save_error')}</div> : null}
        <Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}>
          <Form.Item htmlFor="profile-email" label={t('profile.fields.email')}>
            <Input disabled id="profile-email" value={profile.data.email} />
          </Form.Item>
          <Controller control={control} name="display_name" render={({ field, fieldState }) => (
            <Form.Item htmlFor="profile-display-name" label={t('profile.fields.display_name')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
              <Input {...field} autoComplete="name" id="profile-display-name" maxLength={160} />
            </Form.Item>
          )} />
          <Controller control={control} name="locale" render={({ field, fieldState }) => (
            <Form.Item htmlFor="profile-locale" label={t('profile.fields.locale')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
              <Select {...field} id="profile-locale" options={[
                { label: t('common.language.zh-CN'), value: 'zh-CN' },
                { label: t('common.language.en-US'), value: 'en-US' },
              ]} />
            </Form.Item>
          )} />
          <Controller control={control} name="time_zone" render={({ field, fieldState }) => (
            <Form.Item htmlFor="profile-time-zone" label={t('profile.fields.time_zone')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>
              <Select {...field} id="profile-time-zone" options={zones} showSearch virtual />
            </Form.Item>
          )} />
          <Button disabled={!formState.isDirty} htmlType="submit" loading={formState.isSubmitting} type="primary">{t('common.actions.save')}</Button>
        </Form>
      </Card>
    </div>
  )
}
