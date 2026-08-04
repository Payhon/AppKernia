import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminFilesPage } from "./AdminFilesPage";
export function AdminFilesRoute(){return <AppShell><PermissionBoundary permission="storage.file.read"><AdminFilesPage/></PermissionBoundary></AppShell>}
