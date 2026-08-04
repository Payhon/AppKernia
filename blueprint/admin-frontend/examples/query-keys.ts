export interface UserSearch { q?: string; status?: string; departmentId?: string; page: number; pageSize: number; sort: string; }
export const userKeys = {
  root: (tenantId: string) => ['tenant', tenantId, 'users'] as const,
  lists: (tenantId: string) => [...userKeys.root(tenantId), 'list'] as const,
  list: (tenantId: string, search: UserSearch) => [...userKeys.lists(tenantId), search] as const,
  detail: (tenantId: string, id: string) => [...userKeys.root(tenantId), 'detail', id] as const,
};
