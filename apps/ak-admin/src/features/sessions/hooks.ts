import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { authSession, useAuthStore } from '../auth/store'
import type { AdminOnlineSessionFilters } from '../auth/session'
import { useTenantKey } from '../tenants/hooks'

export function useOnlineSessions(filters: AdminOnlineSessionFilters) {
  const tenantId = useTenantKey()
  return useQuery({
    queryKey: ['tenant', tenantId, 'online-sessions', filters],
    queryFn: () => authSession.adminOnlineSessions(filters),
    placeholderData: (value) => value,
  })
}

export function useRevokeOnlineSession() {
  const client = useQueryClient()
  const tenantId = useTenantKey()
  return useMutation({
    mutationFn: (id: string) => useAuthStore.getState().revokeAdminOnlineSession(id),
    onSuccess: (result) => {
      if (!result.current) void client.invalidateQueries({ queryKey: ['tenant', tenantId, 'online-sessions'] })
    },
  })
}
