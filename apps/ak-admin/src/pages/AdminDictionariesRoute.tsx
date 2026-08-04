import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminDictionariesPage } from "./AdminDictionariesPage";

export function AdminDictionariesRoute() {
  return (
    <AppShell>
      <PermissionBoundary permission="sys.dictionary.read">
        <AdminDictionariesPage />
      </PermissionBoundary>
    </AppShell>
  );
}
