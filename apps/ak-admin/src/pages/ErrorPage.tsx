import { Button, Result, Typography } from 'antd'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

interface ErrorPageProps { status: '403' | '404' | '500'; titleKey: string }

export function ErrorPage({ status, titleKey }: ErrorPageProps) {
  const { t } = useTranslation()
  return (
    <main className="ak-error-page">
      <Result
        extra={<Link to="/dashboard" search={{ range: '30d' }}><Button type="primary">{t('common.actions.back')}</Button></Link>}
        status={status}
        title={<Typography.Title level={1}>{t(titleKey)}</Typography.Title>}
      />
    </main>
  )
}
