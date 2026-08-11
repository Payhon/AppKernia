import { CloseOutlined, FileTextOutlined, MenuUnfoldOutlined, SettingOutlined } from '@ant-design/icons'
import { Button, Drawer, Layout, Menu, Popover, Select, Tooltip, Typography, message, type MenuProps } from 'antd'
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState, type MouseEvent, type PropsWithChildren } from 'react'
import { useTranslation } from 'react-i18next'

import { ChevronLeftIcon, ChevronRightIcon, MenuIcon as HamburgerMenuIcon } from '../app/icons'
import { ConfiguredMenuIcon } from '../app/menu-icons'
import { findMenuAncestorKeys, isSystemPath, partitionShellNavigation, resolveBackendMenus, type ResolvedMenuItem } from '../app/route-registry'
import { useSidebarStore } from '../app/sidebar-store'
import { requiresAppSelection } from '../features/apps/scope'
import { useAuthStore } from '../features/auth/store'
import { openApiDocsHref } from '../openapi/config'
import { useLocale } from '../shared/i18n'
import { FullscreenToggle } from './FullscreenToggle'
import { GlobalAppSelector } from './GlobalAppSelector'
import { LocaleSwitcher } from './LocaleSwitcher'
import { UserMenu } from './UserMenu'

const { Content, Header, Sider } = Layout

