import { Alert, Button, Card, Empty, Popconfirm, Skeleton, Space, Tag, Typography } from 'antd'
import { useNavigate } from '@tanstack/react-router'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { ProfileNavigation } from '../components/ProfileNavigation'
import { useAuthStore } from '../features/auth/store'
import { useOAuthMutations, useSelfOAuthAccounts } from '../features/identity-security/hooks'

const availableProviders = ['local'] as const

export function ProfileConnectionsPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const featureEnabled = useAuthStore((state) => state.context?.feature_flags['oauth'] === true)
  const accounts = useSelfOAuthAccounts()
  const mutations = useOAuthMutations()
  const locale = i18n.resolvedLanguage === 'en-US' ? 'en-US' : 'zh-CN'
  const formatter = useMemo(() => new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }), [locale])

  const start = async (provider: string) => {
    try {
      const result = await mutations.start.mutateAsync(provider)
      const authorization = new URL(result.authorization_url, window.location.origin)
      if (authorization.origin === window.location.origin) {
        const code = authorization.searchParams.get('code') ?? ''
        const state = authorization.searchParams.get('state') ?? ''
        await navigate({
          to: '/auth/callback/$provider',
          params: { provider },
          search: { code, state },
        })
        return
      }
      window.location.assign(authorization.toString())
    } catch {
      // The mutation state renders localized error feedback.
    }
  }

  if (!featureEnabled) {
    return <Alert role="alert" showIcon title={t('profile.connections.feature_disabled')} type="warning" />
  }

  return (
    <div className="ak-page-container">
      <header className="ak-page-heading">
        <Typography.Title level={1}>{t('routes.profile.connections.title')}</Typography.Title>
        <Typography.Paragraph type="secondary">{t('profile.connections.description')}</Typography.Paragraph>
      </header>
      <ProfileNavigation active="connections" />
      {accounts.isPending ? <Card className="ak-connections-card"><Skeleton active paragraph={{ rows: 3 }} /></Card> : null}
      {accounts.isError ? (
        <Alert
          action={<Button onClick={() => { void accounts.refetch() }} size="small">{t('common.actions.retry')}</Button>}
          role="alert"
          showIcon
          title={t('profile.connections.load_error')}
          type="error"
        />
      ) : null}
      {mutations.start.isError || mutations.remove.isError ? <Alert className="ak-session-feedback" role="alert" showIcon title={t('profile.connections.action_error')} type="error" /> : null}
      {mutations.remove.isSuccess ? <Alert className="ak-session-feedback" role="status" showIcon title={t('profile.connections.unbind_success')} type="success" /> : null}
      {accounts.data?.length === 0 ? <Card className="ak-connections-card"><Empty description={t('profile.connections.empty')} /></Card> : null}
      <div className="ak-connections-grid">
        {availableProviders.map((provider) => {
          const account = accounts.data?.find((candidate) => candidate.provider === provider)
          return (
            <Card className="ak-connections-card" key={provider}>
              <div className="ak-connection-heading">
                <Space>
                  <span aria-hidden="true" className="ak-provider-mark">AK</span>
                  <div>
                    <Typography.Title level={2}>{t(`profile.connections.providers.${provider}`)}</Typography.Title>
                    <Typography.Text type="secondary">{t('profile.connections.provider_description')}</Typography.Text>
                  </div>
                </Space>
                <Tag {...(account ? { className: 'ak-status-tag-success' } : {})}>{t(account ? 'profile.connections.connected' : 'profile.connections.not_connected')}</Tag>
              </div>
              {account ? (
                <Space orientation="vertical" size="small">
                  <Typography.Text>{t('profile.connections.account_hint', { hint: account.account_hint })}</Typography.Text>
                  <Typography.Text type="secondary">{t('profile.connections.bound_at', { date: formatter.format(new Date(account.bound_at)) })}</Typography.Text>
                  <Popconfirm
                    cancelText={t('common.actions.cancel')}
                    description={t('profile.connections.unbind_confirm')}
                    okButtonProps={{ danger: true }}
                    okText={t('profile.connections.unbind')}
                    onConfirm={() => { mutations.remove.mutate(provider) }}
                    title={t('profile.connections.unbind_title')}
                  >
                    <Button danger loading={mutations.remove.isPending}>{t('profile.connections.unbind')}</Button>
                  </Popconfirm>
                </Space>
              ) : (
                <Button loading={mutations.start.isPending} onClick={() => { void start(provider) }} type="primary">
                  {t('profile.connections.bind')}
                </Button>
              )}
            </Card>
          )
        })}
      </div>
    </div>
  )
}
