import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import type { AdminUserAssignmentsRequest, AdminUserCreateRequestWritable, AdminUserResetPasswordRequest, AdminUserRolesRequest, AdminUserUpdateRequest } from '../../generated/api/types.gen'
import { authSession } from '../auth/store'
import type { AdminUserFilters } from '../auth/session'

export const adminUsersKey = ['admin-users'] as const

export function useAdminUsers(filters: AdminUserFilters) { return useQuery({ queryKey: [...adminUsersKey, filters], queryFn: () => authSession.adminUsers(filters), placeholderData: (value) => value }) }
export function useAdminUser(id: string) { return useQuery({ queryKey: [...adminUsersKey, id], queryFn: () => authSession.adminUser(id), enabled: Boolean(id) }) }
export function useAdminUserRoleOptions(enabled = true) { return useQuery({ queryKey: [...adminUsersKey, 'role-options'], queryFn: () => authSession.adminUserRoleOptions(), enabled }) }
export function useAdminUserSessions(id: string, enabled = true) { return useQuery({ queryKey: [...adminUsersKey, id, 'sessions'], queryFn: () => authSession.adminUserSessions(id), enabled: Boolean(id) && enabled }) }

export function useAdminUserMutations() {
  const client = useQueryClient()
  const invalidate = () => client.invalidateQueries({ queryKey: adminUsersKey })
  return {
    create: useMutation({ mutationFn: (input: AdminUserCreateRequestWritable) => authSession.createAdminUser(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminUserUpdateRequest }) => authSession.updateAdminUser(id, input), onSuccess: invalidate }),
    status: useMutation({ mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => authSession.setAdminUserEnabled(id, enabled), onSuccess: invalidate }),
    unlock: useMutation({ mutationFn: (id: string) => authSession.unlockAdminUser(id), onSuccess: invalidate }),
    resetPassword: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminUserResetPasswordRequest }) => authSession.resetAdminUserPassword(id, input), onSuccess: invalidate }),
    roles: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminUserRolesRequest }) => authSession.replaceAdminUserRoles(id, input), onSuccess: invalidate }),
    assignments: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminUserAssignmentsRequest }) => authSession.replaceAdminUserAssignments(id, input), onSuccess: invalidate }),
    revokeSession: useMutation({ mutationFn: ({ id, sessionId }: { id: string; sessionId: string }) => authSession.revokeAdminUserSession(id, sessionId), onSuccess: invalidate }),
  }
}
