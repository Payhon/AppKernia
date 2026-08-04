import { createFileRoute } from "@tanstack/react-router";

import { ProtectedPage } from "../app/route-boundaries";
import { AdminFilesRoute } from "../pages/AdminFilesRoute";

export const Route = createFileRoute("/system/storage/files/")({
  component: () => <ProtectedPage><AdminFilesRoute /></ProtectedPage>,
});
