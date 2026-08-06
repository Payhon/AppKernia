import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import type {
  AdminConfigSecretRequestWritable,
  AdminConfigWriteRequestWritable,
  AdminDictionaryItemWriteRequest,
  AdminDictionaryTypeWriteRequest,
  AdminRegionCreateRequest,
  AdminRegionUpdateRequest,
} from "../../generated/api/types.gen";
import { authSession } from "../auth/store";
import type {
  AdminConfigFilters,
  AdminDictionaryItemFilters,
  AdminDictionaryTypeFilters,
  AdminRegionFilters,
} from "../auth/session";
import { useTenantKey } from "../tenants/hooks";

export type AdminConfigSaveOperation =
  | {
      id: string;
      input: AdminConfigWriteRequestWritable;
      kind: "update";
    }
  | {
      id: string;
      input: AdminConfigSecretRequestWritable;
      kind: "secret";
    };

export interface AdminConfigSaveManyResult {
  failures: { error: unknown; id: string }[];
  successIds: string[];
}

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

export function useAdminRegionMutations() {
  const client = useQueryClient();
  const tenantId = useTenantKey();
  const root = ["tenant", tenantId, "settings", "regions"] as const;
  const invalidate = () => client.invalidateQueries({ queryKey: root });
  return {
    create: useMutation({
      mutationFn: (input: AdminRegionCreateRequest) =>
        authSession.createAdminRegion(input),
      onSuccess: invalidate,
    }),
    update: useMutation({
      mutationFn: ({
        code,
        input,
      }: {
        code: string;
        input: AdminRegionUpdateRequest;
      }) => authSession.updateAdminRegion(code, input),
      onSuccess: invalidate,
    }),
    remove: useMutation({
      mutationFn: (code: string) => authSession.deleteAdminRegion(code),
      onSuccess: invalidate,
    }),
  };
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
    saveMany: useMutation({
      mutationFn: async (
        operations: AdminConfigSaveOperation[],
      ): Promise<AdminConfigSaveManyResult> => {
        const result: AdminConfigSaveManyResult = {
          failures: [],
          successIds: [],
        };
        for (const operation of operations) {
          try {
            if (operation.kind === "secret")
              await authSession.rotateAdminConfigSecret(
                operation.id,
                operation.input,
              );
            else
              await authSession.updateAdminConfig(
                operation.id,
                operation.input,
              );
            result.successIds.push(operation.id);
          } catch (error) {
            result.failures.push({ error, id: operation.id });
          }
        }
        return result;
      },
      onSettled: invalidate,
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

export function useAdminDictionary(code: string | null | undefined) {
  const tenantId = useTenantKey();
  const { i18n } = useTranslation();
  return useQuery({
    queryKey: ["tenant", tenantId, "settings", "dictionary", code, i18n.resolvedLanguage],
    queryFn: () => {
      if (!code) throw new Error("dictionary code is required");
      return authSession.adminDictionary(code);
    },
    enabled: Boolean(code),
    staleTime: 60_000,
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
