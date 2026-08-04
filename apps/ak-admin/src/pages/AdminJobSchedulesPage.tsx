import {
  Alert,
  Button,
  Card,
  Descriptions,
  Drawer,
  Form,
  Grid,
  Input,
  InputNumber,
  List,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { Controller, useForm } from "react-hook-form";
import { useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type {
  AdminJobRun,
  AdminJobSchedule,
} from "../generated/api/types.gen";
import { useAuthStore } from "../features/auth/store";
import {
  useJobHandlers,
  useJobScheduleMutations,
  useJobScheduleRuns,
  useJobSchedules,
} from "../features/schedules/hooks";
import {
  cronDescriptionKey,
  emptyScheduleValues,
  fromSchedule,
  scheduleValuesSchema,
  toScheduleRequest,
  type ScheduleValues,
} from "../features/schedules/schema";

interface Filters {
  q: string;
  status: string;
  time_zone: string;
  page: number;
  page_size: number;
}

interface RunFilters {
  status: string;
  page: number;
  page_size: number;
}

const timeZones = [
  "UTC",
  "Asia/Shanghai",
  "Asia/Tokyo",
  "Europe/London",
  "America/New_York",
];

function readFilters(): Filters {
  const params = new URLSearchParams(location.search);
  return {
    q: params.get("q") ?? "",
    status: params.get("status") ?? "",
    time_zone: params.get("time_zone") ?? "",
    page: Math.max(1, Number(params.get("page") ?? 1)),
    page_size: Math.max(1, Number(params.get("page_size") ?? 20)),
  };
}

function persistFilters(filters: Filters) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (
      value !== "" &&
      !(key === "page" && value === 1) &&
      !(key === "page_size" && value === 20)
    ) {
      params.set(key, String(value));
    }
  }
  history.replaceState(
    history.state,
    "",
    `${location.pathname}${params.size ? `?${params.toString()}` : ""}`,
  );
}

