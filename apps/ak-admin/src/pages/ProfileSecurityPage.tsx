import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Card, Descriptions, Empty, Popconfirm, Skeleton, Space, Tag, Typography } from 'antd'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { ProfileNavigation } from '../components/ProfileNavigation'
import { PasswordChangeCard } from '../components/PasswordChangeCard'
import { MfaSecurityCard } from '../components/MfaSecurityCard'
import type { AdminSelfDevice, AdminSelfSession } from '../generated/api/types.gen'
import { useAuthStore } from '../features/auth/store'
import {
  selfDevicesQueryKey,
  selfSessionsQueryKey,
  useSelfDevicesQuery,
  useSelfSessionsQuery,
} from '../features/profile/hooks'

export function ProfileSecurityPage() {
  const { t, i18n } = useTranslation()
  const queryClient = useQueryClient()
  const sessions = useSelfSessionsQuery()
  const devices = useSelfDevicesQuery()
  const revokeSelfSession = useAuthStore((state) => state.revokeSelfSession)
  const removeSelfDevice = useAuthStore((state) => state.removeSelfDevice)
  const locale = i18n.resolvedLanguage === 'en-US' ? 'en-US' : 'zh-CN'
  const formatter = useMemo(() => new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium', timeStyle: 'short',
  }), [locale])
  const revoke = useMutation({
    mutationFn: revokeSelfSession,
    onSuccess: async (result) => {
      if (result.current_session) {
        window.location.assign('/login')
        return
      }
      await queryClient.invalidateQueries({ queryKey: selfSessionsQueryKey })
    },
  })
  const removeDevice = useMutation({
    mutationFn: removeSelfDevice,
    onSuccess: async (result) => {
      if (result.current_device) {
        window.location.assign('/login')
        return
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: selfDevicesQueryKey }),
        queryClient.invalidateQueries({ queryKey: selfSessionsQueryKey }),
      ])
    },
  })

  return (
    <div className="ak-page-container">
      <header className="ak-page-heading">
        <Typography.Title level={1}>{t('routes.profile.security.title')}</Typography.Title>
        <Typography.Paragraph type="secondary">{t('profile.security.description')}</Typography.Paragraph>
      </header>
      <ProfileNavigation active="security" />
      <PasswordChangeCard />
      <MfaSecurityCard />
      {removeDevice.isSuccess && !removeDevice.data.current_device
        ? <Alert className="ak-session-feedback" showIcon title={t('profile.devices.remove_success')} type="success" />
        : null}
      {removeDevice.isError
        ? <Alert className="ak-session-feedback" role="alert" showIcon title={t('profile.devices.remove_error')} type="error" />
        : null}
      <section aria-labelledby="registered-devices-title" className="ak-security-section">
        <Typography.Title id="registered-devices-title" level={2}>{t('profile.devices.title')}</Typography.Title>
        <Typography.Paragraph type="secondary">{t('profile.devices.description')}</Typography.Paragraph>
        {devices.isPending ? <Card><Skeleton active paragraph={{ rows: 4 }} /></Card> : null}
        {devices.isError ? (
          <Alert
            action={<Button onClick={() => { void devices.refetch() }} size="small">{t('common.actions.retry')}</Button>}
            role="alert"
            showIcon
            title={t('profile.devices.load_error')}
            type="error"
          />
        ) : null}
        {devices.data?.length === 0 ? <Card><Empty description={t('profile.devices.empty')} /></Card> : null}
        <div className="ak-session-grid">
          {devices.data?.map((device) => (
            <DeviceCard
              device={device}
              formatDate={(value) => formatter.format(new Date(value))}
              key={device.id}
              onRemove={() => { removeDevice.mutate(device.id) }}
              pending={removeDevice.isPending && removeDevice.variables === device.id}
            />
          ))}
        </div>
      </section>
      {revoke.isSuccess && !revoke.data.current_session
        ? <Alert className="ak-session-feedback" showIcon title={t('profile.sessions.revoke_success')} type="success" />
        : null}
      {revoke.isError
        ? <Alert className="ak-session-feedback" role="alert" showIcon title={t('profile.sessions.revoke_error')} type="error" />
        : null}
      <section aria-labelledby="active-sessions-title" className="ak-security-section">
        <Typography.Title id="active-sessions-title" level={2}>{t('profile.sessions.title')}</Typography.Title>
        {sessions.isPending ? <Card><Skeleton active paragraph={{ rows: 4 }} /></Card> : null}
        {sessions.isError ? (
          <Alert
            action={<Button onClick={() => { void sessions.refetch() }} size="small">{t('common.actions.retry')}</Button>}
            role="alert"
            showIcon
            title={t('profile.sessions.load_error')}
            type="error"
          />
        ) : null}
        {sessions.data?.length === 0 ? <Card><Empty description={t('profile.sessions.empty')} /></Card> : null}
        <div className="ak-session-grid">
          {sessions.data?.map((session) => (
            <SessionCard
              formatDate={(value) => formatter.format(new Date(value))}
              key={session.id}
              onRevoke={() => { revoke.mutate(session.id) }}
              pending={revoke.isPending && revoke.variables === session.id}
              session={session}
            />
          ))}
        </div>
      </section>
    </div>
  )
}

