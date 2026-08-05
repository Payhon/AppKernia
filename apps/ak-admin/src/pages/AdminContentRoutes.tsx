import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminContentPage } from "./AdminContentPage";

export function AdminArticlesRoute() { return <AppShell><PermissionBoundary permission="content.article.read"><AdminContentPage initialView="articles" /></PermissionBoundary></AppShell>; }
export function AdminCategoriesRoute() { return <AppShell><PermissionBoundary permission="content.category.read"><AdminContentPage initialView="categories" /></PermissionBoundary></AppShell>; }
