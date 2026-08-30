import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  AdminShareBindingInput,
  AdminShareConfigInput,
} from "../../generated/api/types.gen";
import { useTenantKey } from "../tenants/hooks";
import {
  createShareConfig,
  deleteAppShareBinding,
  deleteShareConfig,
  listAppShareBindings,
  listShareConfigs,
  preflightAppShareBinding,
  putAppShareBinding,
  transitionShareConfig,
  updateShareConfig,
} from "./api";
import type { ShareConfigFilters } from "./model";

export function useShareConfigs(filters: ShareConfigFilters) {
  const tenant = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenant, "share-configs", filters],
    queryFn: () => listShareConfigs(filters),
    placeholderData: (previous) => previous,
  });
}

export function useShareConfigMutations() {
  const client = useQueryClient();
  const tenant = useTenantKey();
  const root = ["tenant", tenant, "share-configs"] as const;
  const invalidate = () => client.invalidateQueries({ queryKey: root });
  return {
    create: useMutation({ mutationFn: createShareConfig, onSuccess: invalidate }),
    update: useMutation({
      mutationFn: ({ id, input }: { id: string; input: AdminShareConfigInput }) =>
        updateShareConfig(id, input),
      onSuccess: invalidate,
    }),
    transition: useMutation({
      mutationFn: ({ id, transition, lockVersion }: { id: string; transition: "activate" | "disable"; lockVersion: number }) =>
        transitionShareConfig(id, transition, lockVersion),
      onSuccess: invalidate,
    }),
    remove: useMutation({
      mutationFn: ({ id, lockVersion }: { id: string; lockVersion: number }) =>
        deleteShareConfig(id, lockVersion),
      onSuccess: invalidate,
    }),
  };
}

export function useAppShareBindings(appId: string | null) {
  const tenant = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenant, "apps", appId, "share-bindings"],
    queryFn: () => appId ? listAppShareBindings(appId) : Promise.resolve([]),
    enabled: Boolean(appId),
  });
}

export function useAppShareBindingMutations(appId: string | null) {
  const client = useQueryClient();
  const tenant = useTenantKey();
  const root = ["tenant", tenant, "apps", appId, "share-bindings"] as const;
  const invalidate = () => client.invalidateQueries({ queryKey: root });
  const requireAppId = () => {
    if (!appId) throw new Error("App selection is required");
    return appId;
  };
  return {
    preflight: useMutation({
      mutationFn: (input: AdminShareBindingInput) =>
        preflightAppShareBinding(requireAppId(), "wechat", input),
    }),
    save: useMutation({
      mutationFn: (input: AdminShareBindingInput) =>
        putAppShareBinding(requireAppId(), "wechat", input),
      onSuccess: invalidate,
    }),
    remove: useMutation({
      mutationFn: (lockVersion: number) =>
        deleteAppShareBinding(requireAppId(), "wechat", lockVersion),
      onSuccess: invalidate,
    }),
  };
}
