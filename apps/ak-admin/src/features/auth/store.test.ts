import { afterEach, describe, expect, it, vi } from 'vitest'

import { authSession, type AuthContext, useAuthStore } from './store'

function authContext(): AuthContext {
  const tenantId = crypto.randomUUID()
  return {
    user: {
      id: crypto.randomUUID(), email: 'admin@example.test', display_name: 'Admin',
      locale: 'zh-CN', time_zone: 'UTC', avatar_url: null,
    },
    active_tenant: { id: tenantId, code: 'tenant', name: 'Tenant' },
    available_tenants: [{ id: tenantId, code: 'tenant', name: 'Tenant', status: 'active' }],
    roles: [], permissions: [], menus: [], feature_flags: {}, menu_revision: 1,
    permission_revision: 1, server_time: '2026-09-01T00:00:00Z',
  }
}

afterEach(() => {
  vi.restoreAllMocks()
  useAuthStore.setState({ context: null, status: 'bootstrapping' })
})

describe('auth store bootstrap', () => {
  it('shares one cold-session restore across concurrent StrictMode initialization', async () => {
    const context = authContext()
    const restore = vi.spyOn(authSession, 'restore').mockResolvedValue(context)

    const initialize = useAuthStore.getState().initialize
    await Promise.all([initialize(), initialize()])

    expect(restore).toHaveBeenCalledOnce()
    expect(useAuthStore.getState()).toMatchObject({ context, status: 'authenticated' })
  })

  it('settles as anonymous only after cold-session recovery fails', async () => {
    vi.spyOn(authSession, 'restore').mockRejectedValue(new Error('SESSION_MISSING'))
    const clearLocalSession = vi.spyOn(authSession, 'clearLocalSession')

    await useAuthStore.getState().initialize()

    expect(clearLocalSession).toHaveBeenCalledOnce()
    expect(useAuthStore.getState()).toMatchObject({ context: null, status: 'anonymous' })
  })
})
