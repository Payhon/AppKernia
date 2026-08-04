import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import type { AdminMenuMoveRequest, AdminMenuRequest, AdminRoleDataScopeRequest, AdminRoleMenusRequest, AdminRolePermissionsRequest, AdminRoleRequest } from '../../generated/api/types.gen'
import { authSession } from '../auth/store'
import type { AdminPermissionFilters, AdminRoleFilters } from '../auth/session'
import { useTenantKey } from '../tenants/hooks'

export function useAccessKeys() {
  const tenantId = useTenantKey()
  return {
    root: ['tenant', tenantId, 'access'] as const,
    roles: ['tenant', tenantId, 'access', 'roles'] as const,
    permissions: ['tenant', tenantId, 'access', 'permissions'] as const,
    menus: ['tenant', tenantId, 'access', 'menus'] as const,
  }
}

export function useAdminRoles(filters: AdminRoleFilters) {
  const keys = useAccessKeys()
  return useQuery({ queryKey: [...keys.roles, filters], queryFn: () => authSession.adminRoles(filters), placeholderData: (value) => value })
}

export function useAdminPermissions(filters: AdminPermissionFilters) {
  const keys = useAccessKeys()
  return useQuery({ queryKey: [...keys.permissions, filters], queryFn: () => authSession.adminPermissions(filters), staleTime: 60_000 })
}

export function useAdminMenus() {
  const keys = useAccessKeys()
  return useQuery({ queryKey: keys.menus, queryFn: () => authSession.adminMenus() })
}

export function useAdminRoleMutations() {
  const client = useQueryClient()
  const keys = useAccessKeys()
  const invalidate = () => client.invalidateQueries({ queryKey: keys.root })
  return {
    create: useMutation({ mutationFn: (input: AdminRoleRequest) => authSession.createAdminRole(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminRoleRequest }) => authSession.updateAdminRole(id, input), onSuccess: invalidate }),
    remove: useMutation({ mutationFn: (id: string) => authSession.deleteAdminRole(id), onSuccess: invalidate }),
    permissions: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminRolePermissionsRequest }) => authSession.replaceAdminRolePermissions(id, input), onSuccess: invalidate }),
    menus: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminRoleMenusRequest }) => authSession.replaceAdminRoleMenus(id, input), onSuccess: invalidate }),
    scope: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminRoleDataScopeRequest }) => authSession.replaceAdminRoleDataScope(id, input), onSuccess: invalidate }),
  }
}

export function useAdminMenuMutations() {
  const client = useQueryClient()
  const keys = useAccessKeys()
  const invalidate = () => client.invalidateQueries({ queryKey: keys.menus })
  return {
    create: useMutation({ mutationFn: (input: AdminMenuRequest) => authSession.createAdminMenu(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminMenuRequest }) => authSession.updateAdminMenu(id, input), onSuccess: invalidate }),
    move: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminMenuMoveRequest }) => authSession.moveAdminMenu(id, input), onSuccess: invalidate }),
    remove: useMutation({ mutationFn: (id: string) => authSession.deleteAdminMenu(id), onSuccess: invalidate }),
  }
}