export function AppShell({ children }: PropsWithChildren) {
  const { t } = useTranslation()
  const { locale } = useLocale()
  const [mobileOpen, setMobileOpen] = useState(false)
  const [desktopSystemOpen, setDesktopSystemOpen] = useState(false)
  const [mobileSystemOpen, setMobileSystemOpen] = useState(false)
  const [sidebarControlsVisible, setSidebarControlsVisible] = useState(false)
  const [switchingTenant, setSwitchingTenant] = useState(false)
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const showAppSelector = requiresAppSelection(pathname)
  const context = useAuthStore((state) => state.context)
  const logout = useAuthStore((state) => state.logout)
  const switchTenant = useAuthStore((state) => state.switchTenant)
  const sidebarMode = useSidebarStore((state) => state.mode)
  const setSidebarMode = useSidebarStore((state) => state.setMode)
  const collapsed = sidebarMode === 'collapsed'
  const sidebarHidden = sidebarMode === 'hidden'
  const permissions = useMemo(() => new Set(context?.permissions ?? []), [context?.permissions])
  const menuItems = useMemo(() => resolveBackendMenus(
    context?.menus ?? [],
    permissions,
    context?.feature_flags ?? {},
    (key) => { console.warn(`Unknown static component key: ${key}`) },
  ), [context, permissions])
  const { primary: primaryMenuItems, system: systemMenu } = useMemo(
    () => partitionShellNavigation(menuItems),
    [menuItems],
  )
  const activeAncestorKeys = useMemo(
    () => findMenuAncestorKeys(primaryMenuItems, pathname),
    [pathname, primaryMenuItems],
  )
  const activeSystemAncestorKeys = useMemo(
    () => systemMenu
      ? findMenuAncestorKeys([systemMenu], pathname).filter((key) => key !== systemMenu.code)
      : [],
    [pathname, systemMenu],
  )
  const [openKeys, setOpenKeys] = useState<string[]>([])
  const [systemOpenKeys, setSystemOpenKeys] = useState<string[]>([])
  const desktopSystemTrigger = useRef<HTMLButtonElement | null>(null)
  const mobileSystemTrigger = useRef<HTMLButtonElement | null>(null)

  useEffect(() => {
    setOpenKeys((current) => Array.from(new Set([...current, ...activeAncestorKeys])))
  }, [activeAncestorKeys])

  useEffect(() => {
    setSystemOpenKeys((current) => Array.from(new Set([...current, ...activeSystemAncestorKeys])))
  }, [activeSystemAncestorKeys])

  useEffect(() => {
    if (!desktopSystemOpen && !mobileSystemOpen) return undefined
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      event.preventDefault()
      if (desktopSystemOpen) {
        setDesktopSystemOpen(false)
        window.requestAnimationFrame(() => { desktopSystemTrigger.current?.focus() })
      }
      if (mobileSystemOpen) {
        setMobileSystemOpen(false)
        window.requestAnimationFrame(() => { mobileSystemTrigger.current?.focus() })
      }
    }
    document.addEventListener('keydown', closeOnEscape)
    return () => { document.removeEventListener('keydown', closeOnEscape) }
  }, [desktopSystemOpen, mobileSystemOpen])

  const toggleSidebar = () => {
    if (collapsed) {
      setOpenKeys((current) => Array.from(new Set([...current, ...activeAncestorKeys])))
      setSidebarMode('expanded')
      return
    }
    setSidebarMode('collapsed')
  }

  const restoreSidebar = () => {
    setOpenKeys((current) => Array.from(new Set([...current, ...activeAncestorKeys])))
    setSidebarMode('expanded')
  }

  const closeSystemNavigation = () => {
    setDesktopSystemOpen(false)
    setMobileSystemOpen(false)
    setMobileOpen(false)
  }

  const toMenuItem = (item: ResolvedMenuItem, onNavigate?: () => void): NonNullable<MenuProps['items']>[number] => {
    if (item.type === 'directory') {
      return {
        children: item.children.map((child) => toMenuItem(child, onNavigate)),
        icon: <ConfiguredMenuIcon name={item.icon} />,
        key: item.code,
        label: t(item.i18nKey),
        popupClassName: 'ak-navigation-submenu-popup',
      }
    }
    return {
      icon: <ConfiguredMenuIcon name={item.icon} />,
      key: item.path,
      label: <Link onClick={() => { setMobileOpen(false); onNavigate?.() }} to={item.path as never}>{t(item.i18nKey)}</Link>,
    }
  }

  const navigation = (id: string, isCollapsed = false) => (
    <nav aria-label={t('shell.navigation')} className="ak-primary-navigation" id={id}>
      <Menu
        {...(isCollapsed ? {} : { openKeys })}
        className="ak-navigation-menu"
        items={primaryMenuItems.map((item) => toMenuItem(item))}
        mode="inline"
        onOpenChange={(keys) => {
          if (!isCollapsed) setOpenKeys(keys)
        }}
        selectedKeys={[pathname]}
        theme="dark"
        triggerSubMenuAction="hover"
      />
    </nav>
  )

  const setSystemVisibility = (mobile: boolean, open: boolean) => {
    if (mobile) setMobileSystemOpen(open)
    else setDesktopSystemOpen(open)
    if (open) return
    const trigger = mobile ? mobileSystemTrigger : desktopSystemTrigger
    window.requestAnimationFrame(() => { trigger.current?.focus() })
  }

  const systemNavigation = (mobile: boolean) => {
    if (!systemMenu) return null
    const open = mobile ? mobileSystemOpen : desktopSystemOpen
    const menuId = mobile ? 'ak-mobile-system-navigation' : 'ak-desktop-system-navigation'
    const label = t(open ? 'openapi.system_menu.close' : 'openapi.system_menu.open')
    const triggerRef = mobile ? mobileSystemTrigger : desktopSystemTrigger
    const content = (
      <div className={`ak-system-menu-panel${mobile ? ' ak-system-menu-panel-mobile' : ''}`}>
        <div className="ak-system-menu-heading">{t(systemMenu.i18nKey)}</div>
        <nav aria-label={t(systemMenu.i18nKey)} id={menuId}>
          <Menu
            className="ak-system-navigation-menu"
            inlineIndent={16}
            items={systemMenu.children.map((item) => toMenuItem(item, closeSystemNavigation))}
            mode={mobile ? 'inline' : 'vertical'}
            onOpenChange={setSystemOpenKeys}
            openKeys={systemOpenKeys}
            selectedKeys={[pathname]}
            theme="dark"
            triggerSubMenuAction={mobile ? 'click' : 'hover'}
          />
        </nav>
      </div>
    )

    return (
      <Popover
        arrow={false}
        classNames={{ root: 'ak-system-menu-popover' }}
        content={content}
        destroyOnHidden
        onOpenChange={(nextOpen) => { setSystemVisibility(mobile, nextOpen) }}
        open={open}
        placement={mobile || !collapsed ? 'topRight' : 'topLeft'}
        trigger="click"
      >
        <span className="ak-shell-utility-popover-trigger">
          <Tooltip placement={mobile ? 'top' : 'right'} title={label}>
            <Button
              aria-controls={menuId}
              aria-current={isSystemPath(pathname) ? 'page' : undefined}
              aria-expanded={open}
              aria-haspopup="menu"
              aria-label={label}
              className={`ak-shell-utility-button${isSystemPath(pathname) ? ' ak-shell-utility-button-active' : ''}`}
              icon={<SettingOutlined />}
              onClick={(event: MouseEvent<HTMLButtonElement>) => { triggerRef.current = event.currentTarget }}
              type="text"
            />
          </Tooltip>
        </span>
      </Popover>
    )
  }

  const utilityNavigation = (mobile: boolean) => (
    <div className={`ak-shell-utilities${systemMenu ? '' : ' ak-shell-utilities-single'}`}>
      <Tooltip placement={mobile ? 'top' : 'right'} title={t('openapi.navigation.label')}>
        <Button
          aria-label={t('openapi.navigation.label')}
          className="ak-shell-utility-button"
          href={openApiDocsHref(locale)}
          icon={<FileTextOutlined />}
          rel="noopener noreferrer"
          target="_blank"
          type="text"
        />
      </Tooltip>
      {systemNavigation(mobile)}
    </div>
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
      {sidebarHidden ? (
        <Tooltip placement="right" title={t('shell.expand_navigation')}>
          <Button
            aria-label={t('shell.expand_navigation')}
            className="ak-sider-hidden-restore"
            icon={<ChevronRightIcon />}
            onClick={restoreSidebar}
            type="text"
          />
        </Tooltip>
      ) : (
        <Sider
          className={`ak-desktop-sider${sidebarControlsVisible ? ' ak-desktop-sider-controls-visible' : ''}`}
          collapsed={collapsed}
          collapsible
          onMouseEnter={() => { setSidebarControlsVisible(true) }}
          onMouseLeave={() => { setSidebarControlsVisible(false) }}
          trigger={null}
          width={248}
        >
          <div className="ak-shell-brand">
            <img alt="" aria-hidden="true" className="ak-shell-brand-image" height="36" src="/brand/appkernia-icon-64.png" width="36" />
            {collapsed ? null : <span>{t('app.name')}</span>}
          </div>
          {navigation('ak-primary-navigation', collapsed)}
          {utilityNavigation(false)}
          {collapsed ? (
            <Tooltip placement="right" title={t('shell.hide_navigation')}>
              <Button
                aria-label={t('shell.hide_navigation')}
                className="ak-sider-hide-handle"
                icon={<CloseOutlined />}
                onClick={() => { setSidebarMode('hidden') }}
                onMouseEnter={() => { setSidebarControlsVisible(true) }}
                onMouseLeave={() => { setSidebarControlsVisible(false) }}
                type="text"
              />
            </Tooltip>
          ) : null}
          <Button
            aria-controls="ak-primary-navigation"
            aria-expanded={!collapsed}
            aria-label={t(collapsed ? 'shell.expand_navigation' : 'shell.collapse_navigation')}
            className="ak-sider-collapse-handle"
            icon={collapsed ? <ChevronRightIcon /> : <ChevronLeftIcon />}
            onClick={toggleSidebar}
            onMouseEnter={() => { setSidebarControlsVisible(true) }}
            onMouseLeave={() => { setSidebarControlsVisible(false) }}
            type="text"
          />
        </Sider>
      )}
      <Drawer className="ak-mobile-drawer" closable onClose={() => { setMobileOpen(false); setMobileSystemOpen(false) }} open={mobileOpen} placement="left" size={280} title={<span className="ak-drawer-brand"><img alt="" aria-hidden="true" height="32" src="/brand/appkernia-icon-64.png" width="32" />{t('app.name')}</span>}>
        <div className="ak-mobile-navigation-shell">
          {navigation('ak-mobile-navigation')}
          {utilityNavigation(true)}
        </div>
      </Drawer>
      <Layout>
        <Header className={`ak-shell-header${showAppSelector ? ' ak-shell-header-app-scoped' : ''}`}>
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
            {sidebarHidden ? (
              <Tooltip title={t('shell.expand_navigation')}>
                <Button
                  aria-label={t('shell.expand_navigation')}
                  className="ak-shell-icon-button ak-sidebar-header-restore"
                  icon={<MenuUnfoldOutlined />}
                  onClick={restoreSidebar}
                  type="text"
                />
              </Tooltip>
            ) : null}
            {showAppSelector ? <GlobalAppSelector /> : null}
            <FullscreenToggle />
            <LocaleSwitcher variant="icon" />
            {context ? <UserMenu onSignOut={signOut} roles={context.roles} user={context.user} /> : null}
          </div>
        </Header>
        <Content className="ak-shell-content" id="main-content">{children}</Content>
      </Layout>
    </Layout>
  )
}
