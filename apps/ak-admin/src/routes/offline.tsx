import { createFileRoute } from "@tanstack/react-router";

import { OfflinePage } from "../pages/OfflinePage";

export const Route = createFileRoute("/offline")({ component: OfflinePage });
