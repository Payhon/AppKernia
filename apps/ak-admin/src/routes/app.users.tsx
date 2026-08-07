import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminAppUsersRoute } from "../pages/AdminAppRoutes";
export const Route = createFileRoute("/app/users")({ component: () => <ProtectedPage><AdminAppUsersRoute /></ProtectedPage> });
