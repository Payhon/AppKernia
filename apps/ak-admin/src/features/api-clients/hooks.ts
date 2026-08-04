import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AdminApiClientPermissionsRequest, AdminApiClientRequest, AdminApiClientSecretRequest } from "../../generated/api/types.gen";
import type { AdminApiClientFilters } from "../auth/session";
import { authSession } from "../auth/store";
import { useTenantKey } from "../tenants/hooks";
export function useApiClients(filters: AdminApiClientFilters){const tenant=useTenantKey();return useQuery({queryKey:["tenant",tenant,"api-clients",filters],queryFn:()=>authSession.adminApiClients(filters),placeholderData:v=>v})}
export function useApiClient(id:string){const tenant=useTenantKey();return useQuery({queryKey:["tenant",tenant,"api-clients",id],queryFn:()=>authSession.adminApiClient(id),enabled:Boolean(id)})}
export function useApiClientMutations(){const tenant=useTenantKey();const qc=useQueryClient();const invalidate=()=>qc.invalidateQueries({queryKey:["tenant",tenant,"api-clients"]});return{
 create:useMutation({mutationFn:(input:AdminApiClientRequest)=>authSession.createAdminApiClient(input),onSuccess:invalidate}),
 update:useMutation({mutationFn:({id,input}:{id:string;input:AdminApiClientRequest})=>authSession.updateAdminApiClient(id,input),onSuccess:invalidate}),
 secret:useMutation({mutationFn:({id,input}:{id:string;input:AdminApiClientSecretRequest})=>authSession.createAdminApiClientSecret(id,input),onSuccess:invalidate}),
 revoke:useMutation({mutationFn:({id,secretId}:{id:string;secretId:string})=>authSession.revokeAdminApiClientSecret(id,secretId),onSuccess:invalidate}),
 permissions:useMutation({mutationFn:({id,input}:{id:string;input:AdminApiClientPermissionsRequest})=>authSession.replaceAdminApiClientPermissions(id,input),onSuccess:invalidate}),
}}
