import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminContentPage } from "./AdminContentPage";
import { AppScopeContext } from "../features/apps/scope";

function AppContent({ initialView }: { initialView: "articles" | "categories" }) {
  return <AppShell><PermissionBoundary permission="app.content.read"><AppScopeContext>{(scope) => <AdminContentPage appId={scope.appId} initialView={initialView} routeBase="/app/content" />}</AppScopeContext></PermissionBoundary></AppShell>;
}
export function AdminAppArticlesRoute() { return <AppContent initialView="articles" />; }
export function AdminAppCategoriesRoute() { return <AppContent initialView="categories" />; }
