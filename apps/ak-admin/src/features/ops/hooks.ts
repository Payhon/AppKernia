import { useQuery } from "@tanstack/react-query";
import { authSession } from "../auth/store";
import { useTenantKey } from "../tenants/hooks";

export function useOpsHealth() { const tenant=useTenantKey(); return useQuery({queryKey:["tenant",tenant,"ops-health"],queryFn:()=>authSession.adminOpsHealth()}); }
export function useOpsRuntime() { const tenant=useTenantKey(); return useQuery({queryKey:["tenant",tenant,"ops-runtime"],queryFn:()=>authSession.adminOpsRuntime()}); }
