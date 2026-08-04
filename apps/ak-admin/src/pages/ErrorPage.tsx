import { Button, Result, Typography } from 'antd'
import { useTranslation } from 'react-i18next'

interface ErrorPageProps { status: '403' | '404' | '500'; titleKey: string }

export function ErrorPage({ status, titleKey }: ErrorPageProps) {
  const { t } = useTranslation()
  return (
    <main className="ak-error-page">
      <Result
        extra={<Button href="/dashboard" type="primary">{t('common.actions.back')}</Button>}
        status={status}
        title={<Typography.Title level={1}>{t(titleKey)}</Typography.Title>}
      />
    </main>
  )
}
