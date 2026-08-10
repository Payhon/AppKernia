import { Select, Tag, Typography } from "antd";
import { useNavigate, useRouterState } from "@tanstack/react-router";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";

import { useManagedApplications } from "../features/apps/hooks";
import { readAppId, resolveAppId, withSelectedApp } from "../features/apps/scope";
import { useAppSelectionStore } from "../features/apps/selection-store";
import { useAuthStore } from "../features/auth/store";

export function GlobalAppSelector() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const search = useRouterState({ select: (state) => state.location.searchStr });
  const tenantId = useAuthStore((state) => state.context?.active_tenant.id ?? null);
  const rememberedAppId = useAppSelectionStore((state) => tenantId ? state.appIdByTenant[tenantId] ?? null : null);
  const setSelection = useAppSelectionStore((state) => state.setSelection);
  const applications = useManagedApplications({ q: "", page: 1, page_size: 100 });
  const urlAppId = readAppId(search);
  const appId = resolveAppId(urlAppId, rememberedAppId);
  const selectedApplication = applications.data?.items.find((item) => item.id === appId);
  const applicationsById = useMemo(
    () => new Map((applications.data?.items ?? []).map((item) => [item.id, item])),
    [applications.data?.items],
  );

  useEffect(() => {
    if (!tenantId || !applications.data) return;
    const availableIds = new Set(applications.data.items.map((item) => item.id));
    const applicationListIsComplete = applications.data.items.length >= applications.data.total;

    if (urlAppId) {
      if (availableIds.has(urlAppId)) {
        if (rememberedAppId !== urlAppId) setSelection(tenantId, urlAppId);
        return;
      }
      if (!applicationListIsComplete) return;
      const fallbackAppId = rememberedAppId && availableIds.has(rememberedAppId) ? rememberedAppId : undefined;
      if (rememberedAppId && !fallbackAppId) setSelection(tenantId, null);
      void navigate({
        to: pathname as never,
        search: ((previous: Record<string, unknown>) => withSelectedApp(previous, fallbackAppId)) as never,
        replace: true,
      });
      return;
    }

    if (!rememberedAppId) return;
    if (!availableIds.has(rememberedAppId) && applicationListIsComplete) {
      setSelection(tenantId, null);
      return;
    }
    void navigate({
      to: pathname as never,
      search: ((previous: Record<string, unknown>) => withSelectedApp(previous, rememberedAppId)) as never,
      replace: true,
    });
  }, [applications.data, navigate, pathname, rememberedAppId, setSelection, tenantId, urlAppId]);

  const selectApp = (value: string | undefined) => {
    if (tenantId) setSelection(tenantId, value ?? null);
    void navigate({
      to: pathname as never,
      search: ((previous: Record<string, unknown>) => withSelectedApp(previous, value)) as never,
    });
  };

  return (
    <div aria-label={t("apps.scope.label")} className="ak-global-app-context" role="group">
      <Typography.Text className="ak-global-app-context-label" type="secondary">
        {t("apps.scope.label")}
      </Typography.Text>
      <div className="ak-global-app-context-control">
        <Select
          allowClear
          aria-label={t("apps.scope.label")}
          className="ak-global-app-select"
          loading={applications.isPending}
          notFoundContent={t(applications.isError ? "apps.feedback.load_error" : applications.isPending ? "common.states.loading" : "apps.application.empty")}
          onChange={selectApp}
          optionRender={(option) => {
            const application = applicationsById.get(String(option.value));
            return application ? (
              <div className="ak-global-app-option">
                <span>{application.name}</span>
                <code>{application.appid_pending ? t("apps.application.values.appid_pending") : application.appid}</code>
              </div>
            ) : option.label;
          }}
          options={(applications.data?.items ?? []).map((item) => ({
            value: item.id,
            label: item.status === "disabled" ? `${item.name} · ${t("apps.status.disabled")}` : item.name,
          }))}
          placeholder={t("apps.scope.placeholder")}
          popupMatchSelectWidth={280}
          showSearch={{ optionFilterProp: "label" }}
          value={appId ?? undefined}
          {...(applications.isError ? { status: "error" as const } : {})}
        />
        {selectedApplication?.status === "disabled" ? <Tag className="ak-global-app-status ak-status-error">{t("apps.status.disabled")}</Tag> : null}
      </div>
      {applications.isError ? <span className="ak-sr-only" role="alert">{t("apps.feedback.load_error")}</span> : null}
    </div>
  );
}
