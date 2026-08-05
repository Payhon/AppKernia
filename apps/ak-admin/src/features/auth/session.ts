import type {
  AdminAuthContextResponse,
  AdminAvatarUploadCompleteResponse,
  AdminAvatarUploadRequest,
  AdminAvatarUploadSessionResponse,
  AdminDashboardActivityResponse,
  AdminDashboardSummaryResponse,
  AdminDashboardTrendsResponse,
  AdminLoginRequest,
  AdminLoginCaptchaRequest,
  AdminLoginCaptchaResponse,
  AdminRegistrationRequestWritable,
  AdminRegistrationResponse,
  AdminForgotPasswordRequest,
  AdminForgotPasswordResponse,
  AdminResetPasswordRequest,
  AdminResetPasswordResponse,
  AdminMeResponse,
  AdminOrgPositionListResponse,
  AdminOrgPositionRequest,
  AdminOrgPositionResponse,
  AdminOrgUnitMoveRequest,
  AdminOrgUnitRequest,
  AdminOrgUnitResponse,
  AdminOrgUnitTreeResponse,
  AdminDeleteResponse,
  AdminUserActionResponse,
  AdminUserAssignmentsRequest,
  AdminUserCreateRequestWritable,
  AdminUserExportRequest,
  AdminUserImportResponse,
  AdminUserListResponse,
  AdminUserResetPasswordRequest,
  AdminUserResetPasswordResponse,
  AdminUserRoleOptionsResponse,
  AdminUserResponse,
  AdminUserRolesRequest,
  AdminUserSessionListResponse,
  AdminUserUpdateRequest,
  AdminSelfDeviceListResponse,
  AdminMfaStatusResponse,
  AdminTotpEnrollmentResponse,
  AdminTotpVerifyRequest,
  AdminRecoveryCodesResponse,
  AdminStepUpRequestWritable,
  AdminMfaDisableResponse,
  AdminOAuthAccountListResponse,
  AdminOAuthStartResponse,
  AdminOAuthCallbackRequest,
  AdminOAuthAccountResponse,
  AdminOAuthUnbindResponse,
  AdminSelfDeviceRemoveResponse,
  AdminSelfSessionListResponse,
  AdminSelfSessionRevokeResponse,
  AdminSelfPasswordChangeRequest,
  AdminSelfPasswordChangeResponse,
  AdminSwitchTenantRequest,
  AdminTenantCreateRequest,
  AdminTenantListResponse,
  AdminTenantMemberAddRequest,
  AdminTenantMemberListResponse,
  AdminTenantMemberResponse,
  AdminTenantMemberUpdateRequest,
  AdminTenantResponse,
  AdminTenantUpdateRequest,
  AdminTokenResponse,
  AdminUpdateMeRequest,
  AdminRoleDataScopeRequest,
  AdminRoleListResponse,
  AdminRoleMenusRequest,
  AdminRolePermissionsRequest,
  AdminRoleRequest,
  AdminRoleResponse,
  AdminPermissionListResponse,
  AdminMenuMoveRequest,
  AdminMenuRequest,
  AdminMenuResponse,
  AdminMenuTreeResponse,
  AdminAuditOperationListResponse,
  AdminAuditLoginListResponse,
  AdminAuditSecurityEventListResponse,
  AdminAuditSecurityEventResponse,
  AdminOnlineSessionListResponse,
  AdminOnlineSessionRevokeResponse,
  AdminConfigListResponse,
  AdminConfigResponse,
  AdminConfigWriteRequestWritable,
  AdminConfigSecretRequestWritable,
  AdminDictionaryTypeListResponse,
  AdminDictionaryTypeResponse,
  AdminDictionaryTypeWriteRequest,
  AdminDictionaryItemListResponse,
  AdminDictionaryItemResponse,
  AdminDictionaryItemWriteRequest,
  AdminModuleListResponse,
  AdminFile,
  AdminFileListResponse,
  AdminFileResponse,
  AdminFileUploadRequest,
  AdminFileUploadSession,
  AdminFileUploadSessionResponse,
  AdminFileUploadPolicyResponse,
  AdminFilePartResponse,
  AdminFileUsageListResponse,
  AdminFileDownloadResponse,
  AdminNotificationMessageListResponse,
  AdminNotificationMessageResponse,
  AdminNotificationMessageRequest,
  AdminNotificationRecipientPreviewResponse,
  AdminNotificationRecipientStatsResponse,
  AdminNotificationPublishResponse,
  AdminNotificationTemplateListResponse,
  AdminNotificationTemplateResponse,
  AdminNotificationTemplateRequest,
  AdminNotificationDeliveryListResponse,
  AdminNotificationDeliveryResponse,
  AdminJobHandlerListResponse,
  AdminJobCronPreviewResponse,
  AdminJobScheduleListResponse,
  AdminJobScheduleResponse,
  AdminJobScheduleRequest,
  AdminJobRunListResponse,
  AdminJobRunResponse,
  AdminApiClientListResponse,
  AdminApiClientResponse,
  AdminApiClientRequest,
  AdminApiClientSecretRequest,
  AdminApiClientSecretCreatedResponseWritable,
  AdminApiClientPermissionsRequest,
  AdminWebhookListResponse,
  AdminWebhookResponse,
  AdminWebhookRequest,
  AdminWebhookCreatedResponseWritable,
  AdminWebhookTestRequest,
  AdminWebhookDeliveryResponse,
  AdminWebhookDeliveryListResponse,
  AdminBlockRuleCreateRequestWritable,
  AdminBlockRuleUpdateRequest,
  AdminBlockRuleListResponse,
  AdminBlockRuleResponse,
  AdminBlockRuleRevokeResponse,
  AdminOpsHealthResponse,
  AdminOpsRuntimeResponse,
  RegionListResponse,
  PublicConfigResponse,
} from "../../generated/api/types.gen";
import { toApiError } from "../../shared/api/error";

type LoginRequest = AdminLoginRequest;
type LoginCaptchaRequest = AdminLoginCaptchaRequest;
type LoginCaptcha = AdminLoginCaptchaResponse["data"];
type TokenResponse = AdminTokenResponse;
type AuthContextResponse = AdminAuthContextResponse;
type PublicConfig = PublicConfigResponse["data"];
export type DashboardRange = "7d" | "30d" | "90d";
export interface AdminUserFilters {
  q?: string;
  status?: string;
  unit_id?: string;
  position_id?: string;
  role_id?: string;
  page?: number;
  page_size?: number;
  sort?: string;
}
export interface AdminApiClientFilters {
  q?: string;
  status?: string;
  page?: number;
  page_size?: number;
}
export interface AdminWebhookFilters {
  q?: string;
  event_type?: string;
  status?: string;
  page?: number;
  page_size?: number;
}
export interface AdminBlockRuleFilters {
  subject_type?: string;
  subject_hint?: string;
  scope?: string;
  status?: string;
  expiry?: string;
  page?: number;
  page_size?: number;
}

export interface AdminTenantFilters {
  q?: string;
  status?: string;
  page?: number;
  page_size?: number;
  sort?: string;
}

export interface AdminRoleFilters {
  q?: string;
  status?: string;
  role_type?: string;
  page?: number;
  page_size?: number;
}

export interface AdminPermissionFilters {
  q?: string;
  module_code?: string;
  resource_name?: string;
  action_name?: string;
  permission_kind?: string;
  status?: string;
}

