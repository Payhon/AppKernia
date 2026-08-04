import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminBlockRulesPage } from "../pages/AdminBlockRulesPage";
export const Route=createFileRoute("/system/security/block-rules")({component:()=> <ProtectedPage><AppShell><PermissionBoundary permission="iam.block_rule.read"><AdminBlockRulesPage/></PermissionBoundary></AppShell></ProtectedPage>});
