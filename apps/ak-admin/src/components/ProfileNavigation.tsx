import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { useAuthStore } from '../features/auth/store'

export function ProfileNavigation({ active }: { active: 'basic' | 'security' | 'connections' }) {
  const { t } = useTranslation()
  const showConnections = useAuthStore((state) => state.context?.feature_flags['oauth'] === true && state.context.permissions.includes('iam.oauth.manage_self'))
  return (
    <nav aria-label={t('routes.profile.basic.title')} className="ak-profile-navigation">
      <Link aria-current={active === 'basic' ? 'page' : undefined} className="ak-profile-nav-link" to="/profile/basic">{t('profile.navigation.basic')}</Link>
      <Link aria-current={active === 'security' ? 'page' : undefined} className="ak-profile-nav-link" to="/profile/security">{t('profile.navigation.security')}</Link>
      {showConnections ? <Link aria-current={active === 'connections' ? 'page' : undefined} className="ak-profile-nav-link" to="/profile/connections">{t('profile.navigation.connections')}</Link> : null}
    </nav>
  )
}
