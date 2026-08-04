import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import type { AdminOAuthCallbackRequest, AdminStepUpRequestWritable, AdminTotpVerifyRequest } from '../../generated/api/types.gen'
import { authSession, useAuthStore } from '../auth/store'

export const selfMfaQueryKey = ['profile', 'mfa'] as const
export const selfOAuthQueryKey = ['profile', 'oauth-accounts'] as const

function enabled(permission: string) {
  const state = useAuthStore.getState()
  return state.status === 'authenticated' && (state.context?.permissions.includes(permission) ?? false)
}

export function useSelfMfa() {
  return useQuery({ queryKey: selfMfaQueryKey, queryFn: () => authSession.selfMfa(), enabled: enabled('iam.mfa.manage_self') })
}

export function useMfaMutations() {
  const client = useQueryClient()
  const invalidate = async () => { await client.invalidateQueries({ queryKey: selfMfaQueryKey }) }
  return {
    enroll: useMutation({ mutationFn: () => authSession.enrollSelfTotp() }),
    verify: useMutation({ mutationFn: (input: AdminTotpVerifyRequest) => authSession.verifySelfTotp(input), onSuccess: invalidate }),
    disable: useMutation({ mutationFn: (input: AdminStepUpRequestWritable) => authSession.disableSelfTotp(input), onSuccess: invalidate }),
    rotate: useMutation({ mutationFn: (input: AdminStepUpRequestWritable) => authSession.rotateSelfRecoveryCodes(input), onSuccess: invalidate }),
  }
}

export function useSelfOAuthAccounts() {
  return useQuery({ queryKey: selfOAuthQueryKey, queryFn: () => authSession.selfOAuthAccounts(), enabled: enabled('iam.oauth.manage_self') })
}

export function useOAuthMutations() {
  const client = useQueryClient()
  return {
    start: useMutation({ mutationFn: (provider: string) => authSession.startSelfOAuth(provider) }),
    complete: useMutation({ mutationFn: ({ provider, input }: { provider: string; input: AdminOAuthCallbackRequest }) => authSession.completeSelfOAuth(provider, input), onSuccess: async () => { await client.invalidateQueries({ queryKey: selfOAuthQueryKey }) } }),
    remove: useMutation({ mutationFn: (provider: string) => authSession.deleteSelfOAuth(provider), onSuccess: async () => { await client.invalidateQueries({ queryKey: selfOAuthQueryKey }) } }),
  }
}