export interface AdminAuditFilters {
  q?: string;
  from?: string;
  to?: string;
  page?: number;
  page_size?: number;
  module_code?: string;
  result?: string;
  audience?: string;
  auth_method?: string;
  severity?: string;
  source?: string;
  status?: string;
}

export interface AdminOnlineSessionFilters {
  q?: string;
  audience?: string;
  platform?: string;
  status?: string;
  ip?: string;
  from?: string;
  to?: string;
  page?: number;
  page_size?: number;
}

export interface AdminConfigFilters {
  q?: string;
  module_code?: string;
  config_group?: string;
  value_type?: string;
  status?: string;
  is_public?: boolean;
  is_secret?: boolean;
  sort?: string;
  page?: number;
  page_size?: number;
}

export interface AdminDictionaryTypeFilters {
  q?: string;
  status?: string;
  sort?: string;
  page?: number;
  page_size?: number;
}

export interface AdminDictionaryItemFilters {
  q?: string;
  locale?: string;
  status?: string;
  sort?: string;
  page?: number;
  page_size?: number;
}
export interface AdminRegionFilters {
  q?: string;
  parent_code?: string;
  level?: number;
  status?: string;
  limit?: number;
}
export interface AdminModuleFilters {
  q?: string;
  status?: string;
}
export interface AdminFileFilters {
  q?: string;
  status?: string;
  scan_status?: string;
  media_type?: string;
  provider?: string;
  page?: number;
  page_size?: number;
}
export interface AdminNotificationMessageFilters {
  q?: string;
  status?: string;
  message_type?: string;
  page?: number;
  page_size?: number;
}
export interface AdminNotificationTemplateFilters {
  q?: string;
  status?: string;
  channel?: string;
  locale?: string;
  page?: number;
  page_size?: number;
}
export interface AdminNotificationDeliveryFilters {
  q?: string;
  status?: string;
  channel?: string;
  page?: number;
  page_size?: number;
}
export interface AdminJobScheduleFilters {
  q?: string;
  status?: string;
  time_zone?: string;
  page?: number;
  page_size?: number;
}
export interface AdminJobRunFilters {
  status?: string;
  page?: number;
  page_size?: number;
}

export interface TokenSnapshot {
  accessToken: string;
  csrfToken: string;
}

export class MemoryTokenStore {
  #snapshot: TokenSnapshot | null = null;

  read(): TokenSnapshot | null {
    return this.#snapshot;
  }

  update(snapshot: TokenSnapshot): void {
    this.#snapshot = snapshot;
  }

  clear(): void {
    this.#snapshot = null;
  }
}

export interface AuthSessionOptions {
  baseUrl: string;
  tokens: MemoryTokenStore;
  clearTenantCache: () => void;
  readLocale?: () => "zh-CN" | "en-US";
  readDeviceKey?: () => string;
  fetch?: typeof globalThis.fetch;
}

export class AuthSession {
  readonly #baseUrl: string;
  readonly #tokens: MemoryTokenStore;
  readonly #clearTenantCache: () => void;
  readonly #readLocale: () => "zh-CN" | "en-US";
  readonly #readDeviceKey: () => string;
  readonly #fetch: typeof globalThis.fetch;
  #refreshPromise: Promise<string> | null = null;

