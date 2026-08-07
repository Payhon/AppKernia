import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminAppArticlesRoute } from "../pages/AdminAppContentRoutes";
export const Route = createFileRoute("/app/content/articles")({ component: () => <ProtectedPage><AdminAppArticlesRoute /></ProtectedPage> });
