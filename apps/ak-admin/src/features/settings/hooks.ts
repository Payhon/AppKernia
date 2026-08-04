import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  AdminConfigSecretRequestWritable,
  AdminConfigWriteRequestWritable,
  AdminDictionaryItemWriteRequest,
  AdminDictionaryTypeWriteRequest,
} from "../../generated/api/types.gen";
import { authSession } from "../auth/store";
import type {
  AdminConfigFilters,
  AdminDictionaryItemFilters,
  AdminDictionaryTypeFilters,
  AdminModuleFilters,
  AdminRegionFilters,
} from "../auth/session";
import { useTenantKey } from "../tenants/hooks";

export function useAdminConfigs(filters: AdminConfigFilters) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "settings", "configs", filters],
    queryFn: () => authSession.adminConfigs(filters),
    placeholderData: (value) => value,
  });
}

export function useAdminRegions(filters: AdminRegionFilters) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "settings", "regions", filters],
    queryFn: () => authSession.adminRegions(filters),
    placeholderData: (value) => value,
  });
}

export function useAdminModules(filters: AdminModuleFilters) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "settings", "modules", filters],
    queryFn: () => authSession.adminModules(filters),
    placeholderData: (value) => value,
  });
}

export function useAdminConfigMutations() {
  const client = useQueryClient();
  const tenantId = useTenantKey();
  const root = ["tenant", tenantId, "settings", "configs"] as const;
  const invalidate = () => client.invalidateQueries({ queryKey: root });
  return {
    create: useMutation({
      mutationFn: (input: AdminConfigWriteRequestWritable) =>
        authSession.createAdminConfig(input),
      onSuccess: invalidate,
    }),
    update: useMutation({
      mutationFn: ({
        id,
        input,
      }: {
        id: string;
        input: AdminConfigWriteRequestWritable;
      }) => authSession.updateAdminConfig(id, input),
      onSuccess: invalidate,
    }),
    rotate: useMutation({
      mutationFn: ({
        id,
        input,
      }: {
        id: string;
        input: AdminConfigSecretRequestWritable;
      }) => authSession.rotateAdminConfigSecret(id, input),
      onSuccess: invalidate,
    }),
  };
}

export function useAdminDictionaryTypes(filters: AdminDictionaryTypeFilters) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: [
      "tenant",
      tenantId,
      "settings",
      "dictionaries",
      "types",
      filters,
    ],
    queryFn: () => authSession.adminDictionaryTypes(filters),
    placeholderData: (value) => value,
  });
}

export function useAdminDictionaryItems(
  typeId: string,
  filters: AdminDictionaryItemFilters,
) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: [
      "tenant",
      tenantId,
      "settings",
      "dictionaries",
      typeId,
      "items",
      filters,
    ],
    queryFn: () => authSession.adminDictionaryItems(typeId, filters),
    enabled: Boolean(typeId),
    placeholderData: (value) => value,
  });
}

export function useAdminDictionaryMutations() {
  const client = useQueryClient();
  const tenantId = useTenantKey();
  const root = ["tenant", tenantId, "settings", "dictionaries"] as const;
  const invalidate = () => client.invalidateQueries({ queryKey: root });
  return {
    createType: useMutation({
      mutationFn: (input: AdminDictionaryTypeWriteRequest) =>
        authSession.createAdminDictionaryType(input),
      onSuccess: invalidate,
    }),
    updateType: useMutation({
      mutationFn: ({
        id,
        input,
      }: {
        id: string;
        input: AdminDictionaryTypeWriteRequest;
      }) => authSession.updateAdminDictionaryType(id, input),
      onSuccess: invalidate,
    }),
    createItem: useMutation({
      mutationFn: ({
        typeId,
        input,
      }: {
        typeId: string;
        input: AdminDictionaryItemWriteRequest;
      }) => authSession.createAdminDictionaryItem(typeId, input),
      onSuccess: invalidate,
    }),
    updateItem: useMutation({
      mutationFn: ({
        id,
        input,
      }: {
        id: string;
        input: AdminDictionaryItemWriteRequest;
      }) => authSession.updateAdminDictionaryItem(id, input),
      onSuccess: invalidate,
    }),
    deleteItem: useMutation({
      mutationFn: (id: string) => authSession.deleteAdminDictionaryItem(id),
      onSuccess: invalidate,
    }),
  };
}
