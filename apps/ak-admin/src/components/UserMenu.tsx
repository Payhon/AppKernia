import { LogoutOutlined, UserOutlined } from '@ant-design/icons'
import { Avatar, Dropdown, type MenuProps } from 'antd'
import { Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import type { AuthContext } from '../features/auth/store'
import { useSelfAvatarQuery } from '../features/profile/hooks'

interface UserMenuProps {
  onSignOut: () => Promise<void>
  roles: AuthContext['roles']
  user: AuthContext['user']
}

export function UserMenu({ onSignOut, roles, user }: UserMenuProps) {
  const { t } = useTranslation()
  const avatar = useSelfAvatarQuery(user.avatar_url)
  const [avatarURL, setAvatarURL] = useState<string | null>(null)
  const initialSource = user.display_name.trim() || user.email.trim()
  const initial = Array.from(initialSource)[0]?.toLocaleUpperCase() ?? 'A'
  const roleText = roles.length > 0 ? roles.join(' · ') : t('shell.account_no_roles')

  useEffect(() => {
    if (!avatar.data) {
      setAvatarURL(null)
      return
    }
    const next = URL.createObjectURL(avatar.data)
    setAvatarURL(next)
    return () => { URL.revokeObjectURL(next) }
  }, [avatar.data])

  const items: MenuProps['items'] = [
    {
      icon: <UserOutlined />,
      key: 'profile',
      label: <Link to="/profile/basic">{t('shell.personal_center')}</Link>,
    },
    {
      danger: true,
      icon: <LogoutOutlined />,
      key: 'sign-out',
      label: t('common.actions.sign_out'),
      onClick: () => { void onSignOut() },
    },
  ]

  const avatarNode = (
    <Avatar alt={t('profile.avatar.alt', { name: user.display_name })} size={36} src={avatarURL ?? undefined}>
      {initial}
    </Avatar>
  )

  return (
    <Dropdown
      menu={{ items }}
      placement="bottomRight"
      popupRender={(menu) => (
        <div aria-label={t('shell.account_information')} className="ak-account-popup" role="region">
          <div className="ak-account-summary">
            {avatarNode}
            <div className="ak-account-summary-text">
              <strong>{user.display_name}</strong>
              <span><span className="ak-account-role-label">{t('shell.account_roles')}：</span>{roleText}</span>
            </div>
          </div>
          <div className="ak-account-menu">{menu}</div>
        </div>
      )}
      rootClassName="ak-account-dropdown"
      trigger={['click']}
    >
      <button aria-label={t('shell.open_account_menu')} className="ak-account-trigger" type="button">
        {avatarNode}
      </button>
    </Dropdown>
  )
}
