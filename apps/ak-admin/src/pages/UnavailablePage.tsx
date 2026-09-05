import { Button, Result } from 'antd'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

interface UnavailablePageProps { titleKey: string }

export function UnavailablePage({ titleKey }: UnavailablePageProps) {
  const { t } = useTranslation()
  return <Result extra={<Link to="/login"><Button type="primary">{t('common.actions.back')}</Button></Link>} status="info" subTitle={t('auth.unavailable.description')} title={t(titleKey)} />
}
