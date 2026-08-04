import { Button, Result } from 'antd'
import { useTranslation } from 'react-i18next'

interface UnavailablePageProps { titleKey: string }

export function UnavailablePage({ titleKey }: UnavailablePageProps) {
  const { t } = useTranslation()
  return <Result extra={<Button href="/login" type="primary">{t('common.actions.back')}</Button>} status="info" subTitle={t('auth.unavailable.description')} title={t(titleKey)} />
}