function DeviceCard({ device, formatDate, onRemove, pending }: {
  device: AdminSelfDevice
  formatDate: (value: string) => string
  onRemove: () => void
  pending: boolean
}) {
  const { t } = useTranslation()
  const title = device.device_name || device.latest_user_agent || t('profile.devices.unknown')
  return (
    <Card className="ak-session-card">
      <Space align="start" className="ak-session-card-heading">
        <Typography.Text strong>{title}</Typography.Text>
        <Space wrap>
          <Tag>{t(`profile.devices.platform.${device.platform}`)}</Tag>
          {device.current ? <Tag color="blue">{t('profile.devices.current')}</Tag> : null}
        </Space>
      </Space>
      <Descriptions column={{ xs: 1, sm: 2 }} size="small">
        <Descriptions.Item label={t('profile.devices.ip_address')}>{device.last_ip ?? t('profile.devices.unknown')}</Descriptions.Item>
        <Descriptions.Item label={t('profile.devices.last_seen_at')}>{device.last_seen_at ? formatDate(device.last_seen_at) : t('profile.devices.unknown')}</Descriptions.Item>
        <Descriptions.Item label={t('profile.devices.created_at')}>{formatDate(device.created_at)}</Descriptions.Item>
        <Descriptions.Item label={t('profile.devices.active_sessions')}>{device.active_session_count}</Descriptions.Item>
      </Descriptions>
      <Popconfirm
        cancelText={t('common.actions.cancel')}
        description={t(device.current ? 'profile.devices.confirm_current' : 'profile.devices.confirm_other')}
        okButtonProps={{ danger: true }}
        okText={t('profile.devices.remove')}
        onConfirm={onRemove}
        title={t('profile.devices.remove_title')}
      >
        <Button danger loading={pending}>{t('profile.devices.remove')}</Button>
      </Popconfirm>
    </Card>
  )
}

function SessionCard({ session, formatDate, onRevoke, pending }: {
  session: AdminSelfSession
  formatDate: (value: string) => string
  onRevoke: () => void
  pending: boolean
}) {
  const { t } = useTranslation()
  return (
    <Card className="ak-session-card">
      <Space align="start" className="ak-session-card-heading">
        <Typography.Text strong>{session.user_agent || t('profile.sessions.unknown')}</Typography.Text>
        {session.current ? <Tag color="blue">{t('profile.sessions.current')}</Tag> : null}
      </Space>
      <Descriptions column={{ xs: 1, sm: 2 }} size="small">
        <Descriptions.Item label={t('profile.sessions.ip_address')}>{session.ip_address ?? t('profile.sessions.unknown')}</Descriptions.Item>
        <Descriptions.Item label={t('profile.sessions.last_seen_at')}>{formatDate(session.last_seen_at)}</Descriptions.Item>
        <Descriptions.Item label={t('profile.sessions.created_at')}>{formatDate(session.created_at)}</Descriptions.Item>
        <Descriptions.Item label={t('profile.sessions.absolute_expires_at')}>{formatDate(session.absolute_expires_at)}</Descriptions.Item>
      </Descriptions>
      <Popconfirm
        cancelText={t('common.actions.cancel')}
        description={t(session.current ? 'profile.sessions.confirm_current' : 'profile.sessions.confirm_other')}
        okButtonProps={{ danger: true }}
        okText={t('profile.sessions.revoke')}
        onConfirm={onRevoke}
        title={t('profile.sessions.revoke_title')}
      >
        <Button danger loading={pending}>{t('profile.sessions.revoke')}</Button>
      </Popconfirm>
    </Card>
  )
}
