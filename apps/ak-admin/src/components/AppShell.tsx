import { Button, Drawer, Layout, Menu, Select, Typography, message } from 'antd'
import { Link, useNavigate } from '@tanstack/react-router'
import { useMemo, useState, type PropsWithChildren } from 'react'
import { useTranslation } from 'react-i18next'

import { GridIcon, MenuIcon, ShieldIcon, UserIcon } from '../app/icons'
import { isImplementedRoute, resolveBackendMenus } from '../app/route-registry'
import { useAuthStore } from '../features/auth/store'
import { LocaleSwitcher } from './LocaleSwitcher'

const { Content, Header, Sider } = Layout

export function AppShell({ children }: PropsWithChildren) {
  const { t } = useTranslation()
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [switchingTenant, setSwitchingTenant] = useState(false)
  const navigate = useNavigate()
  const context = useAuthStore((state) => state.context)
  const logout = useAuthStore((state) => state.logout)
  const switchTenant = useAuthStore((state) => state.switchTenant)
  const permissions = useMemo(() => new Set(context?.permissions ?? []), [context?.permissions])
  const menuItems = useMemo(() => resolveBackendMenus(
    context?.menus ?? [],
    permissions,
    context?.feature_flags ?? {},
    (key) => { console.warn(`Unknown static component key: ${key}`) },
  ).filter((item) => isImplementedRoute(item.componentKey)), [context, permissions])
  const navigation = (
    <nav aria-label={t('shell.navigation')}>
      <Menu
        items={menuItems.map((item) => ({
          icon: item.componentKey === 'dashboard' ? <GridIcon /> : item.componentKey.includes('users') ? <UserIcon /> : <ShieldIcon />,
          key: item.path,
          label: <Link to={item.path as '/dashboard' | '/profile/basic' | '/profile/security' | '/profile/connections' | '/system/users/accounts' | '/system/users/departments' | '/system/users/positions' | '/system/users/tenants' | '/system/notifications/notices' | '/system/notifications/messages' | '/system/notifications/templates' | '/system/notifications/deliveries' | '/system/integrations/schedules' | '/system/integrations/api-clients' | '/system/integrations/webhooks' | '/system/security/block-rules' | '/system/monitoring/health'}>{t(item.i18nKey)}</Link>,
        }))}
        mode="inline"
        selectedKeys={[window.location.pathname]}
        theme="dark"
      />
    </nav>
  )

  const signOut = async () => {
    await logout()
    window.location.assign('/login')
  }

  const changeTenant = async (tenantId: string) => {
    if (!context || tenantId === context.active_tenant.id) return
    setSwitchingTenant(true)
    try {
      await switchTenant(tenantId)
      await navigate({ to: '/dashboard', search: { range: '30d' }, replace: true })
      void message.success(t('shell.tenant_switch_success'))
    } catch {
      void message.error(t('shell.tenant_switch_error'))
    } finally {
      setSwitchingTenant(false)
    }
  }

  return (
    <Layout className="ak-app-layout">
      <a className="ak-skip-link" href="#main-content">{t('shell.skip_to_content')}</a>
      <Sider className="ak-desktop-sider" collapsed={collapsed} collapsible trigger={null} width={248}>
        <div className="ak-shell-brand"><span className="ak-shell-brand-mark">{t('app.short_name')}</span>{collapsed ? null : <span>{t('app.name')}</span>}</div>
        {navigation}
        <Button aria-label={t(collapsed ? 'shell.expand_navigation' : 'shell.collapse_navigation')} className="ak-collapse-button" ghost icon={<MenuIcon />} onClick={() => { setCollapsed((value) => !value) }} />
      </Sider>
      <Drawer className="ak-mobile-drawer" closable onClose={() => { setMobileOpen(false) }} open={mobileOpen} placement="left" size={280} title={t('app.name')}>{navigation}</Drawer>
      <Layout>
        <Header className="ak-shell-header">
          <Button aria-label={t('shell.open_navigation')} className="ak-mobile-menu-button" icon={<MenuIcon />} onClick={() => { setMobileOpen(true) }} type="text" />
          <div className="ak-tenant-context">
            <Typography.Text type="secondary">{t('shell.current_tenant')}</Typography.Text>
            {context?.feature_flags['multi_tenant'] && context.available_tenants.length > 1 ? (
              <Select
                aria-label={t('shell.switch_tenant')}
                loading={switchingTenant}
                onChange={(value) => { void changeTenant(value) }}
                options={context.available_tenants.map((tenant) => ({ label: tenant.name, value: tenant.id }))}
                popupMatchSelectWidth={false}
                value={context.active_tenant.id}
              />
            ) : <Typography.Text strong>{context?.active_tenant.name}</Typography.Text>}
          </div>
          <div className="ak-shell-actions">
            <LocaleSwitcher />
            <Link className="ak-user-name" to="/profile/basic">{context?.user.display_name}</Link>
            <Button onClick={() => { void signOut() }} type="text">{t('common.actions.sign_out')}</Button>
          </div>
        </Header>
        <Content className="ak-shell-content" id="main-content">{children}</Content>
      </Layout>
    </Layout>
  )
}
