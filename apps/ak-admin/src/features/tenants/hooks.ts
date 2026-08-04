import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import type { AdminTenantCreateRequest, AdminTenantMemberAddRequest, AdminTenantMemberUpdateRequest, AdminTenantUpdateRequest } from '../../generated/api/types.gen'
import { authSession, useAuthStore } from '../auth/store'
import type { AdminTenantFilters } from '../auth/session'

export function useTenantKey() {
  return useAuthStore((state) => state.context?.active_tenant.id ?? 'anonymous')
}

export function useAdminTenants(filters: AdminTenantFilters) {
  const tenantId = useTenantKey()
  return useQuery({ queryKey: ['tenant', tenantId, 'tenants', filters], queryFn: () => authSession.adminTenants(filters), placeholderData: (value) => value })
}

export function useAdminTenant(id: string) {
  const tenantId = useTenantKey()
  return useQuery({ queryKey: ['tenant', tenantId, 'tenants', id], queryFn: () => authSession.adminTenant(id), enabled: Boolean(id) })
}

export function useAdminTenantMembers(id: string) {
  const tenantId = useTenantKey()
  return useQuery({ queryKey: ['tenant', tenantId, 'tenants', id, 'members'], queryFn: () => authSession.adminTenantMembers(id), enabled: Boolean(id) })
}

export function useAdminTenantMutations() {
  const client = useQueryClient()
  const tenantId = useTenantKey()
  const root = ['tenant', tenantId, 'tenants'] as const
  const invalidate = () => client.invalidateQueries({ queryKey: root })
  return {
    create: useMutation({ mutationFn: (input: AdminTenantCreateRequest) => authSession.createAdminTenant(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminTenantUpdateRequest }) => authSession.updateAdminTenant(id, input), onSuccess: invalidate }),
    addMember: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminTenantMemberAddRequest }) => authSession.addAdminTenantMember(id, input), onSuccess: invalidate }),
    updateMember: useMutation({ mutationFn: ({ id, userId, input }: { id: string; userId: string; input: AdminTenantMemberUpdateRequest }) => authSession.updateAdminTenantMember(id, userId, input), onSuccess: invalidate }),
    removeMember: useMutation({ mutationFn: ({ id, userId }: { id: string; userId: string }) => authSession.removeAdminTenantMember(id, userId), onSuccess: invalidate }),
  }
}
