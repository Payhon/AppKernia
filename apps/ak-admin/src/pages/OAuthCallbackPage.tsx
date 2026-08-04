import { Alert, Button, Result, Typography } from 'antd'
import { Link, useParams } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useOAuthMutations } from '../features/identity-security/hooks'

export function OAuthCallbackPage() {
  const { t } = useTranslation()
  const { provider } = useParams({ from: '/auth/callback/$provider' })
  const completion = useOAuthMutations().complete
  const submitted = useRef(false)
  const [missingParameters, setMissingParameters] = useState(false)

  useEffect(() => {
    if (submitted.current) return
    submitted.current = true
    const url = new URL(window.location.href)
    const code = url.searchParams.get('code') ?? ''
    const state = url.searchParams.get('state') ?? ''
    window.history.replaceState(window.history.state, '', `/auth/callback/${encodeURIComponent(provider)}`)
    if (!code || !state) {
      setMissingParameters(true)
      return
    }
    completion.mutate({ provider, input: { code, state } })
  }, [completion, provider])

  if (completion.isSuccess) {
    return (
      <main className="ak-auth-shell">
        <Result
          extra={<Link to="/profile/connections"><Button type="primary">{t('profile.oauth_callback.return')}</Button></Link>}
          status="success"
          subTitle={t('profile.oauth_callback.success_description', { provider: t(`profile.connections.providers.${provider}`, { defaultValue: provider }) })}
          title={<Typography.Title level={1}>{t('profile.oauth_callback.success')}</Typography.Title>}
        />
      </main>
    )
  }
  if (completion.isError || missingParameters) {
    return (
      <main className="ak-auth-shell">
        <Result
          extra={<Link to="/profile/connections"><Button>{t('profile.oauth_callback.return')}</Button></Link>}
          status="error"
          subTitle={t('profile.oauth_callback.error_description')}
          title={<Typography.Title level={1}>{t('profile.oauth_callback.error')}</Typography.Title>}
        />
      </main>
    )
  }
  return <main className="ak-auth-shell"><Alert aria-live="polite" showIcon title={t('profile.oauth_callback.processing')} type="info" /></main>
}
