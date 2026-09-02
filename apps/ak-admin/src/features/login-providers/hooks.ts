import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useTenantKey } from "../tenants/hooks";
import {
  createLoginProviderConfig,
  deleteLoginProviderConfig,
  listAppLoginProviderBindings,
  listLoginProviderCatalog,
  listLoginProviderConfigs,
  putAppLoginProviderBindings,
  rotateLoginProviderSecret,
  transitionLoginProviderConfig,
  updateLoginProviderConfig,
} from "./api";
import type {
  AppLoginProviderBindingsWriteInput,
  LoginProviderConfigFilters,
  LoginProviderConfigWriteInput,
  LoginProviderSecretRotationInput,
} from "./model";

export function useLoginProviderCatalog() {
  const tenant = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenant, "login-provider-catalog"],
    queryFn: listLoginProviderCatalog,
    staleTime: Number.POSITIVE_INFINITY,
  });
}

export function useLoginProviderConfigs(filters: LoginProviderConfigFilters) {
  const tenant = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenant, "login-provider-configs", filters],
    queryFn: () => listLoginProviderConfigs(filters),
    placeholderData: (previous) => previous,
  });
}

export function useLoginProviderConfigMutations() {
  const client = useQueryClient();
  const tenant = useTenantKey();
  const root = ["tenant", tenant, "login-provider-configs"] as const;
  const invalidate = () => client.invalidateQueries({ queryKey: root });
  return {
    create: useMutation({ mutationFn: createLoginProviderConfig, onSuccess: invalidate }),
    update: useMutation({
      mutationFn: ({ id, input }: { id: string; input: LoginProviderConfigWriteInput }) => updateLoginProviderConfig(id, input),
      onSuccess: invalidate,
    }),
    rotateSecret: useMutation({
      mutationFn: ({ id, input }: { id: string; input: LoginProviderSecretRotationInput }) => rotateLoginProviderSecret(id, input),
      onSuccess: invalidate,
    }),
    transition: useMutation({
      mutationFn: ({ id, transition, lockVersion }: { id: string; transition: "preflight" | "activate" | "disable"; lockVersion: number }) => transitionLoginProviderConfig(id, transition, lockVersion),
      onSuccess: invalidate,
    }),
    remove: useMutation({
      mutationFn: ({ id, lockVersion }: { id: string; lockVersion: number }) => deleteLoginProviderConfig(id, lockVersion),
      onSuccess: invalidate,
    }),
  };
}

export function useAppLoginProviderBindings(appId: string | null) {
  const tenant = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenant, "apps", appId, "login-provider-bindings"],
    queryFn: () => appId ? listAppLoginProviderBindings(appId) : Promise.resolve([]),
    enabled: Boolean(appId),
  });
}

export function useAppLoginProviderBindingMutation(appId: string | null) {
  const client = useQueryClient();
  const tenant = useTenantKey();
  const root = ["tenant", tenant, "apps", appId, "login-provider-bindings"] as const;
  return useMutation({
    mutationFn: (input: AppLoginProviderBindingsWriteInput) => {
      if (!appId) throw new Error("App selection is required");
      return putAppLoginProviderBindings(appId, input);
    },
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: root }),
        client.invalidateQueries({ queryKey: ["tenant", tenant, "login-provider-configs"] }),
      ]);
    },
  });
}
