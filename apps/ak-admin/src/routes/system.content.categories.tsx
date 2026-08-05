import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminCategoriesRoute } from "../pages/AdminContentRoutes";

export const Route = createFileRoute("/system/content/categories")({ component: () => <ProtectedPage><AdminCategoriesRoute /></ProtectedPage> });
