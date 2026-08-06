import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Row,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { useTranslation } from "react-i18next";

import { useOpsHealth, useOpsRuntime } from "../features/ops/hooks";
import type {
  AdminOpsDependency,
  AdminOpsModule,
  AdminOpsStatus,
} from "../generated/api/types.gen";

export const enabledCapabilities = (
  capabilities: AdminOpsModule["capabilities"],
): string[] =>
  Object.entries(capabilities)
    .filter(([, enabled]) => enabled)
    .map(([code]) => code)
    .sort((left, right) => left.localeCompare(right));

const statusClass = (value: AdminOpsStatus) => {
  if (value === "ready") return "ak-status-success";
  if (
    value === "degraded" ||
    value === "unknown" ||
    value === "not_configured"
  )
    return "ak-status-warning";
  return "ak-status-error";
};

export function AdminServiceHealthPage() {
  const { i18n, t } = useTranslation();
  const health = useOpsHealth();
  const runtime = useOpsRuntime();
  const locale = i18n.resolvedLanguage ?? "zh-CN";

  const refresh = () => Promise.all([health.refetch(), runtime.refetch()]);
  const copy = async () => {
    if (!health.data || !runtime.data) return;
    await navigator.clipboard.writeText(
      JSON.stringify({ health: health.data, runtime: runtime.data }, null, 2),
    );
  };
  const formatDate = (value: string) =>
    new Intl.DateTimeFormat(locale, {
      dateStyle: "medium",
      timeStyle: "medium",
    }).format(new Date(value));

  const dependencyColumns: TableColumnsType<AdminOpsDependency> = [
    {
      title: t("ops.columns.dependency"),
      dataIndex: "code",
      render: (value: string) =>
        t(`ops.dependency.${value}`, { defaultValue: value }),
    },
    {
      title: t("ops.columns.status"),
      dataIndex: "status",
      render: (value: AdminOpsStatus) => (
        <Tag className={statusClass(value)}>
          {t(`ops.status.${value}`, { defaultValue: value })}
        </Tag>
      ),
    },
    {
      title: t("ops.columns.latency"),
      dataIndex: "latency_ms",
      render: (value: number) => t("ops.latency", { value }),
    },
    {
      title: t("ops.columns.checked"),
      dataIndex: "checked_at",
      render: (value: string) => formatDate(value),
    },
  ];

  const moduleColumns: TableColumnsType<AdminOpsModule> = [
    {
      title: t("ops.columns.module"),
      dataIndex: "code",
      width: 300,
      render: (code: string, module) => (
        <div className="ak-ops-module-identity">
          <Typography.Text strong>
            {t(module.name_key, { defaultValue: code })}
          </Typography.Text>
          <Typography.Text code>{code}</Typography.Text>
          <Typography.Text type="secondary">
            {t(module.description_key, { defaultValue: code })}
          </Typography.Text>
        </div>
      ),
    },
    {
      title: t("ops.columns.capabilities"),
      dataIndex: "capabilities",
      width: 420,
      render: (capabilities: AdminOpsModule["capabilities"]) => (
        <Space size={[4, 6]} wrap>
          {enabledCapabilities(capabilities).map((capability) => (
            <Tag key={capability}>
              {t(`ops.capabilities.${capability}`, {
                defaultValue: capability,
              })}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: t("ops.columns.version"),
      dataIndex: "version",
      width: 120,
    },
    {
      title: t("ops.columns.status"),
      dataIndex: "status",
      width: 110,
      render: (value: AdminOpsModule["status"]) => (
        <Tag className={value === "enabled" ? "ak-status-success" : "ak-status-warning"}>
          {t(`ops.module_status.${value}`, { defaultValue: value })}
        </Tag>
      ),
    },
  ];

  const failed = health.isError || runtime.isError;
  const pendingJobs =
    (runtime.data?.queue.available ?? 0) +
    (runtime.data?.queue.retryable ?? 0) +
    (runtime.data?.queue.scheduled ?? 0);

  return (
    <div className="ak-page-container">
      <header className="ak-page-heading ak-org-heading">
        <div>
          <Typography.Title level={1}>{t("ops.title")}</Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("ops.description")}
          </Typography.Paragraph>
        </div>
        <Space wrap>
          <Button
            onClick={() => void copy()}
            disabled={!health.data || !runtime.data}
          >
            {t("ops.actions.copy")}
          </Button>
          <Button
            type="primary"
            loading={health.isFetching || runtime.isFetching}
            onClick={() => void refresh()}
          >
            {t("common.actions.refresh")}
          </Button>
        </Space>
      </header>

      <div className="ak-ops-stack">
        <Alert
          showIcon
          type="info"
          title={t("ops.safety.title")}
          description={t("ops.safety.description")}
        />

        {failed ? (
          <div className="ak-error-panel" role="alert">
            <span>{t("errors.common.unknown")}</span>
            <Button onClick={() => void refresh()}>
              {t("common.actions.retry")}
            </Button>
          </div>
        ) : null}

        <Row gutter={[16, 16]} align="stretch">
          <Col xs={24} md={8}>
            <Card className="ak-ops-summary-card">
              <Statistic
                title={t("ops.summary.overall")}
                value={
                  health.data
                    ? t(`ops.status.${health.data.status}`)
                    : t("ops.loading")
                }
                styles={{ content: { fontSize: 18 } }}
              />
            </Card>
          </Col>
          <Col xs={24} md={8}>
            <Card className="ak-ops-summary-card">
              <Statistic
                title={t("ops.summary.worker")}
                value={
                  runtime.data
                    ? t(`ops.status.${runtime.data.queue.status}`)
                    : t("ops.loading")
                }
                styles={{ content: { fontSize: 18 } }}
              />
            </Card>
          </Col>
          <Col xs={24} md={8}>
            <Card className="ak-ops-summary-card">
              <Statistic title={t("ops.summary.pending")} value={pendingJobs} />
            </Card>
          </Col>
        </Row>

        <Card title={t("ops.dependencies.title")}>
          <div
            className="ak-ops-table-scroll ak-ops-dependencies-table"
            tabIndex={0}
            role="region"
            aria-label={t("ops.dependencies.table_label")}
          >
            <Table
              rowKey="code"
              size="small"
              pagination={false}
              loading={health.isPending}
              dataSource={health.data?.dependencies ?? []}
              columns={dependencyColumns}
            />
          </div>
        </Card>

        <Card title={t("ops.runtime.title")}>
          <div className="ak-ops-runtime-stack">
            <Descriptions
              column={{ xs: 1, sm: 2, md: 3 }}
              items={[
                {
                  key: "app",
                  label: t("ops.runtime.app_version"),
                  children: runtime.data?.app_version ?? "—",
                },
                {
                  key: "go",
                  label: t("ops.runtime.go_version"),
                  children: runtime.data?.go_version ?? "—",
                },
                {
                  key: "uptime",
                  label: t("ops.runtime.uptime"),
                  children: t("ops.uptime", {
                    value: runtime.data?.uptime_seconds ?? 0,
                  }),
                },
                {
                  key: "heartbeat",
                  label: t("ops.runtime.heartbeat"),
                  children: runtime.data?.queue.last_heartbeat_at
                    ? formatDate(runtime.data.queue.last_heartbeat_at)
                    : t("ops.status.unknown"),
                },
                {
                  key: "runs",
                  label: t("ops.runtime.success_runs"),
                  children: runtime.data?.schedule_runs_24h.succeeded ?? 0,
                },
                {
                  key: "failures",
                  label: t("ops.runtime.failed_runs"),
                  children: runtime.data?.schedule_runs_24h.failed ?? 0,
                },
              ]}
            />

            <Typography.Title level={2} className="ak-ops-modules-title">
              {t("ops.modules.title")}
            </Typography.Title>
            <div
              className="ak-ops-table-scroll ak-ops-modules-table"
              tabIndex={0}
              role="region"
              aria-label={t("ops.modules.table_label")}
            >
              <Table
                rowKey="code"
                size="small"
                pagination={false}
                loading={runtime.isPending}
                dataSource={runtime.data?.modules ?? []}
                columns={moduleColumns}
                locale={{ emptyText: t("ops.modules.empty") }}
              />
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
}