export function AdminJobSchedulesPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(
    useAuthStore((state) => state.context?.permissions ?? []),
  );
  const [filters, setFilterState] = useState(readFilters);
  const setFilters = (next: Filters) => {
    setFilterState(next);
    persistFilters(next);
  };
  const schedules = useJobSchedules(filters);
  const handlers = useJobHandlers();
  const mutations = useJobScheduleMutations();
  const [editing, setEditing] = useState<AdminJobSchedule | "new" | null>(
    null,
  );
  const [runSchedule, setRunSchedule] = useState<AdminJobSchedule | null>(null);
  const [runFilters, setRunFilters] = useState<RunFilters>({
    status: "",
    page: 1,
    page_size: 20,
  });
  const runs = useJobScheduleRuns(runSchedule?.id ?? null, runFilters);
  const [feedback, setFeedback] = useState<{
    key: string;
    error?: boolean;
  } | null>(null);
  const form = useForm<ScheduleValues>({ defaultValues: emptyScheduleValues });
  const cronExpression = form.watch("cron_expression");
  const selectedTimeZone = form.watch("time_zone");

  const formatDate = useMemo(
    () =>
      (value: string, timeZone = "UTC") =>
        new Intl.DateTimeFormat(i18n.language, {
          dateStyle: "medium",
          timeStyle: "long",
          timeZone,
        }).format(new Date(value)),
    [i18n.language],
  );

  const openEditor = (value: AdminJobSchedule | "new") => {
    setEditing(value);
    setFeedback(null);
    mutations.preview.reset();
    form.reset(value === "new" ? emptyScheduleValues : fromSchedule(value));
  };

  const applyValidationErrors = (values: ScheduleValues) => {
    const parsed = scheduleValuesSchema.safeParse(values);
    if (parsed.success) return true;
    for (const issue of parsed.error.issues) {
      const field = issue.path[0];
      if (typeof field === "string") {
        form.setError(field as keyof ScheduleValues, {
          message: t("schedules.validation.invalid"),
        });
      }
    }
    return false;
  };

  const previewCurrent = async () => {
    const values = form.getValues();
    if (!applyValidationErrors(values)) return;
    try {
      await mutations.preview.mutateAsync(toScheduleRequest(values));
    } catch {
      setFeedback({ key: "schedules.feedback.preview_error", error: true });
    }
  };

  const submit = form.handleSubmit(async (values) => {
    if (!applyValidationErrors(values)) return;
    try {
      const input = toScheduleRequest(values);
      if (editing === "new") {
        await mutations.create.mutateAsync(input);
      } else if (editing) {
        await mutations.update.mutateAsync({ id: editing.id, input });
      }
      setEditing(null);
      setFeedback({ key: "schedules.feedback.saved" });
    } catch {
      setFeedback({ key: "schedules.feedback.save_error", error: true });
    }
  });

  const confirmStatus = (schedule: AdminJobSchedule) => {
    const paused = schedule.status === "active";
    Modal.confirm({
      title: t(
        paused
          ? "schedules.status_change.pause_title"
          : "schedules.status_change.resume_title",
      ),
      content: t(
        paused
          ? "schedules.status_change.pause_confirm"
          : "schedules.status_change.resume_confirm",
        { name: schedule.name },
      ),
      okText: t(paused ? "schedules.actions.pause" : "schedules.actions.resume"),
      cancelText: t("common.actions.cancel"),
      onOk: async () => {
        try {
          await mutations.pause.mutateAsync({ id: schedule.id, paused });
          setFeedback({
            key: paused
              ? "schedules.feedback.paused"
              : "schedules.feedback.resumed",
          });
        } catch {
          setFeedback({ key: "schedules.feedback.action_error", error: true });
        }
      },
    });
  };

  const confirmExecute = (schedule: AdminJobSchedule) => {
    Modal.confirm({
      title: t("schedules.execute.title"),
      content: t("schedules.execute.confirm", {
        name: schedule.name,
        handler: schedule.handler_key,
      }),
      okText: t("schedules.actions.execute"),
      cancelText: t("common.actions.cancel"),
      onOk: async () => {
        try {
          await mutations.execute.mutateAsync({
            id: schedule.id,
            idempotencyKey: crypto.randomUUID(),
          });
          setRunSchedule(schedule);
          setRunFilters({ status: "", page: 1, page_size: 20 });
          setFeedback({ key: "schedules.feedback.executed" });
        } catch {
          setFeedback({ key: "schedules.feedback.action_error", error: true });
        }
      },
    });
  };

  const columns: TableColumnsType<AdminJobSchedule> = [
    {
      title: t("schedules.columns.name"),
      key: "name",
      ...(screens.md ? { fixed: "left" as const } : {}),
      width: 230,
      render: (_, item) => (
        <div>
          <strong>{item.name}</strong>
          <div className="ak-table-secondary">{item.code}</div>
        </div>
      ),
    },
    {
      title: t("schedules.columns.handler"),
      dataIndex: "handler_key",
      width: 230,
      render: (value: string) => <code>{value}</code>,
    },
    {
      title: t("schedules.columns.cron"),
      key: "cron",
      width: 210,
      render: (_, item) => (
        <div>
          <code>{item.cron_expression}</code>
          <div className="ak-table-secondary">{item.time_zone}</div>
        </div>
      ),
    },
    {
      title: t("schedules.columns.next_run"),
      dataIndex: "next_run_at",
      width: 210,
      render: (value: string | null | undefined, item) =>
        value ? formatDate(value, item.time_zone) : "—",
      responsive: ["lg"],
    },
    {
      title: t("schedules.columns.status"),
      dataIndex: "status",
      width: 120,
      render: (value: string) => (
        <Tag
          className={
            value === "active"
              ? "ak-status-success"
              : value === "disabled"
                ? "ak-status-error"
                : "ak-status-warning"
          }
        >
          {t(`schedules.status.${value}`)}
        </Tag>
      ),
    },
    {
      title: t("schedules.columns.actions"),
      key: "actions",
      ...(screens.md ? { fixed: "right" as const } : {}),
      width: screens.md ? 330 : 250,
      render: (_, item) => (
        <Space wrap>
          {permissions.has("jobs.schedule.update") ? (
            <Button size="small" onClick={() => { openEditor(item); }}>
              {t("common.actions.edit")}
            </Button>
          ) : null}
          {permissions.has("jobs.schedule.pause") &&
          item.status !== "disabled" ? (
            <Button
              size="small"
              loading={mutations.pause.isPending}
              onClick={() => { confirmStatus(item); }}
            >
              {t(
                item.status === "active"
                  ? "schedules.actions.pause"
                  : "schedules.actions.resume",
              )}
            </Button>
          ) : null}
          {permissions.has("jobs.schedule.execute") &&
          item.status !== "disabled" ? (
            <Button
              size="small"
              type="primary"
              loading={mutations.execute.isPending}
              onClick={() => { confirmExecute(item); }}
            >
              {t("schedules.actions.execute")}
            </Button>
          ) : null}
          {permissions.has("jobs.run.read") ? (
            <Button
              size="small"
              onClick={() => {
                setRunSchedule(item);
                setRunFilters({ status: "", page: 1, page_size: 20 });
              }}
            >
              {t("schedules.actions.runs")}
            </Button>
          ) : null}
        </Space>
      ),
    },
  ];

  const runColumns: TableColumnsType<AdminJobRun> = [
    {
      title: t("schedules.runs.columns.status"),
      dataIndex: "status",
      width: 100,
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
      width: 100,
      render: (value: string) => t(`schedules.trigger.${value}`),
    },
    {
      title: t("schedules.runs.columns.scheduled"),
      dataIndex: "scheduled_at",
      width: 170,
      render: (value: string) =>
        formatDate(value, runSchedule?.time_zone ?? "UTC"),
    },
    {
      title: t("schedules.runs.columns.attempt"),
      dataIndex: "attempt",
      width: 80,
    },
    {
      title: t("schedules.runs.columns.result"),
      key: "result",
      width: 220,
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

  const preview = mutations.preview.data;
  return (
    <div className="ak-page-container">
      <header className="ak-page-heading ak-schedules-heading">
        <div>
          <Typography.Title level={1}>{t("schedules.title")}</Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("schedules.description")}
          </Typography.Paragraph>
        </div>
        {permissions.has("jobs.schedule.create") ? (
          <Button type="primary" onClick={() => { openEditor("new"); }}>
            {t("schedules.actions.create")}
          </Button>
        ) : null}
      </header>

      {feedback ? (
        <div
          className={feedback.error ? "ak-form-error" : "ak-org-feedback"}
          role={feedback.error ? "alert" : "status"}
        >
          {t(feedback.key)}
        </div>
      ) : null}

      <Alert
        className="ak-schedule-safety"
        type="info"
        showIcon
        title={t("schedules.safety.title")}
        description={t("schedules.safety.description")}
      />

      <Card>
        <div
          className="ak-settings-filters"
          role="search"
          aria-label={t("schedules.filters.landmark")}
        >
          <Input.Search
            allowClear
            aria-label={t("schedules.filters.query")}
            placeholder={t("schedules.filters.query")}
            value={filters.q}
            onChange={(event) =>
              { setFilters({ ...filters, q: event.target.value, page: 1 }); }
            }
          />
          <Select
            allowClear
            aria-label={t("schedules.filters.status")}
            placeholder={t("schedules.filters.status")}
            value={filters.status || undefined}
            onChange={(value) =>
              { setFilters({ ...filters, status: value ?? "", page: 1 }); }
            }
            options={["active", "paused", "disabled"].map((value) => ({
              value,
              label: t(`schedules.status.${value}`),
            }))}
          />
          <Select
            allowClear
            showSearch
            aria-label={t("schedules.filters.time_zone")}
            placeholder={t("schedules.filters.time_zone")}
            value={filters.time_zone || undefined}
            onChange={(value) =>
              { setFilters({ ...filters, time_zone: value ?? "", page: 1 }); }
            }
            options={timeZones.map((value) => ({ value, label: value }))}
          />
        </div>
        {schedules.isError || handlers.isError ? (
          <Alert
            type="error"
            showIcon
            title={t("schedules.feedback.load_error")}
            action={
              <Button
                onClick={() => {
                  void schedules.refetch();
                  void handlers.refetch();
                }}
              >
                {t("common.actions.retry")}
              </Button>
            }
          />
        ) : null}
        <div
          className="ak-table-scroll"
          tabIndex={0}
          aria-label={t("schedules.table.landmark")}
        >
          <Table
            rowKey="id"
            columns={columns}
            dataSource={schedules.data?.items ?? []}
            loading={schedules.isPending || handlers.isPending}
            locale={{ emptyText: t("schedules.empty") }}
            {...((schedules.data?.items.length ?? 0) > 0
              ? { scroll: { x: 1320 } }
              : {})}
            pagination={{
              current: filters.page,
              pageSize: filters.page_size,
              total: schedules.data?.total ?? 0,
              showSizeChanger: true,
              onChange: (page, page_size) =>
                { setFilters({ ...filters, page, page_size }); },
            }}
          />
        </div>
      </Card>

      <Drawer
        rootClassName="ak-schedule-drawer"
        destroyOnHidden
        size={screens.md ? "large" : "100%"}
        open={editing !== null}
        title={t(
          editing === "new"
            ? "schedules.editor.create_title"
            : "schedules.editor.edit_title",
        )}
        onClose={() => { setEditing(null); }}
        extra={
          <Button
            type="primary"
            loading={mutations.create.isPending || mutations.update.isPending}
            onClick={() => void submit()}
          >
            {t("common.actions.save")}
          </Button>
        }
      >
        <Form layout="vertical" onFinish={() => void submit()}>
          <div className="ak-schedule-form-grid">
            <Controller
              name="code"
              control={form.control}
              render={({ field, fieldState }) => (
                <Form.Item
                  htmlFor="schedule-code"
                  label={t("schedules.editor.code")}
                  {...(fieldState.error
                    ? {
                        help: fieldState.error.message,
                        validateStatus: "error" as const,
                      }
                    : {})}
                >
                  <Input {...field} id="schedule-code" autoComplete="off" />
                </Form.Item>
              )}
            />
            <Controller
              name="name"
              control={form.control}
              render={({ field, fieldState }) => (
                <Form.Item
                  htmlFor="schedule-name"
                  label={t("schedules.editor.name")}
                  {...(fieldState.error
                    ? {
                        help: fieldState.error.message,
                        validateStatus: "error" as const,
                      }
                    : {})}
                >
                  <Input {...field} id="schedule-name" autoComplete="off" />
                </Form.Item>
              )}
            />
          </div>
          <Controller
            name="handler_key"
            control={form.control}
            render={({ field, fieldState }) => (
              <Form.Item
                htmlFor="schedule-handler"
                label={t("schedules.editor.handler")}
                extra={t("schedules.editor.handler_help")}
                {...(fieldState.error
                  ? {
                      help: fieldState.error.message,
                      validateStatus: "error" as const,
                    }
                  : {})}
              >
                <Select
                  {...field}
                  id="schedule-handler"
                  options={(handlers.data ?? []).map((item) => ({
                    value: item.key,
                    label: `${t(item.name_key)} · ${item.key}`,
                  }))}
                />
              </Form.Item>
            )}
          />
          <div className="ak-schedule-form-grid">
            <Controller
              name="cron_expression"
              control={form.control}
              render={({ field, fieldState }) => (
                <Form.Item
                  htmlFor="schedule-cron"
                  label={t("schedules.editor.cron")}
                  extra={t(cronDescriptionKey(cronExpression), {
                    expression: cronExpression,
                  })}
                  {...(fieldState.error
                    ? {
                        help: fieldState.error.message,
                        validateStatus: "error" as const,
                      }
                    : {})}
                >
                  <Input
                    {...field}
                    id="schedule-cron"
                    autoComplete="off"
                    onBlur={() => {
                      field.onBlur();
                      void previewCurrent();
                    }}
                  />
                </Form.Item>
              )}
            />
            <Controller
              name="time_zone"
              control={form.control}
              render={({ field, fieldState }) => (
                <Form.Item
                  htmlFor="schedule-time-zone"
                  label={t("schedules.editor.time_zone")}
                  extra={t("schedules.editor.time_zone_help")}
                  {...(fieldState.error
                    ? {
                        help: fieldState.error.message,
                        validateStatus: "error" as const,
                      }
                    : {})}
                >
                  <Select
                    {...field}
                    id="schedule-time-zone"
                    showSearch
                    options={timeZones.map((value) => ({
                      value,
                      label: value,
                    }))}
                    onBlur={() => {
                      field.onBlur();
                      void previewCurrent();
                    }}
                  />
                </Form.Item>
              )}
            />
          </div>
          <Card
            size="small"
            className="ak-schedule-preview"
            title={t("schedules.preview.title")}
            extra={
              <Button
                size="small"
                loading={mutations.preview.isPending}
                onClick={() => void previewCurrent()}
              >
                {t("schedules.preview.action")}
              </Button>
            }
          >
            {mutations.preview.isError ? (
              <Alert
                type="error"
                showIcon
                title={t("schedules.feedback.preview_error")}
              />
            ) : null}
            {preview ? (
              <List
                aria-live="polite"
                size="small"
                dataSource={preview.next_runs}
                renderItem={(value, index) => (
                  <List.Item>
                    <span>{t("schedules.preview.item", { count: index + 1 })}</span>
                    <time dateTime={value}>
                      {formatDate(value, selectedTimeZone)}
                    </time>
                  </List.Item>
                )}
              />
            ) : (
              <Typography.Text type="secondary">
                {t("schedules.preview.empty")}
              </Typography.Text>
            )}
          </Card>
          <div className="ak-schedule-form-grid">
            <Controller
              name="overlap_policy"
              control={form.control}
              render={({ field }) => (
                <Form.Item
                  htmlFor="schedule-overlap"
                  label={t("schedules.editor.overlap")}
                >
                  <Select
                    {...field}
                    id="schedule-overlap"
                    options={["allow", "skip", "replace"].map((value) => ({
                      value,
                      label: t(`schedules.overlap.${value}`),
                    }))}
                  />
                </Form.Item>
              )}
            />
            <Controller
              name="misfire_policy"
              control={form.control}
              render={({ field }) => (
                <Form.Item
                  htmlFor="schedule-misfire"
                  label={t("schedules.editor.misfire")}
                >
                  <Select
                    {...field}
                    id="schedule-misfire"
                    options={["ignore", "fire_once", "catch_up"].map(
                      (value) => ({
                        value,
                        label: t(`schedules.misfire.${value}`),
                      }),
                    )}
                  />
                </Form.Item>
              )}
            />
            <Controller
              name="timeout_seconds"
              control={form.control}
              render={({ field, fieldState }) => (
                <Form.Item
                  htmlFor="schedule-timeout"
                  label={t("schedules.editor.timeout")}
                  {...(fieldState.error
                    ? {
                        help: fieldState.error.message,
                        validateStatus: "error" as const,
                      }
                    : {})}
                >
                  <InputNumber
                    id="schedule-timeout"
                    min={1}
                    max={86400}
                    value={field.value}
                    onBlur={field.onBlur}
                    onChange={(value) => { field.onChange(value ?? 0); }}
                  />
                </Form.Item>
              )}
            />
            <Controller
              name="max_attempts"
              control={form.control}
              render={({ field, fieldState }) => (
                <Form.Item
                  htmlFor="schedule-attempts"
                  label={t("schedules.editor.max_attempts")}
                  {...(fieldState.error
                    ? {
                        help: fieldState.error.message,
                        validateStatus: "error" as const,
                      }
                    : {})}
                >
                  <InputNumber
                    id="schedule-attempts"
                    min={1}
                    max={100}
                    value={field.value}
                    onBlur={field.onBlur}
                    onChange={(value) => { field.onChange(value ?? 0); }}
                  />
                </Form.Item>
              )}
            />
          </div>
          <Controller
            name="payload"
            control={form.control}
            render={({ field, fieldState }) => (
              <Form.Item
                htmlFor="schedule-payload"
                label={t("schedules.editor.payload")}
                extra={t("schedules.editor.payload_help")}
                {...(fieldState.error
                  ? {
                      help: fieldState.error.message,
                      validateStatus: "error" as const,
                    }
                  : {})}
              >
                <Input.TextArea
                  {...field}
                  id="schedule-payload"
                  rows={6}
                  spellCheck={false}
                />
              </Form.Item>
            )}
          />
        </Form>
      </Drawer>

      <Drawer
        rootClassName="ak-schedule-drawer"
        destroyOnHidden
        size={screens.md ? "large" : "100%"}
        open={runSchedule !== null}
        title={t("schedules.runs.title", { name: runSchedule?.name ?? "" })}
        onClose={() => { setRunSchedule(null); }}
        extra={
          runSchedule ? (
            <Button
              onClick={() =>
                void navigate({
                  to: "/system/integrations/schedules/$scheduleId/runs",
                  params: { scheduleId: runSchedule.id },
                })
              }
            >
              {t("common.actions.view")}
            </Button>
          ) : null
        }
      >
        {runSchedule ? (
          <Descriptions
            className="ak-schedule-run-summary"
            bordered
            column={{ xs: 1, sm: 2 }}
            items={[
              {
                key: "handler",
                label: t("schedules.columns.handler"),
                children: <code>{runSchedule.handler_key}</code>,
              },
              {
                key: "zone",
                label: t("schedules.editor.time_zone"),
                children: runSchedule.time_zone,
              },
            ]}
          />
        ) : null}
        <Select
          allowClear
          className="ak-schedule-run-filter"
          aria-label={t("schedules.runs.filters.status")}
          placeholder={t("schedules.runs.filters.status")}
          value={runFilters.status || undefined}
          onChange={(value) =>
            { setRunFilters({ ...runFilters, status: value ?? "", page: 1 }); }
          }
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
        {runs.isError ? (
          <Alert
            type="error"
            showIcon
            title={t("schedules.feedback.runs_error")}
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
          aria-label={t("schedules.runs.table_landmark")}
        >
          <Table
            rowKey="id"
            columns={runColumns}
            dataSource={runs.data?.items ?? []}
            loading={runs.isPending}
            locale={{ emptyText: t("schedules.runs.empty") }}
            pagination={{
              current: runFilters.page,
              pageSize: runFilters.page_size,
              total: runs.data?.total ?? 0,
              showSizeChanger: true,
              onChange: (page, page_size) =>
                { setRunFilters({ ...runFilters, page, page_size }); },
            }}
          />
        </div>
      </Drawer>
    </div>
  );
}
