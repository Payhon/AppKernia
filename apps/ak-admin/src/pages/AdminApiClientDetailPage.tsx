import {
  Alert,
  Button,
  Card,
  Descriptions,
  Skeleton,
  Space,
  Tag,
  Typography,
} from "antd";
import { useNavigate } from "@tanstack/react-router";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import { useApiClient } from "../features/api-clients/hooks";
import { ApiError } from "../shared/api/error";

export function AdminApiClientDetailPage({ clientId }: { clientId: string }) {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const query = useApiClient(clientId);
  const formatDate = useMemo(
    () => (value?: string | null) =>
      value
        ? new Intl.DateTimeFormat(i18n.language, {
            dateStyle: "medium",
            timeStyle: "medium",
          }).format(new Date(value))
        : t("common.not_available"),
    [i18n.language, t],
  );
  const goBack = () =>
    navigate({
      to: "/system/integrations/api-clients",
      search: { q: "", status: "", page: 1, page_size: 20 },
    });

  return (
    <div className="ak-page-container">
      <header className="ak-page-heading">
        <div>
          <Typography.Title level={1}>
            {t("routes.system.integrations.api-client-detail.title")}
          </Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("api_clients.detail.description")}
          </Typography.Paragraph>
        </div>
        <Button onClick={() => void goBack()}>
          {t("api_clients.detail.back")}
        </Button>
      </header>

      {query.isPending ? (
        <Card aria-live="polite">
          <Skeleton active paragraph={{ rows: 8 }} />
        </Card>
      ) : null}

      {query.isError ? (
        <Alert
          role="alert"
          showIcon
          type="error"
          title={
            query.error instanceof ApiError && query.error.status === 404
              ? t("routes.errors.not-found.title")
              : t("api_clients.detail.load_error")
          }
          action={
            <Button onClick={() => void query.refetch()}>
              {t("common.actions.retry")}
            </Button>
          }
        />
      ) : null}

      {query.data ? (
        <Space orientation="vertical" size="large" style={{ width: "100%" }}>
          <Card title={query.data.name}>
            <Descriptions column={{ xs: 1, md: 2, lg: 3 }} bordered>
              <Descriptions.Item label={t("api_clients.detail.client_id")}>
                <code>{query.data.client_id}</code>
              </Descriptions.Item>
              <Descriptions.Item label={t("api_clients.columns.status")}>
                <Tag
                  className={
                    query.data.status === "active"
                      ? "ak-status-success"
                      : "ak-status-error"
                  }
                >
                  {t(`api_clients.status.${query.data.status}`)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t("api_clients.editor.description")}>
                {query.data.description || t("common.not_available")}
              </Descriptions.Item>
              <Descriptions.Item label={t("api_clients.editor.cidrs")}>
                <Space wrap>
                  {query.data.allowed_cidrs.length > 0
                    ? query.data.allowed_cidrs.map((cidr) => (
                        <Tag key={cidr}>{cidr}</Tag>
                      ))
                    : t("api_clients.cidrs.any")}
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label={t("api_clients.detail.created_at")}>
                {formatDate(query.data.created_at)}
              </Descriptions.Item>
              <Descriptions.Item label={t("api_clients.detail.updated_at")}>
                {formatDate(query.data.updated_at)}
              </Descriptions.Item>
              <Descriptions.Item label={t("api_clients.detail.expires_at")}>
                {formatDate(query.data.expires_at)}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title={t("api_clients.detail.permissions")}>
            <Space wrap>
              {query.data.permissions.length > 0
                ? query.data.permissions.map((permission) => (
                    <Tag key={permission}>{permission}</Tag>
                  ))
                : t("api_clients.detail.no_permissions")}
            </Space>
          </Card>

          <Card title={t("api_clients.secrets.title")}>
            {query.data.secrets.length > 0 ? (
              <Space orientation="vertical" style={{ width: "100%" }}>
                {query.data.secrets.map((secret) => (
                  <Card key={secret.id} size="small">
                    <Descriptions column={{ xs: 1, md: 3 }}>
                      <Descriptions.Item label={t("api_clients.secret.value")}>
                        <code>{secret.prefix}</code>
                      </Descriptions.Item>
                      <Descriptions.Item label={t("api_clients.detail.created_at")}>
                        {formatDate(secret.created_at)}
                      </Descriptions.Item>
                      <Descriptions.Item label={t("api_clients.columns.status")}>
                        <Tag
                          className={
                            secret.revoked_at
                              ? "ak-status-error"
                              : "ak-status-success"
                          }
                        >
                          {t(
                            secret.revoked_at
                              ? "api_clients.secrets.revoked"
                              : "api_clients.secrets.active",
                          )}
                        </Tag>
                      </Descriptions.Item>
                    </Descriptions>
                  </Card>
                ))}
              </Space>
            ) : (
              <Typography.Text type="secondary">
                {t("api_clients.detail.no_secrets")}
              </Typography.Text>
            )}
          </Card>
        </Space>
      ) : null}
    </div>
  );
}
