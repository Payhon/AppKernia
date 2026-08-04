import {
  Alert,
  Button,
  Card,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { useJobScheduleRuns } from "../features/schedules/hooks";
import type { AdminJobRun } from "../generated/api/types.gen";

interface Filters {
  status: string;
  page: number;
  page_size: number;
}

function readFilters(): Filters {
  const params = new URLSearchParams(location.search);
  return {
    status: params.get("status") ?? "",
    page: Math.max(1, Number(params.get("page") ?? 1)),
    page_size: Math.max(1, Number(params.get("page_size") ?? 20)),
  };
}

function persistFilters(filters: Filters) {
  const params = new URLSearchParams();
  if (filters.status) params.set("status", filters.status);
  if (filters.page !== 1) params.set("page", String(filters.page));
  if (filters.page_size !== 20)
    params.set("page_size", String(filters.page_size));
  history.replaceState(
    history.state,
    "",
    `${location.pathname}${params.size ? `?${params.toString()}` : ""}`,
  );
}

export function AdminJobScheduleRunsPage({
  scheduleId,
}: {
  scheduleId: string;
}) {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const [filters, setFilterState] = useState(readFilters);
  const setFilters = (next: Filters) => {
    setFilterState(next);
    persistFilters(next);
  };
  const runs = useJobScheduleRuns(scheduleId, filters);
  const formatDate = useMemo(
    () => (value: string) =>
      new Intl.DateTimeFormat(i18n.language, {
        dateStyle: "medium",
        timeStyle: "long",
        timeZone: "UTC",
      }).format(new Date(value)),
    [i18n.language],
  );
  const columns: TableColumnsType<AdminJobRun> = [
    {
      title: t("schedules.runs.columns.status"),
      dataIndex: "status",
      width: 120,
      render: (value: string) => (
        <Tag
          className={
            value === "succeeded"
              ? "ak-status-success"
              : value === "failed" || value === "cancelled"
                ? "ak-status-error"
                : "ak-status-warning"
          }
        >
          {t(`schedules.run_status.${value}`)}
        </Tag>
      ),
    },
    {
      title: t("schedules.runs.columns.trigger"),
      dataIndex: "trigger_type",
      width: 120,
      render: (value: string) => t(`schedules.trigger.${value}`),
    },
    {
      title: t("schedules.runs.columns.scheduled"),
      dataIndex: "scheduled_at",
      width: 220,
      render: (value: string) => formatDate(value),
    },
    {
      title: t("schedules.runs.columns.attempt"),
      dataIndex: "attempt",
      width: 100,
    },
    {
      title: t("schedules.runs.columns.result"),
      key: "result",
      width: 320,
      render: (_, item) => (
        <div>
          {item.error_code ? <code>{item.error_code}</code> : null}
          {item.error_summary ? (
            <div className="ak-table-secondary">{item.error_summary}</div>
          ) : null}
          {item.output ? (
            <pre className="ak-schedule-run-output">
              {JSON.stringify(item.output, null, 2)}
            </pre>
          ) : null}
          {!item.error_code && !item.output ? "—" : null}
        </div>
      ),
    },
  ];

  return (
    <div className="ak-page-container">
      <header className="ak-page-heading">
        <div>
          <Typography.Title level={1}>
            {t("schedules.runs.page_title")}
          </Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("schedules.runs.description", { id: scheduleId })}
          </Typography.Paragraph>
        </div>
        <Button
          onClick={() =>
            void navigate({
              to: "/system/integrations/schedules",
              search: {
                q: "",
                status: "",
                time_zone: "",
                page: 1,
                page_size: 20,
              },
            })
          }
        >
          {t("schedules.runs.back")}
        </Button>
      </header>

      <Card>
        <Space orientation="vertical" size="middle" style={{ width: "100%" }}>
          <Typography.Text>
            {t("schedules.runs.schedule_id")}: <code>{scheduleId}</code>
          </Typography.Text>
          <div role="search">
            <Select
              allowClear
              aria-label={t("schedules.runs.filters.status")}
              placeholder={t("schedules.runs.filters.status")}
              value={filters.status || undefined}
              onChange={(value) => {
                setFilters({ ...filters, status: value ?? "", page: 1 });
              }}
              options={[
                "queued",
                "running",
                "succeeded",
                "failed",
                "cancelled",
                "skipped",
              ].map((value) => ({
                value,
                label: t(`schedules.run_status.${value}`),
              }))}
            />
          </div>
          {runs.isError ? (
            <Alert
              role="alert"
              showIcon
              type="error"
              title={t("schedules.runs.load_error")}
              action={
                <Button onClick={() => void runs.refetch()}>
                  {t("common.actions.retry")}
                </Button>
              }
            />
          ) : null}
          <div
            className="ak-table-scroll"
            tabIndex={0}
            role="region"
            aria-label={t("schedules.runs.table_landmark")}
            ref={(node) => {
              node
                ?.querySelector(".ant-table-content")
                ?.setAttribute("tabindex", "0");
            }}
          >
            <Table
              rowKey="id"
              columns={columns}
              dataSource={runs.data?.items ?? []}
              loading={runs.isPending}
              locale={{ emptyText: t("schedules.runs.empty") }}
              scroll={{ x: 880 }}
              pagination={{
                current: filters.page,
                pageSize: filters.page_size,
                total: runs.data?.total ?? 0,
                showSizeChanger: true,
                onChange: (page, page_size) => {
                  setFilters({ ...filters, page, page_size });
                },
              }}
            />
          </div>
        </Space>
      </Card>
    </div>
  );
}
