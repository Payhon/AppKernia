import { create } from 'zustand'

import type { AdminAuthContextResponse, AdminLoginRequest, AdminMe, AdminOnlineSessionRevokeResponse, AdminSelfDeviceRemoveResponse, AdminSelfSessionRevokeResponse, AdminUpdateMeRequest } from '../../generated/api/types.gen'
import { readActiveLocale } from '../../shared/i18n'
import { purgeTenantScopedQueries, queryClient } from '../../shared/query-client'
import { AuthSession, MemoryTokenStore } from './session'
import { readOrCreateAdminDeviceKey } from './device-key'

export type AuthContext = AdminAuthContextResponse['data']
type AuthStatus = 'anonymous' | 'authenticating' | 'authenticated'

const tokens = new MemoryTokenStore()
const configuredApiBaseUrl: unknown = import.meta.env['VITE_AK_API_BASE_URL']
const apiBaseUrl = typeof configuredApiBaseUrl === 'string'
  ? configuredApiBaseUrl
  : 'http://localhost:8080/admin-api/v1'

export const authSession = new AuthSession({
  baseUrl: apiBaseUrl,
  tokens,
  clearTenantCache: () => { purgeTenantScopedQueries(queryClient) },
  readLocale: readActiveLocale,
  readDeviceKey: readOrCreateAdminDeviceKey,
})

interface AuthState {
  context: AuthContext | null
  status: AuthStatus
  login: (input: AdminLoginRequest) => Promise<AuthContext>
  logout: () => Promise<void>
  switchTenant: (tenantId: string) => Promise<AuthContext>
  refreshContext: () => Promise<AuthContext>
  updateLocale: (locale: 'zh-CN' | 'en-US') => Promise<void>
  updateProfile: (input: AdminUpdateMeRequest) => Promise<AdminMe>
  uploadAvatar: (file: File, onProgress?: (percent: number) => void) => Promise<AdminMe>
  revokeSelfSession: (sessionId: string) => Promise<AdminSelfSessionRevokeResponse['data']>
  removeSelfDevice: (deviceId: string) => Promise<AdminSelfDeviceRemoveResponse['data']>
  revokeAdminOnlineSession: (sessionId: string) => Promise<AdminOnlineSessionRevokeResponse['data']>
}

export const useAuthStore = create<AuthState>((set) => ({
  context: null,
  status: 'anonymous',
  login: async (input) => {
    set({ status: 'authenticating' })
    try {
      await authSession.login(input)
      const context = await authSession.bootstrap()
      set({ context, status: 'authenticated' })
      return context
    } catch (error) {
      set({ context: null, status: 'anonymous' })
      throw error
    }
  },
  logout: async () => {
    await authSession.logout()
    set({ context: null, status: 'anonymous' })
  },
  switchTenant: async (tenantId) => {
    const context = await authSession.switchTenant(tenantId)
    set({ context, status: 'authenticated' })
    return context
  },
  refreshContext: async () => {
    const context = await authSession.bootstrap()
    set({ context, status: 'authenticated' })
    return context
  },
  updateLocale: async (locale) => {
    const profile = await authSession.updateMe({ locale })
    set((state) => state.context
      ? { context: { ...state.context, user: { ...state.context.user, ...profile } } }
      : state)
  },
  updateProfile: async (input) => {
    const profile = await authSession.updateMe(input)
    set((state) => state.context
      ? { context: { ...state.context, user: { ...state.context.user, ...profile } } }
      : state)
    return profile
  },
  uploadAvatar: async (file, onProgress) => {
    await authSession.uploadAvatar(file, onProgress)
    const profile = await authSession.me()
    set((state) => state.context
      ? { context: { ...state.context, user: { ...state.context.user, ...profile } } }
      : state)
    return profile
  },
  revokeSelfSession: async (sessionId) => {
    const result = await authSession.revokeSelfSession(sessionId)
    if (result.current_session) set({ context: null, status: 'anonymous' })
    return result
  },
  removeSelfDevice: async (deviceId) => {
    const result = await authSession.removeSelfDevice(deviceId)
    if (result.current_device) set({ context: null, status: 'anonymous' })
    return result
  },
  revokeAdminOnlineSession: async (sessionId) => {
    const result = await authSession.revokeAdminOnlineSession(sessionId)
    if (result.current) {
      authSession.clearLocalSession()
      set({ context: null, status: 'anonymous' })
    }
    return result
  },
}))
