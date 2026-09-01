import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminFeedbackPage } from "../pages/AdminFeedbackPage";
import { feedbackSearchSchema } from "../features/feedback/model";
export const Route = createFileRoute("/app/feedbacks")({ validateSearch: (search) => feedbackSearchSchema.parse(search), component: () => <ProtectedPage><AppShell><PermissionBoundary permission="app.feedback.read"><AdminFeedbackPage /></PermissionBoundary></AppShell></ProtectedPage> });