  constructor(options: AuthSessionOptions) {
    this.#baseUrl = options.baseUrl.replace(/\/$/, "");
    this.#tokens = options.tokens;
    this.#clearTenantCache = options.clearTenantCache;
    this.#readLocale = options.readLocale ?? (() => "zh-CN");
    this.#readDeviceKey = options.readDeviceKey ?? (() => "");
    this.#fetch = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  async login(input: LoginRequest): Promise<void> {
    const deviceKey = this.#readDeviceKey();
    const headers = new Headers({
      "Accept-Language": this.#readLocale(),
      "Content-Type": "application/json",
    });
    if (deviceKey) headers.set("X-AK-Device-Key", deviceKey);
    const response = await this.#fetch(`${this.#baseUrl}/auth/login`, {
      method: "POST",
      credentials: "include",
      headers,
      body: JSON.stringify(input),
    });
    if (!response.ok) throw await toApiError(response);
    this.#storeTokenResponse((await response.json()) as TokenResponse);
  }

  async createLoginCaptcha(input: LoginCaptchaRequest): Promise<LoginCaptcha> {
    const response = await this.#anonymousWrite("/auth/login/captcha", input);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminLoginCaptchaResponse).data;
  }

  async publicConfig(): Promise<PublicConfig> {
    const response = await this.#fetch(`${this.#baseUrl}/auth/public-config`, {
      credentials: "include",
      headers: { "Accept-Language": this.#readLocale() },
    });
    if (!response.ok) throw await toApiError(response);
    const body = (await response.json()) as PublicConfigResponse;
    return body.data;
  }

  async register(
    input: AdminRegistrationRequestWritable,
  ): Promise<AdminRegistrationResponse["data"]> {
    const response = await this.#anonymousWrite("/auth/register", input);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminRegistrationResponse).data;
  }

  async forgotPassword(
    input: AdminForgotPasswordRequest,
  ): Promise<AdminForgotPasswordResponse["data"]> {
    const response = await this.#anonymousWrite("/auth/password/forgot", input);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminForgotPasswordResponse).data;
  }

  async resetPassword(
    input: AdminResetPasswordRequest,
  ): Promise<AdminResetPasswordResponse["data"]> {
    const response = await this.#anonymousWrite("/auth/password/reset", input);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminResetPasswordResponse).data;
  }

  refreshSingleFlight(): Promise<string> {
    if (!this.#refreshPromise) {
      this.#refreshPromise = this.#refresh().finally(() => {
        this.#refreshPromise = null;
      });
    }
    return this.#refreshPromise;
  }

  async request(
    input: string | URL,
    init: RequestInit = {},
  ): Promise<Response> {
    const tokenUsed = this.#tokens.read()?.accessToken;
    const first = await this.#authorizedFetch(input, init);
    if (first.status !== 401) return first;
    if (tokenUsed && this.#tokens.read()?.accessToken !== tokenUsed) {
      return this.#authorizedFetch(input, init);
    }
    await this.refreshSingleFlight();
    return this.#authorizedFetch(input, init);
  }

  async bootstrap(): Promise<AuthContextResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/auth/context`);
    if (!response.ok) throw await toApiError(response);
    const body = (await response.json()) as AuthContextResponse;
    return body.data;
  }

  async switchTenant(tenantId: string): Promise<AuthContextResponse["data"]> {
    const headers = new Headers({ "Content-Type": "application/json" });
    const deviceKey = this.#readDeviceKey();
    if (deviceKey) headers.set("X-AK-Device-Key", deviceKey);
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/auth/switch-tenant`,
      {
        method: "POST",
        headers,
        body: JSON.stringify({
          tenant_id: tenantId,
        } satisfies AdminSwitchTenantRequest),
      },
    );
    if (!response.ok) throw await toApiError(response);
    this.#storeTokenResponse((await response.json()) as TokenResponse);
    this.#clearTenantCache();
    return this.bootstrap();
  }

  async updateMe(
    input: AdminUpdateMeRequest,
  ): Promise<AdminMeResponse["data"]> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/me`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    if (!response.ok) throw await toApiError(response);
    const body = (await response.json()) as AdminMeResponse;
    return body.data;
  }

  async me(): Promise<AdminMeResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/me`);
    if (!response.ok) throw await toApiError(response);
    const body = (await response.json()) as AdminMeResponse;
    return body.data;
  }

  async createAvatarUpload(
    input: AdminAvatarUploadRequest,
  ): Promise<AdminAvatarUploadSessionResponse["data"]> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/me/avatar/upload-session`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      },
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminAvatarUploadSessionResponse).data;
  }

  async uploadAvatar(
    file: File,
    onProgress?: (percent: number) => void,
  ): Promise<AdminAvatarUploadCompleteResponse["data"]> {
    onProgress?.(10);
    const target = await this.createAvatarUpload({
      file_name: file.name,
      media_type: file.type as AdminAvatarUploadRequest["media_type"],
      size_bytes: file.size,
    });
    onProgress?.(35);
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}${target.upload_url}`,
      {
        method: target.method,
        headers: { "Content-Type": file.type },
        body: file,
      },
    );
    if (!response.ok) throw await toApiError(response);
    onProgress?.(100);
    return ((await response.json()) as AdminAvatarUploadCompleteResponse).data;
  }

  async avatarBlob(path: string): Promise<Blob> {
    const response = await this.request(`${this.#baseUrl}${path}`);
    if (!response.ok) throw await toApiError(response);
    return response.blob();
  }

  async selfSessions(): Promise<AdminSelfSessionListResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/me/sessions`);
    if (!response.ok) throw await toApiError(response);
    const body = (await response.json()) as AdminSelfSessionListResponse;
    return body.data;
  }

  async revokeSelfSession(
    sessionId: string,
  ): Promise<AdminSelfSessionRevokeResponse["data"]> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/me/sessions/${encodeURIComponent(sessionId)}`,
      {
        method: "DELETE",
      },
    );
    if (!response.ok) throw await toApiError(response);
    const body = (await response.json()) as AdminSelfSessionRevokeResponse;
    if (body.data.current_session) {
      this.#tokens.clear();
      this.#clearTenantCache();
    }
    return body.data;
  }

  async selfDevices(): Promise<AdminSelfDeviceListResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/me/devices`);
    if (!response.ok) throw await toApiError(response);
    const body = (await response.json()) as AdminSelfDeviceListResponse;
    return body.data;
  }

  async removeSelfDevice(
    deviceId: string,
  ): Promise<AdminSelfDeviceRemoveResponse["data"]> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/me/devices/${encodeURIComponent(deviceId)}`,
      {
        method: "DELETE",
      },
    );
    if (!response.ok) throw await toApiError(response);
    const body = (await response.json()) as AdminSelfDeviceRemoveResponse;
    if (body.data.current_device) {
      this.#tokens.clear();
      this.#clearTenantCache();
    }
    return body.data;
  }

  async changeSelfPassword(
    input: AdminSelfPasswordChangeRequest,
  ): Promise<AdminSelfPasswordChangeResponse["data"]> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/me/password/change`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      },
    );
    if (!response.ok) throw await toApiError(response);
    const body = (await response.json()) as AdminSelfPasswordChangeResponse;
    return body.data;
  }

  async selfMfa(): Promise<AdminMfaStatusResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/me/mfa`)
    if (!response.ok) throw await toApiError(response)
    return ((await response.json()) as AdminMfaStatusResponse).data
  }

  async enrollSelfTotp(): Promise<AdminTotpEnrollmentResponse["data"]> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/me/mfa/totp/enroll`, { method: 'POST' })
    if (!response.ok) throw await toApiError(response)
    return ((await response.json()) as AdminTotpEnrollmentResponse).data
  }

  async verifySelfTotp(input: AdminTotpVerifyRequest): Promise<AdminRecoveryCodesResponse["data"]> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/me/mfa/totp/verify`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) })
    if (!response.ok) throw await toApiError(response)
    return ((await response.json()) as AdminRecoveryCodesResponse).data
  }

  async disableSelfTotp(input: AdminStepUpRequestWritable): Promise<AdminMfaDisableResponse["data"]> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/me/mfa/totp`, { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) })
    if (!response.ok) throw await toApiError(response)
    return ((await response.json()) as AdminMfaDisableResponse).data
  }

  async rotateSelfRecoveryCodes(input: AdminStepUpRequestWritable): Promise<AdminRecoveryCodesResponse["data"]> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/me/mfa/recovery-codes/rotate`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) })
    if (!response.ok) throw await toApiError(response)
    return ((await response.json()) as AdminRecoveryCodesResponse).data
  }

  async selfOAuthAccounts(): Promise<AdminOAuthAccountListResponse["data"]["items"]> {
    const response = await this.request(`${this.#baseUrl}/me/oauth-accounts`)
    if (!response.ok) throw await toApiError(response)
    return ((await response.json()) as AdminOAuthAccountListResponse).data.items
  }

  async startSelfOAuth(provider: string): Promise<AdminOAuthStartResponse["data"]> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/me/oauth/${encodeURIComponent(provider)}/start`, { method: 'POST' })
    if (!response.ok) throw await toApiError(response)
    return ((await response.json()) as AdminOAuthStartResponse).data
  }

  async completeSelfOAuth(provider: string, input: AdminOAuthCallbackRequest): Promise<AdminOAuthAccountResponse["data"]> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/me/oauth/${encodeURIComponent(provider)}/callback`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) })
    if (!response.ok) throw await toApiError(response)
    return ((await response.json()) as AdminOAuthAccountResponse).data
  }

  async deleteSelfOAuth(provider: string): Promise<AdminOAuthUnbindResponse["data"]> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/me/oauth/${encodeURIComponent(provider)}`, { method: 'DELETE' })
    if (!response.ok) throw await toApiError(response)
    return ((await response.json()) as AdminOAuthUnbindResponse).data
  }

  async dashboardSummary(
    range: DashboardRange,
  ): Promise<AdminDashboardSummaryResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/dashboard/summary?range=${range}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminDashboardSummaryResponse).data;
  }

  async dashboardTrends(
    range: DashboardRange,
  ): Promise<AdminDashboardTrendsResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/dashboard/trends?range=${range}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminDashboardTrendsResponse).data;
  }

  async dashboardActivity(
    range: DashboardRange,
  ): Promise<AdminDashboardActivityResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/dashboard/activity?range=${range}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminDashboardActivityResponse).data;
  }

  async orgUnitTree(
    query = "",
    status = "",
  ): Promise<AdminOrgUnitTreeResponse["data"]> {
    const params = new URLSearchParams();
    if (query) params.set("q", query);
    if (status) params.set("status", status);
    const response = await this.request(
      `${this.#baseUrl}/org/units/tree?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminOrgUnitTreeResponse).data;
  }

  async createOrgUnit(
    input: AdminOrgUnitRequest,
  ): Promise<AdminOrgUnitResponse["data"]> {
    return this.#orgWrite<AdminOrgUnitResponse>(
      "/org/units",
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateOrgUnit(
    id: string,
    input: AdminOrgUnitRequest,
  ): Promise<AdminOrgUnitResponse["data"]> {
    return this.#orgWrite<AdminOrgUnitResponse>(
      `/org/units/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async moveOrgUnit(
    id: string,
    input: AdminOrgUnitMoveRequest,
  ): Promise<AdminOrgUnitResponse["data"]> {
    return this.#orgWrite<AdminOrgUnitResponse>(
      `/org/units/${encodeURIComponent(id)}/move`,
      "POST",
      input,
    ).then((body) => body.data);
  }

  async deleteOrgUnit(id: string): Promise<AdminDeleteResponse["data"]> {
    return this.#orgWrite<AdminDeleteResponse>(
      `/org/units/${encodeURIComponent(id)}`,
      "DELETE",
    ).then((body) => body.data);
  }

  async orgPositions(
    query = "",
    status = "",
    unitId = "",
  ): Promise<AdminOrgPositionListResponse["data"]> {
    const params = new URLSearchParams();
    if (query) params.set("q", query);
    if (status) params.set("status", status);
    if (unitId) params.set("unit_id", unitId);
    const response = await this.request(
      `${this.#baseUrl}/org/positions?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminOrgPositionListResponse).data;
  }

  async createOrgPosition(
    input: AdminOrgPositionRequest,
  ): Promise<AdminOrgPositionResponse["data"]> {
    return this.#orgWrite<AdminOrgPositionResponse>(
      "/org/positions",
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateOrgPosition(
    id: string,
    input: AdminOrgPositionRequest,
  ): Promise<AdminOrgPositionResponse["data"]> {
    return this.#orgWrite<AdminOrgPositionResponse>(
      `/org/positions/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async deleteOrgPosition(id: string): Promise<AdminDeleteResponse["data"]> {
    return this.#orgWrite<AdminDeleteResponse>(
      `/org/positions/${encodeURIComponent(id)}`,
      "DELETE",
    ).then((body) => body.data);
  }

  async adminUsers(
    filters: AdminUserFilters,
  ): Promise<AdminUserListResponse["data"]> {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters))
      if (value !== undefined && value !== "") params.set(key, String(value));
    const response = await this.request(
      `${this.#baseUrl}/users?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminUserListResponse).data;
  }

  async adminTenants(
    filters: AdminTenantFilters,
  ): Promise<AdminTenantListResponse["data"]> {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters))
      if (value !== undefined && value !== "") params.set(key, String(value));
    const response = await this.request(
      `${this.#baseUrl}/tenants?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminTenantListResponse).data;
  }

  async adminTenant(id: string): Promise<AdminTenantResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/tenants/${encodeURIComponent(id)}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminTenantResponse).data;
  }

  async createAdminTenant(
    input: AdminTenantCreateRequest,
  ): Promise<AdminTenantResponse["data"]> {
    return this.#tenantWrite<AdminTenantResponse>(
      "/tenants",
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateAdminTenant(
    id: string,
    input: AdminTenantUpdateRequest,
  ): Promise<AdminTenantResponse["data"]> {
    return this.#tenantWrite<AdminTenantResponse>(
      `/tenants/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async adminTenantMembers(
    id: string,
  ): Promise<AdminTenantMemberListResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/tenants/${encodeURIComponent(id)}/members`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminTenantMemberListResponse).data;
  }

  async addAdminTenantMember(
    id: string,
    input: AdminTenantMemberAddRequest,
  ): Promise<AdminTenantMemberResponse["data"]> {
    return this.#tenantWrite<AdminTenantMemberResponse>(
      `/tenants/${encodeURIComponent(id)}/members`,
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateAdminTenantMember(
    id: string,
    userId: string,
    input: AdminTenantMemberUpdateRequest,
  ): Promise<AdminTenantMemberResponse["data"]> {
    return this.#tenantWrite<AdminTenantMemberResponse>(
      `/tenants/${encodeURIComponent(id)}/members/${encodeURIComponent(userId)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async removeAdminTenantMember(
    id: string,
    userId: string,
  ): Promise<AdminTenantMemberResponse["data"]> {
    return this.#tenantWrite<AdminTenantMemberResponse>(
      `/tenants/${encodeURIComponent(id)}/members/${encodeURIComponent(userId)}`,
      "DELETE",
    ).then((body) => body.data);
  }

  async adminRoles(
    filters: AdminRoleFilters,
  ): Promise<AdminRoleListResponse["data"]> {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters))
      if (value !== undefined && value !== "") params.set(key, String(value));
    const response = await this.request(
      `${this.#baseUrl}/roles?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminRoleListResponse).data;
  }

  async createAdminRole(
    input: AdminRoleRequest,
  ): Promise<AdminRoleResponse["data"]> {
    return this.#accessWrite<AdminRoleResponse>("/roles", "POST", input).then(
      (body) => body.data,
    );
  }

  async updateAdminRole(
    id: string,
    input: AdminRoleRequest,
  ): Promise<AdminRoleResponse["data"]> {
    return this.#accessWrite<AdminRoleResponse>(
      `/roles/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async deleteAdminRole(id: string): Promise<AdminDeleteResponse["data"]> {
    return this.#accessWrite<AdminDeleteResponse>(
      `/roles/${encodeURIComponent(id)}`,
      "DELETE",
    ).then((body) => body.data);
  }

  async replaceAdminRolePermissions(
    id: string,
    input: AdminRolePermissionsRequest,
  ): Promise<AdminRoleResponse["data"]> {
    return this.#accessWrite<AdminRoleResponse>(
      `/roles/${encodeURIComponent(id)}/permissions`,
      "PUT",
      input,
    ).then((body) => body.data);
  }

  async replaceAdminRoleMenus(
    id: string,
    input: AdminRoleMenusRequest,
  ): Promise<AdminRoleResponse["data"]> {
    return this.#accessWrite<AdminRoleResponse>(
      `/roles/${encodeURIComponent(id)}/menus`,
      "PUT",
      input,
    ).then((body) => body.data);
  }

  async replaceAdminRoleDataScope(
    id: string,
    input: AdminRoleDataScopeRequest,
  ): Promise<AdminRoleResponse["data"]> {
    return this.#accessWrite<AdminRoleResponse>(
      `/roles/${encodeURIComponent(id)}/data-scope`,
      "PUT",
      input,
    ).then((body) => body.data);
  }

  async adminPermissions(
    filters: AdminPermissionFilters,
  ): Promise<AdminPermissionListResponse["data"]> {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters))
      if (value !== undefined && value !== "") params.set(key, String(value));
    const response = await this.request(
      `${this.#baseUrl}/permissions?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminPermissionListResponse).data;
  }

  async adminMenus(): Promise<AdminMenuTreeResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/menus/tree`);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminMenuTreeResponse).data;
  }

  async createAdminMenu(
    input: AdminMenuRequest,
  ): Promise<AdminMenuResponse["data"]> {
    return this.#accessWrite<AdminMenuResponse>("/menus", "POST", input).then(
      (body) => body.data,
    );
  }

  async updateAdminMenu(
    id: string,
    input: AdminMenuRequest,
  ): Promise<AdminMenuResponse["data"]> {
    return this.#accessWrite<AdminMenuResponse>(
      `/menus/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async moveAdminMenu(
    id: string,
    input: AdminMenuMoveRequest,
  ): Promise<AdminMenuResponse["data"]> {
    return this.#accessWrite<AdminMenuResponse>(
      `/menus/${encodeURIComponent(id)}/move`,
      "POST",
      input,
    ).then((body) => body.data);
  }

  async deleteAdminMenu(id: string): Promise<AdminDeleteResponse["data"]> {
    return this.#accessWrite<AdminDeleteResponse>(
      `/menus/${encodeURIComponent(id)}`,
      "DELETE",
    ).then((body) => body.data);
  }

  async adminAuditOperations(
    filters: AdminAuditFilters,
  ): Promise<AdminAuditOperationListResponse["data"]> {
    return this.#auditList<AdminAuditOperationListResponse>(
      "/audit/operations",
      filters,
    ).then((body) => body.data);
  }

  async adminAuditLogins(
    filters: AdminAuditFilters,
  ): Promise<AdminAuditLoginListResponse["data"]> {
    return this.#auditList<AdminAuditLoginListResponse>(
      "/audit/logins",
      filters,
    ).then((body) => body.data);
  }

  async adminAuditSecurityEvents(
    filters: AdminAuditFilters,
  ): Promise<AdminAuditSecurityEventListResponse["data"]> {
    return this.#auditList<AdminAuditSecurityEventListResponse>(
      "/audit/security-events",
      filters,
    ).then((body) => body.data);
  }

  async adminAuditSecurityEvent(
    id: string,
  ): Promise<AdminAuditSecurityEventResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/audit/security-events/${encodeURIComponent(id)}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminAuditSecurityEventResponse).data;
  }

  async resolveAdminAuditSecurityEvent(
    id: string,
  ): Promise<AdminAuditSecurityEventResponse["data"]> {
    return this.#accessWrite<AdminAuditSecurityEventResponse>(
      `/audit/security-events/${encodeURIComponent(id)}/resolve`,
      "POST",
    ).then((body) => body.data);
  }

  async adminOnlineSessions(
    filters: AdminOnlineSessionFilters,
  ): Promise<AdminOnlineSessionListResponse["data"]> {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters))
      if (value !== undefined && value !== "") params.set(key, String(value));
    const response = await this.request(
      `${this.#baseUrl}/online-sessions?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminOnlineSessionListResponse).data;
  }

  async revokeAdminOnlineSession(
    id: string,
  ): Promise<AdminOnlineSessionRevokeResponse["data"]> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/online-sessions/${encodeURIComponent(id)}`,
      { method: "DELETE" },
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminOnlineSessionRevokeResponse).data;
  }

  async adminConfigs(
    filters: AdminConfigFilters,
  ): Promise<AdminConfigListResponse["data"]> {
    return this.#settingsList<AdminConfigListResponse>(
      "/configs",
      filters,
    ).then((body) => body.data);
  }

  async adminRegions(
    filters: AdminRegionFilters,
  ): Promise<RegionListResponse["data"]> {
    return this.#settingsList<RegionListResponse>("/regions", filters).then(
      (body) => body.data,
    );
  }

  async adminModules(
    filters: AdminModuleFilters,
  ): Promise<AdminModuleListResponse["data"]> {
    return this.#settingsList<AdminModuleListResponse>(
      "/modules",
      filters,
    ).then((body) => body.data);
  }

  async adminFiles(
    filters: AdminFileFilters,
  ): Promise<AdminFileListResponse["data"]> {
    return this.#settingsList<AdminFileListResponse>("/files", filters).then(
      (body) => body.data,
    );
  }

  async adminFile(id: string): Promise<AdminFileResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/files/${encodeURIComponent(id)}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminFileResponse).data;
  }

  async adminFileUploadPolicy(): Promise<AdminFileUploadPolicyResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/files/upload-policy`);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminFileUploadPolicyResponse).data;
  }

  async adminFileUsages(
    id: string,
  ): Promise<AdminFileUsageListResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/files/${encodeURIComponent(id)}/usages`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminFileUsageListResponse).data;
  }

  async createAdminFileUpload(
    input: AdminFileUploadRequest,
  ): Promise<AdminFileUploadSession> {
    return this.#accessWrite<AdminFileUploadSessionResponse>(
      "/files/upload-sessions",
      "POST",
      input,
    ).then((body) => body.data);
  }

  async adminFileUploadSession(id: string): Promise<AdminFileUploadSession> {
    const response = await this.request(
      `${this.#baseUrl}/files/upload-sessions/${encodeURIComponent(id)}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminFileUploadSessionResponse).data;
  }

  async uploadAdminFile(
    file: File,
    options: {
      sessionId?: string;
      signal?: AbortSignal;
      onSession?: (session: AdminFileUploadSession) => void;
      onProgress?: (percent: number) => void;
    } = {},
  ): Promise<AdminFile> {
    let session = options.sessionId
      ? await this.adminFileUploadSession(options.sessionId)
      : await this.createAdminFileUpload({
          file_name: file.name,
          media_type: file.type || "application/octet-stream",
          size_bytes: file.size,
        });
    options.onSession?.(session);
    const uploaded = new Set(session.uploaded_parts.map((part) => part.part_number));
    const partCount = Math.ceil(file.size / session.part_size);
    let completeBytes = session.uploaded_parts.reduce(
      (total, part) => total + part.size_bytes,
      0,
    );
    options.onProgress?.(Math.floor((completeBytes / file.size) * 100));
    for (let partNumber = 1; partNumber <= partCount; partNumber += 1) {
      if (uploaded.has(partNumber)) continue;
      const start = (partNumber - 1) * session.part_size;
      const content = file.slice(start, Math.min(start + session.part_size, file.size));
      const response = await this.#authorizedFetch(
        `${this.#baseUrl}/files/upload-sessions/${encodeURIComponent(session.id)}/parts/${String(partNumber)}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/octet-stream" },
          body: content,
          ...(options.signal ? { signal: options.signal } : {}),
        },
      );
      if (!response.ok) throw await toApiError(response);
      const part = ((await response.json()) as AdminFilePartResponse).data;
      session = {
        ...session,
        status: "uploading",
        uploaded_parts: [...session.uploaded_parts, part],
      };
      options.onSession?.(session);
      completeBytes += part.size_bytes;
      options.onProgress?.(Math.floor((completeBytes / file.size) * 100));
    }
    const completed = await this.#accessWrite<AdminFileResponse>(
      `/files/upload-sessions/${encodeURIComponent(session.id)}/complete`,
      "POST",
    );
    options.onProgress?.(100);
    return completed.data;
  }

  async cancelAdminFileUpload(id: string): Promise<void> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/files/upload-sessions/${encodeURIComponent(id)}`,
      { method: "DELETE" },
    );
    if (!response.ok) throw await toApiError(response);
  }

  async downloadAdminFile(id: string): Promise<{ file: AdminFile; blob: Blob }> {
    const file = await this.adminFile(id);
    const targetResponse = await this.#authorizedFetch(
      `${this.#baseUrl}/files/${encodeURIComponent(id)}/presign-download`,
      { method: "POST" },
    );
    if (!targetResponse.ok) throw await toApiError(targetResponse);
    const target = ((await targetResponse.json()) as AdminFileDownloadResponse).data;
    const response = await this.request(`${this.#baseUrl}${target.download_url}`);
    if (!response.ok) throw await toApiError(response);
    return { file, blob: await response.blob() };
  }

  async deleteAdminFile(id: string): Promise<void> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/files/${encodeURIComponent(id)}`,
      { method: "DELETE" },
    );
    if (!response.ok) throw await toApiError(response);
  }

  async createAdminConfig(
    input: AdminConfigWriteRequestWritable,
  ): Promise<AdminConfigResponse["data"]> {
    return this.#accessWrite<AdminConfigResponse>(
      "/configs",
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateAdminConfig(
    id: string,
    input: AdminConfigWriteRequestWritable,
  ): Promise<AdminConfigResponse["data"]> {
    return this.#accessWrite<AdminConfigResponse>(
      `/configs/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async rotateAdminConfigSecret(
    id: string,
    input: AdminConfigSecretRequestWritable,
  ): Promise<AdminConfigResponse["data"]> {
    return this.#accessWrite<AdminConfigResponse>(
      `/configs/${encodeURIComponent(id)}/rotate-secret`,
      "POST",
      input,
    ).then((body) => body.data);
  }

  async adminDictionaryTypes(
    filters: AdminDictionaryTypeFilters,
  ): Promise<AdminDictionaryTypeListResponse["data"]> {
    return this.#settingsList<AdminDictionaryTypeListResponse>(
      "/dict-types",
      filters,
    ).then((body) => body.data);
  }

  async createAdminDictionaryType(
    input: AdminDictionaryTypeWriteRequest,
  ): Promise<AdminDictionaryTypeResponse["data"]> {
    return this.#accessWrite<AdminDictionaryTypeResponse>(
      "/dict-types",
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateAdminDictionaryType(
    id: string,
    input: AdminDictionaryTypeWriteRequest,
  ): Promise<AdminDictionaryTypeResponse["data"]> {
    return this.#accessWrite<AdminDictionaryTypeResponse>(
      `/dict-types/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async adminDictionaryItems(
    id: string,
    filters: AdminDictionaryItemFilters,
  ): Promise<AdminDictionaryItemListResponse["data"]> {
    return this.#settingsList<AdminDictionaryItemListResponse>(
      `/dict-types/${encodeURIComponent(id)}/items`,
      filters,
    ).then((body) => body.data);
  }

  async createAdminDictionaryItem(
    id: string,
    input: AdminDictionaryItemWriteRequest,
  ): Promise<AdminDictionaryItemResponse["data"]> {
    return this.#accessWrite<AdminDictionaryItemResponse>(
      `/dict-types/${encodeURIComponent(id)}/items`,
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateAdminDictionaryItem(
    id: string,
    input: AdminDictionaryItemWriteRequest,
  ): Promise<AdminDictionaryItemResponse["data"]> {
    return this.#accessWrite<AdminDictionaryItemResponse>(
      `/dict-items/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async deleteAdminDictionaryItem(
    id: string,
  ): Promise<AdminDeleteResponse["data"]> {
    return this.#accessWrite<AdminDeleteResponse>(
      `/dict-items/${encodeURIComponent(id)}`,
      "DELETE",
    ).then((body) => body.data);
  }

  async adminNotificationMessages(
    kind: "notices" | "messages",
    filters: AdminNotificationMessageFilters,
  ): Promise<AdminNotificationMessageListResponse["data"]> {
    return this.#settingsList<AdminNotificationMessageListResponse>(
      `/${kind}`,
      filters,
    ).then((body) => body.data);
  }

  async adminNotificationMessage(
    kind: "notices" | "messages",
    id: string,
  ): Promise<AdminNotificationMessageResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/${kind}/${encodeURIComponent(id)}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminNotificationMessageResponse).data;
  }

  async createAdminNotificationMessage(
    kind: "notices" | "messages",
    input: AdminNotificationMessageRequest,
  ): Promise<AdminNotificationMessageResponse["data"]> {
    return this.#accessWrite<AdminNotificationMessageResponse>(
      `/${kind}`,
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateAdminNotificationMessage(
    kind: "notices" | "messages",
    id: string,
    input: AdminNotificationMessageRequest,
  ): Promise<AdminNotificationMessageResponse["data"]> {
    return this.#accessWrite<AdminNotificationMessageResponse>(
      `/${kind}/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async previewAdminNotificationRecipients(
    kind: "notices" | "messages",
    id: string,
  ): Promise<AdminNotificationRecipientPreviewResponse["data"]> {
    return this.#accessWrite<AdminNotificationRecipientPreviewResponse>(
      `/${kind}/${encodeURIComponent(id)}/recipient-preview`,
      "POST",
    ).then((body) => body.data);
  }

  async publishAdminNotificationMessage(
    kind: "notices" | "messages",
    id: string,
  ): Promise<AdminNotificationPublishResponse["data"]> {
    return this.#accessWrite<AdminNotificationPublishResponse>(
      `/${kind}/${encodeURIComponent(id)}/publish`,
      "POST",
    ).then((body) => body.data);
  }

  async cancelAdminNotificationMessage(
    kind: "notices" | "messages",
    id: string,
  ): Promise<AdminNotificationMessageResponse["data"]> {
    return this.#accessWrite<AdminNotificationMessageResponse>(
      `/${kind}/${encodeURIComponent(id)}/cancel`,
      "POST",
    ).then((body) => body.data);
  }

  async adminNotificationRecipientStats(
    kind: "notices" | "messages",
    id: string,
  ): Promise<AdminNotificationRecipientStatsResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/${kind}/${encodeURIComponent(id)}/recipients`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminNotificationRecipientStatsResponse)
      .data;
  }

  async adminNotificationTemplates(
    filters: AdminNotificationTemplateFilters,
  ): Promise<AdminNotificationTemplateListResponse["data"]> {
    return this.#settingsList<AdminNotificationTemplateListResponse>(
      "/notification-templates",
      filters,
    ).then((body) => body.data);
  }

  async createAdminNotificationTemplate(
    input: AdminNotificationTemplateRequest,
  ): Promise<AdminNotificationTemplateResponse["data"]> {
    return this.#accessWrite<AdminNotificationTemplateResponse>(
      "/notification-templates",
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateAdminNotificationTemplate(
    id: string,
    input: AdminNotificationTemplateRequest,
  ): Promise<AdminNotificationTemplateResponse["data"]> {
    return this.#accessWrite<AdminNotificationTemplateResponse>(
      `/notification-templates/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async adminNotificationDeliveries(
    filters: AdminNotificationDeliveryFilters,
  ): Promise<AdminNotificationDeliveryListResponse["data"]> {
    return this.#settingsList<AdminNotificationDeliveryListResponse>(
      "/notification-deliveries",
      filters,
    ).then((body) => body.data);
  }

  async adminNotificationDelivery(
    id: string,
  ): Promise<AdminNotificationDeliveryResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/notification-deliveries/${encodeURIComponent(id)}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminNotificationDeliveryResponse).data;
  }

  async retryAdminNotificationDelivery(
    id: string,
  ): Promise<AdminNotificationDeliveryResponse["data"]> {
    return this.#accessWrite<AdminNotificationDeliveryResponse>(
      `/notification-deliveries/${encodeURIComponent(id)}/retry`,
      "POST",
    ).then((body) => body.data);
  }

  async adminJobHandlers(): Promise<AdminJobHandlerListResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/job-handlers`);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminJobHandlerListResponse).data;
  }

  async previewAdminJobSchedule(
    input: AdminJobScheduleRequest,
  ): Promise<AdminJobCronPreviewResponse["data"]> {
    return this.#accessWrite<AdminJobCronPreviewResponse>(
      "/job-schedules/preview",
      "POST",
      input,
    ).then((body) => body.data);
  }

  async adminJobSchedules(
    filters: AdminJobScheduleFilters,
  ): Promise<AdminJobScheduleListResponse["data"]> {
    return this.#settingsList<AdminJobScheduleListResponse>(
      "/job-schedules",
      filters,
    ).then((body) => body.data);
  }

  async createAdminJobSchedule(
    input: AdminJobScheduleRequest,
  ): Promise<AdminJobScheduleResponse["data"]> {
    return this.#accessWrite<AdminJobScheduleResponse>(
      "/job-schedules",
      "POST",
      input,
    ).then((body) => body.data);
  }

  async updateAdminJobSchedule(
    id: string,
    input: AdminJobScheduleRequest,
  ): Promise<AdminJobScheduleResponse["data"]> {
    return this.#accessWrite<AdminJobScheduleResponse>(
      `/job-schedules/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async setAdminJobSchedulePaused(
    id: string,
    paused: boolean,
  ): Promise<AdminJobScheduleResponse["data"]> {
    return this.#accessWrite<AdminJobScheduleResponse>(
      `/job-schedules/${encodeURIComponent(id)}/${paused ? "pause" : "resume"}`,
      "POST",
    ).then((body) => body.data);
  }

  async executeAdminJobSchedule(
    id: string,
    idempotencyKey: string,
  ): Promise<AdminJobRunResponse["data"]> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/job-schedules/${encodeURIComponent(id)}/execute`,
      { method: "POST", headers: { "Idempotency-Key": idempotencyKey } },
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminJobRunResponse).data;
  }

  async adminJobScheduleRuns(
    id: string,
    filters: AdminJobRunFilters,
  ): Promise<AdminJobRunListResponse["data"]> {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters))
      if (value !== undefined && value !== "") params.set(key, String(value));
    const response = await this.request(
      `${this.#baseUrl}/job-schedules/${encodeURIComponent(id)}/runs?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminJobRunListResponse).data;
  }

  async adminApiClients(filters: AdminApiClientFilters): Promise<AdminApiClientListResponse["data"]> {
    return this.#settingsList<AdminApiClientListResponse>("/api-clients", filters).then((body) => body.data);
  }

  async adminApiClient(id: string): Promise<AdminApiClientResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/api-clients/${encodeURIComponent(id)}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminApiClientResponse).data;
  }

  async createAdminApiClient(input: AdminApiClientRequest): Promise<AdminApiClientResponse["data"]> {
    return this.#accessWrite<AdminApiClientResponse>("/api-clients", "POST", input).then((body) => body.data);
  }

  async updateAdminApiClient(id: string, input: AdminApiClientRequest): Promise<AdminApiClientResponse["data"]> {
    return this.#accessWrite<AdminApiClientResponse>(`/api-clients/${encodeURIComponent(id)}`, "PATCH", input).then((body) => body.data);
  }

  async createAdminApiClientSecret(id: string, input: AdminApiClientSecretRequest): Promise<AdminApiClientSecretCreatedResponseWritable["data"]> {
    return this.#accessWrite<AdminApiClientSecretCreatedResponseWritable>(`/api-clients/${encodeURIComponent(id)}/secrets`, "POST", input).then((body) => body.data);
  }

  async revokeAdminApiClientSecret(id: string, secretId: string): Promise<void> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/api-clients/${encodeURIComponent(id)}/secrets/${encodeURIComponent(secretId)}`, { method: "DELETE" });
    if (!response.ok) throw await toApiError(response);
  }

  async replaceAdminApiClientPermissions(id: string, input: AdminApiClientPermissionsRequest): Promise<AdminApiClientResponse["data"]> {
    return this.#accessWrite<AdminApiClientResponse>(`/api-clients/${encodeURIComponent(id)}/permissions`, "PUT", input).then((body) => body.data);
  }

  async adminWebhooks(filters: AdminWebhookFilters): Promise<AdminWebhookListResponse["data"]> {
    return this.#settingsList<AdminWebhookListResponse>("/webhooks", filters).then((body) => body.data);
  }

  async createAdminWebhook(input: AdminWebhookRequest): Promise<AdminWebhookCreatedResponseWritable["data"]> {
    return this.#accessWrite<AdminWebhookCreatedResponseWritable>("/webhooks", "POST", input).then((body) => body.data);
  }

  async updateAdminWebhook(id: string, input: AdminWebhookRequest): Promise<AdminWebhookResponse["data"]> {
    return this.#accessWrite<AdminWebhookResponse>(`/webhooks/${encodeURIComponent(id)}`, "PATCH", input).then((body) => body.data);
  }

  async testAdminWebhook(id: string, idempotencyKey: string, input: AdminWebhookTestRequest): Promise<AdminWebhookDeliveryResponse["data"]> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}/webhooks/${encodeURIComponent(id)}/test`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "Idempotency-Key": idempotencyKey },
      body: JSON.stringify(input),
    });
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminWebhookDeliveryResponse).data;
  }

  async adminWebhookDeliveries(id: string, page: number, pageSize: number): Promise<AdminWebhookDeliveryListResponse["data"]> {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
    const response = await this.request(`${this.#baseUrl}/webhooks/${encodeURIComponent(id)}/deliveries?${params.toString()}`);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminWebhookDeliveryListResponse).data;
  }

  async adminBlockRules(filters: AdminBlockRuleFilters): Promise<AdminBlockRuleListResponse["data"]> {
    return this.#settingsList<AdminBlockRuleListResponse>("/block-rules", filters).then((body) => body.data);
  }

  async createAdminBlockRule(input: AdminBlockRuleCreateRequestWritable): Promise<AdminBlockRuleResponse["data"]> {
    return this.#accessWrite<AdminBlockRuleResponse>("/block-rules", "POST", input).then((body) => body.data);
  }

  async updateAdminBlockRule(id: string, input: AdminBlockRuleUpdateRequest): Promise<AdminBlockRuleResponse["data"]> {
    return this.#accessWrite<AdminBlockRuleResponse>(`/block-rules/${encodeURIComponent(id)}`, "PATCH", input).then((body) => body.data);
  }

  async revokeAdminBlockRule(id: string): Promise<AdminBlockRuleRevokeResponse["data"]> {
    return this.#accessWrite<AdminBlockRuleRevokeResponse>(`/block-rules/${encodeURIComponent(id)}`, "DELETE").then((body) => body.data);
  }

  async adminOpsHealth(): Promise<AdminOpsHealthResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/ops/health`);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminOpsHealthResponse).data;
  }

  async adminOpsRuntime(): Promise<AdminOpsRuntimeResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/ops/runtime-summary`);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminOpsRuntimeResponse).data;
  }

  clearLocalSession(): void {
    this.#tokens.clear();
    this.#clearTenantCache();
  }

  async adminUser(id: string): Promise<AdminUserResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/users/${encodeURIComponent(id)}`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminUserResponse).data;
  }

  async adminUserRoleOptions(): Promise<AdminUserRoleOptionsResponse["data"]> {
    const response = await this.request(`${this.#baseUrl}/users/role-options`);
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminUserRoleOptionsResponse).data;
  }

  async createAdminUser(
    input: AdminUserCreateRequestWritable,
  ): Promise<AdminUserResponse["data"]> {
    return this.#userWrite<AdminUserResponse>("/users", "POST", input).then(
      (body) => body.data,
    );
  }

  async updateAdminUser(
    id: string,
    input: AdminUserUpdateRequest,
  ): Promise<AdminUserResponse["data"]> {
    return this.#userWrite<AdminUserResponse>(
      `/users/${encodeURIComponent(id)}`,
      "PATCH",
      input,
    ).then((body) => body.data);
  }

  async setAdminUserEnabled(
    id: string,
    enabled: boolean,
  ): Promise<AdminUserResponse["data"]> {
    return this.#userWrite<AdminUserResponse>(
      `/users/${encodeURIComponent(id)}/${enabled ? "enable" : "disable"}`,
      "POST",
    ).then((body) => body.data);
  }

  async unlockAdminUser(id: string): Promise<AdminUserActionResponse["data"]> {
    return this.#userWrite<AdminUserActionResponse>(
      `/users/${encodeURIComponent(id)}/unlock`,
      "POST",
    ).then((body) => body.data);
  }

  async resetAdminUserPassword(
    id: string,
    input: AdminUserResetPasswordRequest,
  ): Promise<AdminUserResetPasswordResponse["data"]> {
    return this.#userWrite<AdminUserResetPasswordResponse>(
      `/users/${encodeURIComponent(id)}/reset-password`,
      "POST",
      input,
    ).then((body) => body.data);
  }

  async replaceAdminUserRoles(
    id: string,
    input: AdminUserRolesRequest,
  ): Promise<AdminUserResponse["data"]> {
    return this.#userWrite<AdminUserResponse>(
      `/users/${encodeURIComponent(id)}/roles`,
      "PUT",
      input,
    ).then((body) => body.data);
  }

  async replaceAdminUserAssignments(
    id: string,
    input: AdminUserAssignmentsRequest,
  ): Promise<AdminUserResponse["data"]> {
    return this.#userWrite<AdminUserResponse>(
      `/org/users/${encodeURIComponent(id)}/assignments`,
      "PUT",
      input,
    ).then((body) => body.data);
  }

  async adminUserSessions(
    id: string,
  ): Promise<AdminUserSessionListResponse["data"]> {
    const response = await this.request(
      `${this.#baseUrl}/users/${encodeURIComponent(id)}/sessions`,
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminUserSessionListResponse).data;
  }

  async revokeAdminUserSession(
    id: string,
    sessionId: string,
  ): Promise<AdminUserActionResponse["data"]> {
    return this.#userWrite<AdminUserActionResponse>(
      `/users/${encodeURIComponent(id)}/sessions/${encodeURIComponent(sessionId)}`,
      "DELETE",
    ).then((body) => body.data);
  }

  async importAdminUsers(
    csv: string,
  ): Promise<AdminUserImportResponse["data"]> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/users/import`,
      {
        method: "POST",
        headers: { "Content-Type": "text/csv; charset=utf-8" },
        body: csv,
      },
    );
    if (!response.ok) throw await toApiError(response);
    return ((await response.json()) as AdminUserImportResponse).data;
  }

  async exportAdminUsers(input: AdminUserExportRequest): Promise<Blob> {
    const response = await this.#authorizedFetch(
      `${this.#baseUrl}/users/export`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      },
    );
    if (!response.ok) throw await toApiError(response);
    return response.blob();
  }

  async logout(): Promise<void> {
    const snapshot = this.#tokens.read();
    try {
      if (snapshot) {
        await this.#fetch(`${this.#baseUrl}/auth/logout`, {
          method: "POST",
          credentials: "include",
          headers: {
            "Accept-Language": this.#readLocale(),
            Authorization: `Bearer ${snapshot.accessToken}`,
            "X-CSRF-Token": snapshot.csrfToken,
          },
        });
      }
    } finally {
      this.#tokens.clear();
      this.#clearTenantCache();
    }
  }

  async #refresh(): Promise<string> {
    const snapshot = this.#tokens.read();
    if (!snapshot) throw new Error("AUTH_SESSION_MISSING");
    const response = await this.#fetch(`${this.#baseUrl}/auth/token/refresh`, {
      method: "POST",
      credentials: "include",
      headers: {
        "Accept-Language": this.#readLocale(),
        "X-CSRF-Token": snapshot.csrfToken,
      },
    });
    if (!response.ok) {
      this.#tokens.clear();
      this.#clearTenantCache();
      throw await toApiError(response);
    }
    const body = (await response.json()) as TokenResponse;
    this.#storeTokenResponse(body);
    return body.data.access_token;
  }

  #authorizedFetch(input: string | URL, init: RequestInit): Promise<Response> {
    const headers = new Headers(init.headers);
    headers.set("Accept-Language", this.#readLocale());
    const accessToken = this.#tokens.read()?.accessToken;
    if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
    return this.#fetch(input, { ...init, credentials: "include", headers });
  }

  #anonymousWrite(path: string, input: object): Promise<Response> {
    return this.#fetch(`${this.#baseUrl}${path}`, {
      method: "POST",
      credentials: "include",
      headers: {
        "Accept-Language": this.#readLocale(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(input),
    });
  }

  async #orgWrite<T>(
    path: string,
    method: "POST" | "PATCH" | "DELETE",
    input?: object,
  ): Promise<T> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}${path}`, {
      method,
      ...(input
        ? {
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(input),
          }
        : {}),
    });
    if (!response.ok) throw await toApiError(response);
    return (await response.json()) as T;
  }

  async #userWrite<T>(
    path: string,
    method: "POST" | "PATCH" | "PUT" | "DELETE",
    input?: object,
  ): Promise<T> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}${path}`, {
      method,
      ...(input
        ? {
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(input),
          }
        : {}),
    });
    if (!response.ok) throw await toApiError(response);
    return (await response.json()) as T;
  }

  async #tenantWrite<T>(
    path: string,
    method: "POST" | "PATCH" | "DELETE",
    input?: object,
  ): Promise<T> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}${path}`, {
      method,
      ...(input
        ? {
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(input),
          }
        : {}),
    });
    if (!response.ok) throw await toApiError(response);
    return (await response.json()) as T;
  }

  async #accessWrite<T>(
    path: string,
    method: "POST" | "PATCH" | "PUT" | "DELETE",
    input?: object,
  ): Promise<T> {
    const response = await this.#authorizedFetch(`${this.#baseUrl}${path}`, {
      method,
      ...(input
        ? {
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(input),
          }
        : {}),
    });
    if (!response.ok) throw await toApiError(response);
    return (await response.json()) as T;
  }

  async #auditList<T>(path: string, filters: AdminAuditFilters): Promise<T> {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters))
      if (value !== undefined && value !== "") params.set(key, String(value));
    const response = await this.request(
      `${this.#baseUrl}${path}?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return (await response.json()) as T;
  }

  async #settingsList<T>(path: string, filters: object): Promise<T> {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters))
      if (value !== undefined && value !== "") params.set(key, String(value));
    const response = await this.request(
      `${this.#baseUrl}${path}?${params.toString()}`,
    );
    if (!response.ok) throw await toApiError(response);
    return (await response.json()) as T;
  }

  #storeTokenResponse(response: TokenResponse): void {
    this.#tokens.update({
      accessToken: response.data.access_token,
      csrfToken: response.data.csrf_token,
    });
  }
}
