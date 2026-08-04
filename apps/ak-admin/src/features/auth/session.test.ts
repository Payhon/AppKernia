import { describe, expect, it, vi } from 'vitest'

import { AuthSession, MemoryTokenStore } from './session'

function tokenResponse(accessToken: string, csrfToken: string): Response {
  return Response.json({
    code: 'OK',
    message: 'OK',
    data: {
      access_token: accessToken,
      token_type: 'Bearer',
      expires_in: 900,
      csrf_token: csrfToken,
    },
    request_id: 'test-request',
  })
}

describe('AuthSession', () => {
  it('shares one refresh across parallel 401 responses and retries once', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'expired', csrfToken: 'csrf-old-value-that-is-long-enough' })
    let refreshCalls = 0
    const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
      const url = input instanceof Request ? input.url : input instanceof URL ? input.href : input
      if (url.endsWith('/auth/token/refresh')) {
        refreshCalls += 1
        expect(init?.credentials).toBe('include')
        expect(new Headers(init?.headers).get('X-CSRF-Token')).toBe('csrf-old-value-that-is-long-enough')
        await Promise.resolve()
        return tokenResponse('fresh', 'csrf-fresh-value-that-is-long-enough')
      }
      const authorization = new Headers(init?.headers).get('Authorization')
      return authorization === 'Bearer fresh' ? new Response(null, { status: 200 }) : new Response(null, { status: 401 })
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const responses = await Promise.all([session.request('/resource-a'), session.request('/resource-b')])

    expect(responses.map((response) => response.status)).toEqual([200, 200])
    expect(refreshCalls).toBe(1)
    expect(tokens.read()?.accessToken).toBe('fresh')
  })

  it('purges memory tokens and tenant cache on logout', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const clearTenantCache = vi.fn()
    const fetchMock = vi.fn<typeof fetch>((_input, init) => {
      expect(init?.credentials).toBe('include')
      expect(new Headers(init?.headers).get('X-CSRF-Token')).toBe('csrf-value-that-is-long-enough')
      return Promise.resolve(new Response(null, { status: 200 }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache, fetch: fetchMock })

    await session.logout()

    expect(tokens.read()).toBeNull()
    expect(clearTenantCache).toHaveBeenCalledOnce()
  })

  it('does not refresh a forbidden response', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>(() => Promise.resolve(new Response(null, { status: 403 })))
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const response = await session.request('/forbidden')

    expect(response.status).toBe(403)
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('propagates the active locale through login and authorized requests', async () => {
    const tokens = new MemoryTokenStore()
    const deviceKey = crypto.randomUUID()
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(new Headers(init?.headers).get('Accept-Language')).toBe('en-US')
      const url = input instanceof Request ? input.url : input instanceof URL ? input.href : input
      if (url.endsWith('/auth/login')) {
        expect(new Headers(init?.headers).get('X-AK-Device-Key')).toBe(deviceKey)
      }
      return Promise.resolve(url.endsWith('/auth/login')
        ? tokenResponse('access', 'csrf-value-that-is-long-enough')
        : new Response(null, { status: 200 }))
    })
    const session = new AuthSession({
      baseUrl: '/admin-api/v1',
      tokens,
      clearTenantCache: vi.fn(),
      readLocale: () => 'en-US',
      readDeviceKey: () => deviceKey,
      fetch: fetchMock,
    })

    await session.login({ email: 'admin@example.com', password: 'secret-value' })
    await session.request('/resource')

    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('loads public auth feature flags without credentials and with locale', async () => {
    const fetchMock = vi.fn<typeof fetch>((_input, init) => {
      expect(init?.credentials).toBe('include')
      expect(new Headers(init?.headers).get('Accept-Language')).toBe('en-US')
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test-request',
        data: { locale: 'en-US', default_locale: 'zh-CN', supported_locales: ['zh-CN', 'en-US'], feature_flags: { admin_registration: false } },
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens: new MemoryTokenStore(), clearTenantCache: vi.fn(), readLocale: () => 'en-US', fetch: fetchMock })

    const config = await session.publicConfig()

    expect(config.feature_flags['admin_registration']).toBe(false)
  })

  it('submits anonymous registration and recovery writes once with locale and no auth token', async () => {
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      const url = input instanceof Request ? input.url : input instanceof URL ? input.href : input
      const headers = new Headers(init?.headers)
      expect(init?.method).toBe('POST')
      expect(init?.credentials).toBe('include')
      expect(headers.get('Accept-Language')).toBe('en-US')
      expect(headers.get('Authorization')).toBeNull()
      expect(headers.get('Content-Type')).toBe('application/json')
      if (url.endsWith('/auth/register')) return Promise.resolve(Response.json({ code: 'OK', message: 'OK', request_id: 'test', data: { accepted: true } }))
      if (url.endsWith('/auth/password/forgot')) return Promise.resolve(Response.json({ code: 'OK', message: 'OK', request_id: 'test', data: { accepted: true, retry_after_seconds: 60 } }))
      return Promise.resolve(Response.json({ code: 'OK', message: 'OK', request_id: 'test', data: { reset: true } }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens: new MemoryTokenStore(), clearTenantCache: vi.fn(), readLocale: () => 'en-US', fetch: fetchMock })

    await session.register({ email: 'new@example.test', display_name: 'New Admin', password: 'long password value', locale: 'en-US', accept_terms: true })
    await session.forgotPassword({ email: 'new@example.test' })
    await session.resetPassword({ token: 'one-time-token-that-is-long-enough', new_password: 'new long password value' })

    expect(fetchMock).toHaveBeenCalledTimes(3)
    const registrationBody = fetchMock.mock.calls[0]?.[1]?.body
    const resetBody = fetchMock.mock.calls[2]?.[1]?.body
    expect(typeof registrationBody).toBe('string')
    expect(typeof resetBody).toBe('string')
    if (typeof registrationBody !== 'string' || typeof resetBody !== 'string') throw new Error('expected JSON request bodies')
    expect(JSON.parse(registrationBody)).toEqual({ email: 'new@example.test', display_name: 'New Admin', password: 'long password value', locale: 'en-US', accept_terms: true })
    expect(JSON.parse(resetBody)).toEqual({ token: 'one-time-token-that-is-long-enough', new_password: 'new long password value' })
  })

  it('updates the self profile once without replaying a non-idempotent write', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'expired', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(input).toBe('/admin-api/v1/me')
      expect(init?.method).toBe('PATCH')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer expired')
      expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json')
      expect(init?.body).toBe(JSON.stringify({ locale: 'en-US' }))
      return Promise.resolve(Response.json({
        error: { code: 'AUTH.SESSION.UNAUTHORIZED', message_key: 'errors.common.unauthorized', message: 'Unauthorized' },
        request_id: 'test-request',
      }, { status: 401 }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    await expect(session.updateMe({ locale: 'en-US' })).rejects.toMatchObject({ status: 401 })
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('reads the authenticated self profile through the safe GET retry path', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(input).toBe('/admin-api/v1/me')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test-request',
        data: { id: crypto.randomUUID(), email: 'admin@example.test', display_name: 'Admin', avatar_url: null, locale: 'zh-CN', time_zone: 'UTC' },
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const profile = await session.me()

    expect(profile.locale).toBe('zh-CN')
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('creates and completes one authenticated avatar upload without replaying writes', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const uploadID = crypto.randomUUID()
    const fileID = crypto.randomUUID()
    const file = new File([new Uint8Array([137, 80, 78, 71])], 'avatar.png', { type: 'image/png' })
    const progress: number[] = []
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      const headers = new Headers(init?.headers)
      expect(headers.get('Authorization')).toBe('Bearer access')
      if (input === '/admin-api/v1/me/avatar/upload-session') {
        expect(init?.method).toBe('POST')
        expect(headers.get('Content-Type')).toBe('application/json')
        expect(init?.body).toBe(JSON.stringify({ file_name: 'avatar.png', media_type: 'image/png', size_bytes: 4 }))
        return Promise.resolve(Response.json({
          code: 'OK', message: 'OK', request_id: 'test-request',
          data: { id: uploadID, upload_url: `/me/avatar/upload-sessions/${uploadID}/content`, method: 'PUT', expires_at: '2026-08-03T01:00:00Z' },
        }, { status: 201 }))
      }
      expect(input).toBe(`/admin-api/v1/me/avatar/upload-sessions/${uploadID}/content`)
      expect(init?.method).toBe('PUT')
      expect(headers.get('Content-Type')).toBe('image/png')
      expect(init?.body).toBe(file)
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test-request',
        data: { file_id: fileID, avatar_url: `/me/avatar/content?v=${fileID}` },
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const result = await session.uploadAvatar(file, (percent) => progress.push(percent))

    expect(result).toEqual({ file_id: fileID, avatar_url: `/me/avatar/content?v=${fileID}` })
    expect(progress).toEqual([10, 35, 100])
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('loads private avatar bytes through the authenticated retry-safe GET path', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(input).toBe('/admin-api/v1/me/avatar/content?v=file-id')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      return Promise.resolve(new Response(new Uint8Array([1, 2, 3]), { headers: { 'Content-Type': 'image/png' } }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const result = await session.avatarBlob('/me/avatar/content?v=file-id')

    expect(result.type).toBe('image/png')
    expect(Array.from(new Uint8Array(await result.arrayBuffer()))).toEqual([1, 2, 3])
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('lists self sessions through the safe GET path', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const sessionId = crypto.randomUUID()
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(input).toBe('/admin-api/v1/me/sessions')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test-request',
        data: [{
          id: sessionId, audience: 'ak-admin', status: 'active', ip_address: '127.0.0.1',
          user_agent: 'Test browser', last_seen_at: '2026-08-03T00:00:00Z',
          absolute_expires_at: '2026-09-02T00:00:00Z', created_at: '2026-08-03T00:00:00Z', current: true,
        }],
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const result = await session.selfSessions()

    expect(result).toHaveLength(1)
    expect(result[0]?.id).toBe(sessionId)
  })

  it('does not refresh or replay a session revocation and clears current-session state', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const clearTenantCache = vi.fn()
    const sessionId = crypto.randomUUID()
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(input).toBe(`/admin-api/v1/me/sessions/${sessionId}`)
      expect(init?.method).toBe('DELETE')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test-request',
        data: { revoked: true, current_session: true },
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache, fetch: fetchMock })

    const result = await session.revokeSelfSession(sessionId)

    expect(result).toEqual({ revoked: true, current_session: true })
    expect(fetchMock).toHaveBeenCalledOnce()
    expect(tokens.read()).toBeNull()
    expect(clearTenantCache).toHaveBeenCalledOnce()
  })

  it('does not refresh or replay a rejected session revocation', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'expired', csrfToken: 'csrf-value-that-is-long-enough' })
    const sessionId = crypto.randomUUID()
    const fetchMock = vi.fn<typeof fetch>(() => Promise.resolve(Response.json({
      error: { code: 'AUTH.SESSION.UNAUTHORIZED', message_key: 'errors.common.unauthorized', message: 'Unauthorized' },
      request_id: 'test-request',
    }, { status: 401 })))
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    await expect(session.revokeSelfSession(sessionId)).rejects.toMatchObject({ status: 401 })
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('lists registered self devices through the safe GET path', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const deviceId = crypto.randomUUID()
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(input).toBe('/admin-api/v1/me/devices')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test-request',
        data: [{
          id: deviceId, platform: 'web', device_name: '', model: '', os_version: '', app_version: '',
          last_ip: '127.0.0.1', last_seen_at: '2026-08-03T00:00:00Z', created_at: '2026-08-03T00:00:00Z',
          latest_user_agent: 'Test browser', active_session_count: 1, current: true,
        }],
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const result = await session.selfDevices()

    expect(result).toHaveLength(1)
    expect(result[0]?.id).toBe(deviceId)
  })

  it('removes a self device once and clears state when it is current', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const clearTenantCache = vi.fn()
    const deviceId = crypto.randomUUID()
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(input).toBe(`/admin-api/v1/me/devices/${deviceId}`)
      expect(init?.method).toBe('DELETE')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test-request',
        data: { removed: true, current_device: true },
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache, fetch: fetchMock })

    const result = await session.removeSelfDevice(deviceId)

    expect(result).toEqual({ removed: true, current_device: true })
    expect(fetchMock).toHaveBeenCalledOnce()
    expect(tokens.read()).toBeNull()
    expect(clearTenantCache).toHaveBeenCalledOnce()
  })

  it('changes the self password once without replaying the security write', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(input).toBe('/admin-api/v1/me/password/change')
      expect(init?.method).toBe('POST')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json')
      expect(init?.body).toBe(JSON.stringify({ current_password: 'current-password-value', new_password: 'new-password-value' }))
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test-request',
        data: { changed: true, other_sessions_revoked: true },
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const result = await session.changeSelfPassword({
      current_password: 'current-password-value', new_password: 'new-password-value',
    })

    expect(result).toEqual({ changed: true, other_sessions_revoked: true })
    expect(fetchMock).toHaveBeenCalledOnce()
    expect(tokens.read()?.accessToken).toBe('access')
  })

  it('loads all dashboard resources with the selected URL range through safe GET requests', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      const url = input instanceof Request ? input.url : input instanceof URL ? input.href : input
      expect(url).toContain('?range=7d')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      const window = { range: '7d', start_at: '2026-07-28T00:00:00Z', end_at: '2026-08-03T00:00:00Z' }
      if (url.includes('/dashboard/summary')) {
        return Promise.resolve(Response.json({ code: 'OK', message: 'OK', request_id: 'test', data: { ...window, metrics: [] } }))
      }
      if (url.includes('/dashboard/trends')) {
        return Promise.resolve(Response.json({ code: 'OK', message: 'OK', request_id: 'test', data: { ...window, series: [] } }))
      }
      return Promise.resolve(Response.json({ code: 'OK', message: 'OK', request_id: 'test', data: { ...window, operations: [], failed_jobs: [], security_events: [] } }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const [summary, trends, activity] = await Promise.all([
      session.dashboardSummary('7d'), session.dashboardTrends('7d'), session.dashboardActivity('7d'),
    ])

    expect(summary.range).toBe('7d')
    expect(trends.series).toEqual([])
    expect(activity.security_events).toEqual([])
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('does not refresh or replay a rejected password change', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'expired', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>(() => Promise.resolve(Response.json({
      error: { code: 'AUTH.SESSION.UNAUTHORIZED', message_key: 'errors.common.unauthorized', message: 'Unauthorized' },
      request_id: 'test-request',
    }, { status: 401 })))
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    await expect(session.changeSelfPassword({
      current_password: 'current-password-value', new_password: 'new-password-value',
    })).rejects.toMatchObject({ status: 401 })
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('switches tenant once, replaces tokens, purges tenant cache, and bootstraps the new context', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'old-access', csrfToken: 'old-csrf-value-that-is-long-enough' })
    const clearTenantCache = vi.fn()
    const targetTenant = crypto.randomUUID()
    const deviceKey = crypto.randomUUID()
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      const url = input instanceof Request ? input.url : input instanceof URL ? input.href : input
      if (url.endsWith('/auth/switch-tenant')) {
        expect(init?.method).toBe('POST')
        expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer old-access')
        expect(new Headers(init?.headers).get('X-AK-Device-Key')).toBe(deviceKey)
        expect(init?.body).toBe(JSON.stringify({ tenant_id: targetTenant }))
        return Promise.resolve(tokenResponse('new-access', 'new-csrf-value-that-is-long-enough'))
      }
      expect(url).toBe('/admin-api/v1/auth/context')
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer new-access')
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test',
        data: {
          user: { id: crypto.randomUUID(), email: 'admin@example.test', display_name: 'Admin', locale: 'zh-CN', time_zone: 'UTC', avatar_url: null },
          active_tenant: { id: targetTenant, code: 'target', name: 'Target' },
          available_tenants: [{ id: targetTenant, code: 'target', name: 'Target', status: 'active' }],
          roles: [], permissions: [], menus: [], feature_flags: { multi_tenant: true }, menu_revision: 1, permission_revision: 1,
          server_time: '2026-08-03T00:00:00Z',
        },
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache, readDeviceKey: () => deviceKey, fetch: fetchMock })

    const context = await session.switchTenant(targetTenant)

    expect(context.active_tenant.id).toBe(targetTenant)
    expect(tokens.read()?.accessToken).toBe('new-access')
    expect(clearTenantCache).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('does not refresh or replay a rejected tenant switch', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'expired', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>(() => Promise.resolve(Response.json({
      error: { code: 'AUTH.SESSION.UNAUTHORIZED', message_key: 'errors.common.unauthorized', message: 'Unauthorized' }, request_id: 'test',
    }, { status: 401 })))
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    await expect(session.switchTenant(crypto.randomUUID())).rejects.toMatchObject({ status: 401 })
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('loads self MFA and OAuth state through retry-safe authenticated GET requests', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      if (input === '/admin-api/v1/me/mfa') {
        return Promise.resolve(Response.json({ code: 'OK', message: 'OK', request_id: 'test', data: { totp_enabled: false, recovery_codes_remaining: 0 } }))
      }
      expect(input).toBe('/admin-api/v1/me/oauth-accounts')
      return Promise.resolve(Response.json({ code: 'OK', message: 'OK', request_id: 'test', data: { items: [] } }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const [mfa, oauth] = await Promise.all([session.selfMfa(), session.selfOAuthAccounts()])

    expect(mfa).toEqual({ totp_enabled: false, recovery_codes_remaining: 0 })
    expect(oauth).toEqual([])
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('loads one tenant API Client through the retry-safe authenticated GET path', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'access', csrfToken: 'csrf-value-that-is-long-enough' })
    const clientId = crypto.randomUUID()
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(input).toBe(`/admin-api/v1/api-clients/${clientId}`)
      expect(init?.method).toBeUndefined()
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access')
      return Promise.resolve(Response.json({
        code: 'OK', message: 'OK', request_id: 'test',
        data: {
          id: clientId, client_id: 'ak_safe_hint', name: 'Automation', description: '',
          allowed_cidrs: [], status: 'active', created_at: '2026-08-03T00:00:00Z',
          updated_at: '2026-08-03T00:00:00Z', secrets: [], permissions: [],
        },
      }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    const client = await session.adminApiClient(clientId)

    expect(client.id).toBe(clientId)
    expect(client).not.toHaveProperty('secret')
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('sends step-up proof and OAuth callback values once without replaying rejected writes', async () => {
    const tokens = new MemoryTokenStore()
    tokens.update({ accessToken: 'expired', csrfToken: 'csrf-value-that-is-long-enough' })
    const fetchMock = vi.fn<typeof fetch>((input, init) => {
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer expired')
      expect(init?.credentials).toBe('include')
      expect(init?.method).toBe(input === '/admin-api/v1/me/mfa/totp' ? 'DELETE' : 'POST')
      if (input === '/admin-api/v1/me/mfa/totp') {
        expect(init?.body).toBe(JSON.stringify({ method: 'password', proof: 'current-password-value' }))
      } else {
        expect(input).toBe('/admin-api/v1/me/oauth/local/callback')
        expect(init?.body).toBe(JSON.stringify({ code: 'authorization-code', state: 'single-use-state' }))
      }
      return Promise.resolve(Response.json({
        error: { code: 'AUTH.SESSION.UNAUTHORIZED', message_key: 'errors.common.unauthorized', message: 'Unauthorized' },
        request_id: 'test-request',
      }, { status: 401 }))
    })
    const session = new AuthSession({ baseUrl: '/admin-api/v1', tokens, clearTenantCache: vi.fn(), fetch: fetchMock })

    await expect(session.disableSelfTotp({ method: 'password', proof: 'current-password-value' })).rejects.toMatchObject({ status: 401 })
    await expect(session.completeSelfOAuth('local', { code: 'authorization-code', state: 'single-use-state' })).rejects.toMatchObject({ status: 401 })

    expect(fetchMock).toHaveBeenCalledTimes(2)
  })
})
