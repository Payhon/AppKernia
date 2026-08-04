import { describe, expect, it, vi } from 'vitest'

import type { AdminMenuItem } from '../generated/api/types.gen'
import { findRegisteredRoute, isSafeInternalRedirect, resolveBackendMenus } from './route-registry'

function menu(componentKey: string): AdminMenuItem {
  return {
    id: crypto.randomUUID(), parent_id: null, code: componentKey, i18n_key: 'menu.dashboard',
    title: 'diagnostic-only', type: 'page', path: '/dashboard', component_key: componentKey,
    icon: null, affix: false, sort: 1, feature_flag: '',
  }
}

describe('static route registry', () => {
  it('drops unknown backend component keys and emits a warning callback', () => {
    const onUnknown = vi.fn()
    expect(resolveBackendMenus([menu('server.arbitrary.import')], new Set(), {}, onUnknown)).toEqual([])
    expect(onUnknown).toHaveBeenCalledWith('server.arbitrary.import')
  })

  it('applies view permission to direct route resolution', () => {
    const route = findRegisteredRoute('system.users.accounts')
    expect(route).toBeDefined()
    expect(resolveBackendMenus([menu('system.users.accounts')], new Set(), {})).toEqual([])
    expect(resolveBackendMenus([menu('system.users.accounts')], new Set(['iam.user.read']), {})).toHaveLength(1)
  })

  it('exposes implemented notification routes only with their backend permissions', () => {
    expect(resolveBackendMenus([menu('system.notifications.notices')], new Set(), {})).toEqual([])
    expect(resolveBackendMenus([menu('system.notifications.notices')], new Set(['notify.notice.read']), {})).toHaveLength(1)
    expect(resolveBackendMenus([menu('system.notifications.deliveries')], new Set(['notify.delivery.read']), {})).toHaveLength(1)
  })

  it('exposes the implemented schedule route only with its backend permission', () => {
    expect(resolveBackendMenus([menu('system.integrations.schedules')], new Set(), {})).toEqual([])
    expect(resolveBackendMenus([menu('system.integrations.schedules')], new Set(['jobs.schedule.read']), {})).toHaveLength(1)
  })

  it('accepts only registered authenticated same-origin paths as redirects', () => {
    expect(isSafeInternalRedirect('/dashboard')).toBe(true)
    expect(isSafeInternalRedirect('/profile/basic')).toBe(true)
    expect(isSafeInternalRedirect('/profile/security')).toBe(true)
    expect(isSafeInternalRedirect('/system/notifications/notices')).toBe(true)
    expect(isSafeInternalRedirect('/system/integrations/schedules')).toBe(true)
    expect(isSafeInternalRedirect('//evil.example/dashboard')).toBe(false)
    expect(isSafeInternalRedirect('https://evil.example')).toBe(false)
    expect(isSafeInternalRedirect('/login')).toBe(false)
  })
})
