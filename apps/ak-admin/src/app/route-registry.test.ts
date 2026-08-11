import { describe, expect, it, vi } from 'vitest'

import type { AdminMenuItem } from '../generated/api/types.gen'
import { findMenuAncestorKeys, findRegisteredRoute, isSafeInternalRedirect, isSystemPath, partitionShellNavigation, resolveBackendMenus } from './route-registry'

let menuId = 0

function page(componentKey: string, parentId: string | null = null, code = componentKey): AdminMenuItem {
  return {
    id: `menu-${String(menuId += 1)}`, parent_id: parentId, code, i18n_key: `menu.${code}`,
    title: 'diagnostic-only', type: 'page', path: '/dashboard', component_key: componentKey,
    icon: 'FileOutlined', affix: false, sort: 1, feature_flag: '',
  }
}

function directory(code: string, parentId: string | null = null, sort = 1): AdminMenuItem {
  return {
    id: `menu-${String(menuId += 1)}`, parent_id: parentId, code, i18n_key: `menu.${code}`,
    title: 'diagnostic-only', type: 'directory', path: `/${code.replaceAll('.', '/')}`, component_key: null,
    icon: 'AppstoreOutlined', affix: false, sort, feature_flag: '',
  }
}

function systemTree(componentKey: string, permissionParent = 'system.users'): AdminMenuItem[] {
  const system = directory('system', null, 20)
  const group = directory(permissionParent, system.id)
  return [system, group, page(componentKey, group.id)]
}

describe('static route registry', () => {
  it('drops unknown backend component keys and emits a warning callback', () => {
    const onUnknown = vi.fn()
    expect(resolveBackendMenus(systemTree('server.arbitrary.import'), new Set(), {}, onUnknown)).toEqual([])
    expect(onUnknown).toHaveBeenCalledWith('server.arbitrary.import')
  })

  it('applies view permission to direct route resolution', () => {
    const route = findRegisteredRoute('system.users.accounts')
    expect(route).toBeDefined()
    const menus = systemTree('system.users.accounts')
    expect(resolveBackendMenus(menus, new Set(), {})).toEqual([])
    expect(resolveBackendMenus(menus, new Set(['iam.user.read']), {})).toHaveLength(1)
  })

  it('exposes implemented notification routes only with their backend permissions', () => {
    expect(resolveBackendMenus(systemTree('system.notifications.notices', 'system.notifications'), new Set(), {})).toEqual([])
    expect(resolveBackendMenus(systemTree('system.notifications.notices', 'system.notifications'), new Set(['notify.notice.read']), {})).toHaveLength(1)
    expect(resolveBackendMenus(systemTree('system.notifications.deliveries', 'system.notifications'), new Set(['notify.delivery.read']), {})).toHaveLength(1)
  })

  it('exposes the implemented schedule route only with its backend permission', () => {
    const menus = systemTree('system.integrations.schedules', 'system.integrations')
    expect(resolveBackendMenus(menus, new Set(), {})).toEqual([])
    expect(resolveBackendMenus(menus, new Set(['jobs.schedule.read']), {})).toHaveLength(1)
  })

  it('keeps App management as a static permission-gated root', () => {
    const app = directory('app', null, 15)
    const users = page('app.users', app.id, 'app.users')
    expect(resolveBackendMenus([app, users], new Set(), {})).toEqual([])
    const resolved = resolveBackendMenus([app, users], new Set(['app.user.read']), {})
    expect(resolved).toHaveLength(1)
    expect(resolved[0]?.code).toBe('app')
    expect(resolved[0]?.children[0]?.path).toBe('/app/users')
  })

  it('preserves the Dashboard and System hierarchy and prunes inaccessible empty groups', () => {
    const dashboard = page('dashboard', null, 'dashboard')
    dashboard.sort = 10
    const system = directory('system', null, 20)
    const settings = directory('system.settings', system.id, 10)
    const users = directory('system.users', system.id, 20)
    const access = directory('system.access', system.id, 30)
    const menus = [
      page('system.access.menus', access.id),
      users,
      page('system.users.accounts', users.id),
      dashboard,
      access,
      page('system.settings.configs', settings.id),
      system,
      settings,
    ]

    const resolved = resolveBackendMenus(
      menus,
      new Set(['iam.user.read', 'sys.config.read', 'sys.menu.read']),
      {},
    )

    expect(resolved.map((item) => item.code)).toEqual(['dashboard', 'system'])
    expect(resolved[1]?.children.map((item) => item.code)).toEqual([
      'system.settings',
      'system.users',
      'system.access',
    ])
    expect(resolved[1]?.children[1]?.children[0]?.code).toBe('system.users.accounts')
    expect(resolved[1]?.children[1]?.children[0]?.icon).toBe('FileOutlined')
    expect(findMenuAncestorKeys(resolved, '/system/users/accounts')).toEqual(['system', 'system.users'])
  })

  it('does not expose arbitrary backend root nodes', () => {
    const arbitraryRoot = directory('custom-root')
    const child = page('dashboard', arbitraryRoot.id)
    expect(resolveBackendMenus([arbitraryRoot, child], new Set(), {})).toEqual([])
  })

  it('moves the permission-filtered System directory out of primary navigation', () => {
    const dashboard = page('dashboard', null, 'dashboard')
    const menus = systemTree('system.users.accounts')
    const resolved = resolveBackendMenus([dashboard, ...menus], new Set(['iam.user.read']), {})

    const partitioned = partitionShellNavigation(resolved)
    expect(partitioned.primary.map((item) => item.code)).toEqual(['dashboard'])
    expect(partitioned.system?.code).toBe('system')
    expect(partitioned.system?.children[0]?.code).toBe('system.users')
  })

  it('keeps the primary menu usable when System is missing or fully pruned', () => {
    const dashboard = page('dashboard', null, 'dashboard')
    const resolved = resolveBackendMenus([dashboard, ...systemTree('system.users.accounts')], new Set(), {})

    expect(partitionShellNavigation(resolved)).toEqual({ primary: resolved, system: null })
  })

  it('recognizes only System routes for the utility active state', () => {
    expect(isSystemPath('/system/settings/configs')).toBe(true)
    expect(isSystemPath('/system')).toBe(true)
    expect(isSystemPath('/systems')).toBe(false)
    expect(isSystemPath('/dashboard')).toBe(false)
  })

  it('accepts only registered authenticated same-origin paths as redirects', () => {
    expect(isSafeInternalRedirect('/dashboard')).toBe(true)
    expect(isSafeInternalRedirect('/profile/basic')).toBe(true)
    expect(isSafeInternalRedirect('/profile/security')).toBe(true)
    expect(isSafeInternalRedirect('/system/notifications/notices')).toBe(true)
    expect(isSafeInternalRedirect('/system/integrations/schedules')).toBe(true)
    expect(isSafeInternalRedirect('/app/content/pages')).toBe(true)
    expect(isSafeInternalRedirect('//evil.example/dashboard')).toBe(false)
    expect(isSafeInternalRedirect('https://evil.example')).toBe(false)
    expect(isSafeInternalRedirect('/login')).toBe(false)
  })
})
