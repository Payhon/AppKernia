import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminFileDetailPage } from "./AdminFileDetailPage";

export function AdminFileDetailRoute({ fileId }: { fileId: string }) {
  return <AppShell><PermissionBoundary permission="storage.file.read"><AdminFileDetailPage fileId={fileId} /></PermissionBoundary></AppShell>;
}
