import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminAppCategoriesRoute } from "../pages/AdminAppContentRoutes";
export const Route = createFileRoute("/app/content/categories")({ component: () => <ProtectedPage><AdminAppCategoriesRoute /></ProtectedPage> });
