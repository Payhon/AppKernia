import { createFileRoute } from "@tanstack/react-router";

import { ProtectedPage } from "../app/route-boundaries";
import { AdminFileDetailRoute } from "../pages/AdminFileDetailRoute";

export const Route = createFileRoute("/system/storage/files/$fileId")({ component: FileDetail });

function FileDetail() {
  const { fileId } = Route.useParams();
  return <ProtectedPage><AdminFileDetailRoute fileId={fileId} /></ProtectedPage>;
}
