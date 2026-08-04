import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { authSession } from '../auth/store'
import type { AdminAuditFilters } from '../auth/session'
import { useTenantKey } from '../tenants/hooks'

function useAuditKeys() {
  const tenantId = useTenantKey()
  return {
    root: ['tenant', tenantId, 'audit'] as const,
    operations: ['tenant', tenantId, 'audit', 'operations'] as const,
    logins: ['tenant', tenantId, 'audit', 'logins'] as const,
    events: ['tenant', tenantId, 'audit', 'security-events'] as const,
  }
}

export function useAuditOperations(filters: AdminAuditFilters) {
  const keys = useAuditKeys()
  return useQuery({ queryKey: [...keys.operations, filters], queryFn: () => authSession.adminAuditOperations(filters), placeholderData: (value) => value })
}

export function useAuditLogins(filters: AdminAuditFilters) {
  const keys = useAuditKeys()
  return useQuery({ queryKey: [...keys.logins, filters], queryFn: () => authSession.adminAuditLogins(filters), placeholderData: (value) => value })
}

export function useAuditSecurityEvents(filters: AdminAuditFilters) {
  const keys = useAuditKeys()
  return useQuery({ queryKey: [...keys.events, filters], queryFn: () => authSession.adminAuditSecurityEvents(filters), placeholderData: (value) => value })
}

export function useAuditSecurityEvent(id: string) {
  const keys = useAuditKeys()
  return useQuery({ queryKey: [...keys.events, id], queryFn: () => authSession.adminAuditSecurityEvent(id), enabled: Boolean(id) })
}

export function useResolveAuditSecurityEvent() {
  const client = useQueryClient()
  const keys = useAuditKeys()
  return useMutation({ mutationFn: (id: string) => authSession.resolveAdminAuditSecurityEvent(id), onSuccess: () => client.invalidateQueries({ queryKey: keys.events }) })
}
