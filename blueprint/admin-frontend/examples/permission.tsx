import type { PropsWithChildren, ReactNode } from 'react';
export interface PermissionSnapshot { readonly permissions: ReadonlySet<string>; }
export function can(snapshot: PermissionSnapshot, permission: string): boolean { return snapshot.permissions.has(permission); }
export function Can({ snapshot, permission, fallback = null, children }: PropsWithChildren<{ snapshot: PermissionSnapshot; permission: string; fallback?: ReactNode }>) {
  return can(snapshot, permission) ? children : fallback;
}
