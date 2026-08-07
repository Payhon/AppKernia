import { Alert, Card, Select, Space, Tag, Typography } from "antd";
import { useNavigate, useRouterState } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useManagedApplications } from "./hooks";

export interface AppScope { appId: string | null; appName: string | null; disabled: boolean; }

function readAppId(search: string): string | null {
  const value = new URLSearchParams(search).get("app_id");
  return value && /^[0-9a-f-]{36}$/i.test(value) ? value : null;
}

export function useAppScope(): AppScope {
  const search = useRouterState({ select: (state) => state.location.searchStr });
  const apps = useManagedApplications({ q: "", page: 1, page_size: 100 });
  const appId = readAppId(search);
  const application = apps.data?.items.find((item) => item.id === appId) ?? null;
  return { appId, appName: application?.name ?? null, disabled: application?.status === "disabled" };
}

export function AppScopeContext({ children }: { children: (scope: AppScope) => React.ReactNode }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const apps = useManagedApplications({ q: "", page: 1, page_size: 100 });
  const scope = useAppScope();
  const selectApp = (appId: string | undefined) => {
    void navigate({
      to: pathname as never,
      search: ((previous: Record<string, unknown>) => ({ ...previous, ...(appId ? { app_id: appId } : { app_id: undefined }) })) as never,
    });
  };
  return <>
    <Card className="ak-app-scope-context" size="small">
      <Space align="center" wrap>
        <Typography.Text strong>{t("apps.scope.label")}</Typography.Text>
        <Select
          allowClear
          aria-label={t("apps.scope.label")}
          loading={apps.isPending}
          onChange={selectApp}
          options={(apps.data?.items ?? []).map((item) => ({ value: item.id, label: item.name }))}
          placeholder={t("apps.scope.placeholder")}
          value={scope.appId ?? undefined}
        />
        {scope.appName ? <Typography.Text type="secondary">{scope.appName}</Typography.Text> : null}
        {scope.appId && scope.disabled ? <Tag className="ak-status-error">{t("apps.status.disabled")}</Tag> : null}
      </Space>
    </Card>
    {apps.isError ? <Alert className="ak-app-scope-alert" showIcon type="error" title={t("apps.feedback.load_error")} /> : null}
    {!scope.appId && !apps.isPending ? <Alert className="ak-app-scope-alert" showIcon type="info" title={t("apps.scope.required")} /> : null}
    {children(scope)}
  </>;
}
