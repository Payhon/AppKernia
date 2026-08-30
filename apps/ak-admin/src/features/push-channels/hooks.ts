import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AdminPushProviderConfigRequest, AdminPushSecretRotationRequestWritable, AdminPushTestRequest, PushEnvironment, PushWritableProvider } from "../../generated/api/types.gen";
import { useTenantKey } from "../tenants/hooks";
import { getPushDeliverySummary, listPushProviderCatalog, listPushProviderConfigs, listPushTestDevices, rotatePushProviderSecret, sendPushTest, transitionPushProviderConfig, upsertPushProviderConfig } from "./api";

export function usePushChannels(appId: string | null, environment: PushEnvironment) {
  const tenant = useTenantKey();
  const root = ["tenant", tenant, "apps", appId, "push"] as const;
  const requireApp = () => { if (!appId) throw new Error("App selection is required"); return appId; };
  return {
    catalog: useQuery({ queryKey: [...root, "catalog"], queryFn: () => listPushProviderCatalog(requireApp()), enabled: Boolean(appId), staleTime: Infinity }),
    configs: useQuery({ queryKey: [...root, "configs", environment], queryFn: () => listPushProviderConfigs(requireApp(), environment), enabled: Boolean(appId) }),
    summary: useQuery({ queryKey: [...root, "summary"], queryFn: () => getPushDeliverySummary(requireApp()), enabled: Boolean(appId) }),
  };
}

export function usePushChannelMutations(appId: string | null, environment: PushEnvironment) {
  const tenant = useTenantKey();
  const client = useQueryClient();
  const root = ["tenant", tenant, "apps", appId, "push"] as const;
  const requireApp = () => { if (!appId) throw new Error("App selection is required"); return appId; };
  const invalidate = () => client.invalidateQueries({ queryKey: root });
  return {
    save: useMutation({ mutationFn: ({ provider, input }: { provider: PushWritableProvider; input: AdminPushProviderConfigRequest }) => upsertPushProviderConfig(requireApp(), provider, { ...input, environment }), onSuccess: invalidate }),
    rotate: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminPushSecretRotationRequestWritable }) => rotatePushProviderSecret(requireApp(), id, input), onSuccess: invalidate }),
    transition: useMutation({ mutationFn: ({ id, action, lockVersion }: { id: string; action: "preflight" | "activate" | "disable"; lockVersion: number }) => transitionPushProviderConfig(requireApp(), id, action, lockVersion), onSuccess: invalidate }),
    test: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminPushTestRequest }) => sendPushTest(requireApp(), id, input) }),
  };
}

export function usePushTestDevices(appId: string | null, provider: PushWritableProvider | null, enabled: boolean) {
  const tenant = useTenantKey();
  const requireSelection = () => {
    if (!appId || !provider) throw new Error("App and provider selection are required");
    return { appId, provider };
  };
  return useQuery({
    queryKey: ["tenant", tenant, "apps", appId, "push-test-devices", provider],
    queryFn: () => { const selected = requireSelection(); return listPushTestDevices(selected.appId, selected.provider); },
    enabled: enabled && Boolean(appId && provider),
  });
}
