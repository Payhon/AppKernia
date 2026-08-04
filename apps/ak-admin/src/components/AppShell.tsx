import { Button, Drawer, Layout, Menu, Select, Typography, message, type MenuProps } from 'antd'
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { useEffect, useMemo, useState, type PropsWithChildren } from 'react'
import { useTranslation } from 'react-i18next'

import { MenuIcon as HamburgerMenuIcon } from '../app/icons'
import { ConfiguredMenuIcon } from '../app/menu-icons'
import { findMenuAncestorKeys, resolveBackendMenus, type ResolvedMenuItem } from '../app/route-registry'
import { useAuthStore } from '../features/auth/store'
import { LocaleSwitcher } from './LocaleSwitcher'

const { Content, Header, Sider } = Layout

export function AppShell({ children }: PropsWithChildren) {
  const { t } = useTranslation()
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [switchingTenant, setSwitchingTenant] = useState(false)
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const context = useAuthStore((state) => state.context)
  const logout = useAuthStore((state) => state.logout)
  const switchTenant = useAuthStore((state) => state.switchTenant)
  const permissions = useMemo(() => new Set(context?.permissions ?? []), [context?.permissions])
  const menuItems = useMemo(() => resolveBackendMenus(
    context?.menus ?? [],
    permissions,
    context?.feature_flags ?? {},
    (key) => { console.warn(`Unknown static component key: ${key}`) },
  ), [context, permissions])
  const activeAncestorKeys = useMemo(() => findMenuAncestorKeys(menuItems, pathname), [menuItems, pathname])
  const [openKeys, setOpenKeys] = useState<string[]>([])

  useEffect(() => {
    setOpenKeys((current) => Array.from(new Set([...current, ...activeAncestorKeys])))
  }, [activeAncestorKeys])

  const toMenuItem = (item: ResolvedMenuItem): NonNullable<MenuProps['items']>[number] => {
    if (item.type === 'directory') {
      return {
        children: item.children.map(toMenuItem),
        icon: <ConfiguredMenuIcon name={item.icon} />,
        key: item.code,
        label: t(item.i18nKey),
      }
    }
    return {
      icon: <ConfiguredMenuIcon name={item.icon} />,
      key: item.path,
      label: <Link onClick={() => { setMobileOpen(false) }} to={item.path as never}>{t(item.i18nKey)}</Link>,
    }
  }

  const navigation = (
    <nav aria-label={t('shell.navigation')}>
      <Menu
        className="ak-navigation-menu"
        items={menuItems.map(toMenuItem)}
        mode="inline"
        onOpenChange={(keys) => { setOpenKeys(keys) }}
        openKeys={openKeys}
        selectedKeys={[pathname]}
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
        <Button aria-label={t(collapsed ? 'shell.expand_navigation' : 'shell.collapse_navigation')} className="ak-collapse-button" ghost icon={<HamburgerMenuIcon />} onClick={() => { setCollapsed((value) => !value) }} />
      </Sider>
      <Drawer className="ak-mobile-drawer" closable onClose={() => { setMobileOpen(false) }} open={mobileOpen} placement="left" size={280} title={t('app.name')}>{navigation}</Drawer>
      <Layout>
        <Header className="ak-shell-header">
          <Button aria-label={t('shell.open_navigation')} className="ak-mobile-menu-button" icon={<HamburgerMenuIcon />} onClick={() => { setMobileOpen(true) }} type="text" />
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
