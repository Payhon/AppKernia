import { Card, Typography } from 'antd'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { ShieldIcon } from '../app/icons'
import { LocaleSwitcher } from './LocaleSwitcher'

interface AuthFrameProps {
  children: ReactNode
  descriptionKey: string
  headingKey: string
}

export function AuthFrame({ children, descriptionKey, headingKey }: AuthFrameProps) {
  const { t } = useTranslation()
  return (
    <main className="ak-auth-layout">
      <section className="ak-auth-brand" aria-labelledby="auth-brand-title">
        <div className="ak-brand-mark" aria-hidden="true"><ShieldIcon /></div>
        <Typography.Title id="auth-brand-title" level={1}>{t('app.name')}</Typography.Title>
        <Typography.Paragraph>{t('auth.login.security')}</Typography.Paragraph>
      </section>
      <section className="ak-auth-form-section">
        <div className="ak-auth-toolbar"><LocaleSwitcher /></div>
        <Card className="ak-auth-card" variant="borderless">
          <Typography.Title level={2}>{t(headingKey)}</Typography.Title>
          <Typography.Paragraph type="secondary">{t(descriptionKey)}</Typography.Paragraph>
          {children}
        </Card>
      </section>
    </main>
  )
}
