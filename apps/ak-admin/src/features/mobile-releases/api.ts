import type { AdminMobileRelease, AdminMobileReleaseRequest } from "../../generated/api/types.gen";
import { toApiError } from "../../shared/api/error";
import { authSession } from "../auth/store";
import type { MobileReleaseInput } from "./model";

async function data<T>(path: `/${string}`, init?: RequestInit): Promise<T> {
  const response = await authSession.adminRequest(path, init);
  if (!response.ok) throw await toApiError(response);
  return ((await response.json()) as { data: T }).data;
}

function request(input: MobileReleaseInput): AdminMobileReleaseRequest {
  return {
    platform: input.platform,
    current_version: input.current_version,
    minimum_version: input.minimum_version,
    upgrade_url: input.upgrade_url || null,
    release_notes: input.release_notes,
    active: input.active,
    ...(input.lock_version === undefined ? {} : { lock_version: input.lock_version }),
  };
}

const json = (method: "POST" | "PATCH", body: AdminMobileReleaseRequest): RequestInit => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

export const mobileReleasesApi = {
  list: () => data<AdminMobileRelease[]>("/mobile/releases"),
  create: (input: MobileReleaseInput) => data<AdminMobileRelease>("/mobile/releases", json("POST", request(input))),
  update: (id: string, input: MobileReleaseInput) => data<AdminMobileRelease>(`/mobile/releases/${encodeURIComponent(id)}`, json("PATCH", request(input))),
};
