import type { PropsWithChildren } from 'react'
import { useTranslation } from 'react-i18next'

import { useAuthStore } from '../features/auth/store'
import { ErrorPage } from '../pages/ErrorPage'

export function PermissionBoundary({ permission, children }: PropsWithChildren<{ permission: string }>) {
  const { t } = useTranslation()
  const permissions = useAuthStore((state) => state.context?.permissions ?? [])
  if (!permissions.includes(permission)) return <ErrorPage status="403" titleKey="routes.errors.forbidden.title" />
  return <>{children}<span className="ak-sr-only">{t('common.states.enabled')}</span></>
}
