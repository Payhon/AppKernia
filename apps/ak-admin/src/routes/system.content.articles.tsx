import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminArticlesRoute } from "../pages/AdminContentRoutes";

export const Route = createFileRoute("/system/content/articles")({ component: () => <ProtectedPage><AdminArticlesRoute /></ProtectedPage> });
