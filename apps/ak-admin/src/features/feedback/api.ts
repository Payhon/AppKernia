import { authSession } from "../auth/store";
import { toApiError } from "../../shared/api/error";
import type { FeedbackResponse, FeedbackListResponse, FeedbackReplyInput, FeedbackStatusInput } from "../../generated/api/types.gen";
import type { FeedbackSearch } from "./model";
async function request<T>(path: `/${string}`, init?: RequestInit): Promise<T> {
  const response = await authSession.adminRequest(path, init);
  if (!response.ok) throw await toApiError(response);
  return await response.json() as T;
}
const root = (appId: string): `/${string}` => `/apps/${encodeURIComponent(appId)}/feedbacks`;
function write(method: string, body: object, key?: string): RequestInit { return { method, headers: { "Content-Type": "application/json", ...(key ? { "Idempotency-Key": key } : {}) }, body: JSON.stringify(body) }; }
export async function listFeedback(appId: string, filters: FeedbackSearch, signal?: AbortSignal) {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) if (key !== "app_id" && value !== undefined && value !== "") search.set(key, String(value));
  return (await request<FeedbackListResponse>(`${root(appId)}?${search}`, signal ? { signal } : undefined)).data;
}
export async function getFeedback(appId: string, id: string, signal?: AbortSignal) { return (await request<FeedbackResponse>(`${root(appId)}/${encodeURIComponent(id)}`, signal ? { signal } : undefined)).data; }
export async function updateFeedback(appId: string, id: string, input: FeedbackStatusInput) { return (await request<FeedbackResponse>(`${root(appId)}/${encodeURIComponent(id)}`, write("PATCH", input))).data; }
export async function replyFeedback(appId: string, id: string, input: FeedbackReplyInput, key: string) { return (await request<FeedbackResponse>(`${root(appId)}/${encodeURIComponent(id)}/replies`, write("POST", input, key))).data; }
export async function feedbackImage(appId: string, id: string, fileId: string, signal: AbortSignal): Promise<Blob> {
  const response = await authSession.adminRequest(`${root(appId)}/${encodeURIComponent(id)}/attachments/${encodeURIComponent(fileId)}/content`, { signal });
  if (!response.ok) throw await toApiError(response);
  const blob = await response.blob();
  if (!["image/png", "image/jpeg", "image/webp"].includes(blob.type)) throw new Error("Invalid feedback image response");
  return blob;
}
