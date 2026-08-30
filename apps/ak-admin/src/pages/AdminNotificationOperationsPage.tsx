import { ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Checkbox, Col, Descriptions, Drawer, Empty, Input, Modal, Progress, Row, Segmented, Select, Skeleton, Space, Statistic, Table, Tabs, Tag, Typography, message, type TableColumnsType } from "antd";
import type { TFunction } from "i18next";
import { lazy, Suspense, useMemo, useState, type Key } from "react";
import { useTranslation } from "react-i18next";
import { AppSelectionRequiredState } from "../components/AppSelectionRequiredState";
import type { AdminNotificationFailure, AdminNotificationOperationsSummary, AdminNotificationRun, AdminNotificationTask, AdminNotificationTrendBucket, PushWritableProvider } from "../generated/api/types.gen";
import { useAppScope } from "../features/apps/scope";
import { useAuthStore } from "../features/auth/store";
import { useNotificationOperations, useNotificationRetry, useNotificationRun, useNotificationTask } from "../features/notification-operations/hooks";
import { persistOperationsFilters, readOperationsFilters, type OperationsFilters, type OperationsTab } from "../features/notification-operations/model";

const TrendChart = lazy(async () => import("../components/NotificationOperationsTrendChart").then((module) => ({ default: module.NotificationOperationsTrendChart })));
const providers: PushWritableProvider[] = ["apns", "fcm", "huawei_android", "honor", "xiaomi", "oppo", "vivo", "meizu", "harmony"];
const taskKinds = ["appkernia-message-publish", "appkernia-push-fanout", "appkernia-notification-delivery"];

export function AdminNotificationOperationsPage() {
  const { t, i18n } = useTranslation();
  const scope = useAppScope();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [filters, setFilterState] = useState(() => readOperationsFilters(location.search));
  const [selectedTask, setSelectedTask] = useState<string | null>(null);
  const [selectedRun, setSelectedRun] = useState<string | null>(null);
  const [selectedFailures, setSelectedFailures] = useState<string[]>([]);
  const [liveMessage, setLiveMessage] = useState("");
  const [messageApi, holder] = message.useMessage();
  const queries = useNotificationOperations(scope.appId, filters);
  const retry = useNotificationRetry(scope.appId, queries.root);
  const taskDetail = useNotificationTask(scope.appId, selectedTask);
  const runDetail = useNotificationRun(scope.appId, selectedRun);
  const formatter = useMemo(() => new Intl.DateTimeFormat(i18n.resolvedLanguage === "en-US" ? "en-US" : "zh-CN", { dateStyle: "medium", timeStyle: "short" }), [i18n.resolvedLanguage]);

  const setFilters = (patch: Partial<OperationsFilters>, resetPage = true) => {
    const next = { ...filters, ...patch, ...(resetPage ? { page: 1 } : {}) };
    setFilterState(next);
    persistOperationsFilters(next, scope.appId);
    setSelectedFailures([]);
  };
  const refresh = async () => {
    await queries.summary.refetch();
    if (filters.tab === "overview") await queries.trends.refetch();
    if (filters.tab === "runs") await queries.runs.refetch();
    if (filters.tab === "tasks") await queries.tasks.refetch();
    if (filters.tab === "failures") await queries.failures.refetch();
    setLiveMessage(t("notification_operations.feedback.refreshed"));
  };
  const retryTasks = (tasks: AdminNotificationTask[]) => {
    if (tasks.length === 0) return;
    const duplicateRisk = tasks.some((task) => task.retry_risk === "duplicate_possible");
    if (duplicateRisk && tasks.length !== 1) {
      messageApi.error(t("notification_operations.retry.batch_risk_rejected"));
      return;
    }
    let acknowledged = !duplicateRisk;
    Modal.confirm({
      title: t("notification_operations.retry.title"),
      content: <Space orientation="vertical"><Typography.Text>{t("notification_operations.retry.confirm", { count: tasks.length })}</Typography.Text>{duplicateRisk ? <Alert showIcon type="warning" title={t("notification_operations.retry.duplicate_warning")} description={<Checkbox onChange={(event) => { acknowledged = event.target.checked; }}>{t("notification_operations.retry.acknowledge")}</Checkbox>} /> : null}</Space>,
      okText: t("notification_operations.retry.action"), cancelText: t("common.actions.cancel"),
      onOk: async () => {
        if (!acknowledged) throw new Error("duplicate risk acknowledgement is required");
        const results = await retry.mutateAsync({ taskIds: tasks.map((task) => task.id), acknowledge: acknowledged });
        const accepted = results.filter((item) => item.accepted).length;
        messageApi.success(t("notification_operations.retry.result", { accepted, total: results.length }));
        setSelectedFailures([]);
      },
    });
  };

  const activeData = filters.tab === "runs" ? queries.runs : filters.tab === "tasks" ? queries.tasks : filters.tab === "failures" ? queries.failures : queries.trends;
  const tabs = (["overview", "runs", "tasks", "failures"] as OperationsTab[]).map((key) => ({ key, label: t(`notification_operations.tabs.${key}`), children: key === "overview" ? <Overview summary={queries.summary.data} trends={queries.trends.data ?? []} loading={queries.summary.isPending || queries.trends.isPending} t={t} /> : key === "runs" ? <RunsTable data={queries.runs.data?.items ?? []} total={queries.runs.data?.total ?? 0} loading={queries.runs.isPending} filters={filters} formatter={formatter} onPage={(page, pageSize) => { setFilters({ page, page_size: pageSize }, false); }} onView={setSelectedRun} t={t} /> : key === "tasks" ? <TasksTable data={queries.tasks.data?.items ?? []} total={queries.tasks.data?.total ?? 0} loading={queries.tasks.isPending} filters={filters} formatter={formatter} canRetry={permissions.has("notify.task.retry")} onPage={(page, pageSize) => { setFilters({ page, page_size: pageSize }, false); }} onView={setSelectedTask} onRetry={(task) => { retryTasks([task]); }} t={t} /> : <FailuresTable data={queries.failures.data?.items ?? []} total={queries.failures.data?.total ?? 0} loading={queries.failures.isPending} filters={filters} formatter={formatter} selected={selectedFailures} canRetry={permissions.has("notify.task.retry")} onSelected={setSelectedFailures} onPage={(page, pageSize) => { setFilters({ page, page_size: pageSize }, false); }} onView={setSelectedTask} onRetry={(task) => { retryTasks([task]); }} t={t} /> }));
  const selectedFailureItems = (queries.failures.data?.items ?? []).filter((item) => selectedFailures.includes(item.id));

  return <div className="ak-page-container ak-notification-operations-page">
    {holder}<div className="ak-sr-only" aria-live="polite">{liveMessage}</div>
    <header className="ak-page-heading"><div><Typography.Title level={1}>{t("notification_operations.title")}</Typography.Title><Typography.Paragraph type="secondary">{t("notification_operations.description")}</Typography.Paragraph></div><Button icon={<ReloadOutlined />} loading={activeData.isFetching || queries.summary.isFetching} onClick={() => void refresh()}>{t("common.actions.refresh")}</Button></header>
    {!scope.appId ? <AppSelectionRequiredState /> : <Space orientation="vertical" size="large" style={{ width: "100%" }}>
      <Card className="ak-notification-operations-filters"><div role="search" aria-label={t("notification_operations.filters.landmark")} className="ak-operations-filter-grid">
        <div><Typography.Text strong>{t("notification_operations.filters.range")}</Typography.Text><Segmented block value={filters.range} options={(["7d", "30d", "90d"] as const).map((value) => ({ value, label: t(`notification_operations.range.${value}`) }))} onChange={(value) => { setFilters({ range: value }); }} /></div>
        <Select allowClear aria-label={t("notification_operations.filters.environment")} placeholder={t("notification_operations.filters.environment")} value={filters.environment || undefined} options={(["development", "test", "staging", "production"] as const).map((value) => ({ value, label: t(`push_channels.environment.${value}`) }))} onChange={(value) => { setFilters({ environment: value ?? "" }); }} />
        <Select allowClear aria-label={t("notification_operations.filters.category")} placeholder={t("notification_operations.filters.category")} value={filters.category || undefined} options={(["service_security", "news_operations"] as const).map((value) => ({ value, label: t(`notification_operations.category.${value}`) }))} onChange={(value) => { setFilters({ category: value ?? "" }); }} />
        <Select allowClear aria-label={t("notification_operations.filters.channel")} placeholder={t("notification_operations.filters.channel")} value={filters.channel || undefined} options={(["in_app", "email", "sms", "push", "webhook"] as const).map((value) => ({ value, label: t(`notifications.channel.${value}`) }))} onChange={(value) => { setFilters({ channel: value ?? "" }); }} />
        <Select allowClear aria-label={t("notification_operations.filters.provider")} placeholder={t("notification_operations.filters.provider")} value={filters.provider || undefined} options={providers.map((value) => ({ value, label: t(`push_channels.provider.${value}`) }))} onChange={(value) => { setFilters({ provider: value ?? "" }); }} />
        {filters.tab === "tasks" || filters.tab === "failures" ? <Select allowClear aria-label={t("notification_operations.filters.task_kind")} placeholder={t("notification_operations.filters.task_kind")} value={filters.task_kind || undefined} options={taskKinds.map((value) => ({ value, label: t(`notification_operations.task_kind.${value}`) }))} onChange={(value) => { setFilters({ task_kind: value ?? "" }); }} /> : null}
        {filters.tab !== "overview" ? <Select allowClear aria-label={t("notification_operations.filters.status")} placeholder={t("notification_operations.filters.status")} value={filters.status || undefined} options={(filters.tab === "runs" ? ["scheduled", "queued", "running", "completed", "completed_with_failures", "failed", "cancelled", "expired"] : ["scheduled", "queued", "running", "retry_wait", "succeeded", "failed", "cancelled"]).map((value) => ({ value, label: t(`notification_operations.status.${value}`) }))} onChange={(value) => { setFilters({ status: value ?? "" }); }} /> : null}
        {filters.tab !== "overview" ? <Input.Search allowClear aria-label={t("notification_operations.filters.query")} placeholder={t("notification_operations.filters.query")} value={filters.q} onChange={(event) => { setFilters({ q: event.target.value }); }} /> : null}
      </div></Card>
      {queries.summary.isError || activeData.isError ? <Alert showIcon role="alert" type="error" title={t("notification_operations.feedback.load_error")} action={<Button onClick={() => void refresh()}>{t("common.actions.retry")}</Button>} /> : null}
      {queries.summary.data?.has_unfinished_work ? <Alert showIcon type="info" title={t("notification_operations.polling.active")} description={t("notification_operations.polling.description")} /> : null}
      <Tabs activeKey={filters.tab} items={tabs} onChange={(value) => { setFilters({ tab: value as OperationsTab, status: "", q: "" }); }} />
      {filters.tab === "failures" && selectedFailures.length > 0 && permissions.has("notify.task.retry") ? <div className="ak-operations-batch-bar"><Typography.Text>{t("notification_operations.failures.selected", { count: selectedFailures.length })}</Typography.Text><Button type="primary" loading={retry.isPending} onClick={() => { retryTasks(selectedFailureItems); }}>{t("notification_operations.retry.batch")}</Button></div> : null}
    </Space>}
    <TaskDrawer open={selectedTask !== null} task={taskDetail.data} loading={taskDetail.isPending} formatter={formatter} onClose={() => { setSelectedTask(null); }} t={t} />
    <RunDrawer open={selectedRun !== null} run={runDetail.data} loading={runDetail.isPending} formatter={formatter} onClose={() => { setSelectedRun(null); }} t={t} />
  </div>;
}

type Translator = TFunction;

function statusTag(status: string, t: Translator) {
  const className = ["completed", "succeeded"].includes(status) ? "ak-status-success" : ["failed", "expired"].includes(status) ? "ak-status-error" : ["running", "retry_wait", "completed_with_failures"].includes(status) ? "ak-status-warning" : "ak-status-neutral";
  return <Tag className={className}>{t(`notification_operations.status.${status}`)}</Tag>;
}

function localizedResult(resultClass: string | undefined, fallback: string | undefined, t: Translator) {
  if (!resultClass) return fallback ?? "—";
  const key = `notification_operations.result_class.${resultClass}`;
  const translated = t(key);
  return translated === key ? fallback ?? resultClass : translated;
}

function Overview({ summary, trends, loading, t }: { summary: AdminNotificationOperationsSummary | undefined; trends: AdminNotificationTrendBucket[]; loading: boolean; t: Translator }) {
  if (loading) return <Skeleton active paragraph={{ rows: 12 }} />;
  if (!summary) return <Empty description={t("common.states.empty")} />;
  const metrics = ["accepted", "failed", "invalid_tokens", "skipped", "opened", "queued", "retry_waiting", "open_rate"] as const;
  const labels = Object.fromEntries(["accepted", "failed", "invalid_tokens", "opened", "skipped"].map((key) => [key, t(`notification_operations.metrics.${key}`)]));
  const columns: TableColumnsType<AdminNotificationTrendBucket> = [{ title: t("notification_operations.trends.day"), dataIndex: "bucket", render: (value: string) => value.slice(0, 10) }, ...(["accepted", "failed", "invalid_tokens", "opened", "skipped"] as const).map((key) => ({ title: labels[key], dataIndex: key }))];
  return <Space orientation="vertical" size="large" style={{ width: "100%" }}><Row gutter={[16, 16]}>{metrics.map((key) => <Col key={key} xs={12} md={8} xl={6}><Card className="ak-kpi-card"><Statistic title={t(`notification_operations.metrics.${key}`)} value={key === "open_rate" ? Number((summary[key] * 100).toFixed(1)) : summary[key]} suffix={key === "open_rate" ? "%" : undefined} /></Card></Col>)}</Row><Row gutter={[16, 16]}><Col xs={24} md={8}><Card title={t("notification_operations.queue.oldest")}><Statistic value={summary.oldest_waiting_seconds} suffix="s" /></Card></Col><Col xs={24} md={8}><Card title={t("notification_operations.queue.p95")}><Statistic value={summary.p95_queue_delay_ms} suffix="ms" /></Card></Col><Col xs={24} md={8}><Card title={t("notification_operations.queue.running")}><Statistic value={summary.running} /></Card></Col></Row><Card title={t("notification_operations.trends.title")}>{trends.length ? <><Suspense fallback={<Skeleton active paragraph={{ rows: 8 }} />}><TrendChart ariaLabel={t("notification_operations.trends.chart_label")} items={trends} labels={labels} /></Suspense><details className="ak-dashboard-table-alternative"><summary>{t("notification_operations.trends.table_toggle")}</summary><Table rowKey="bucket" size="small" pagination={false} scroll={{ x: 680 }} columns={columns} dataSource={trends} /></details></> : <Empty description={t("notification_operations.trends.empty")} />}</Card></Space>;
}

function RunsTable({ data, total, loading, filters, formatter, onPage, onView, t }: { data: AdminNotificationRun[]; total: number; loading: boolean; filters: OperationsFilters; formatter: Intl.DateTimeFormat; onPage: (page: number, size: number) => void; onView: (id: string) => void; t: Translator }) {
  const columns: TableColumnsType<AdminNotificationRun> = [
    { title: t("notification_operations.runs.message"), dataIndex: "message_title", render: (value: string) => <Typography.Text strong>{value}</Typography.Text> },
    { title: t("notification_operations.runs.status"), dataIndex: "status", render: (value: string) => statusTag(value, t) },
    { title: t("notification_operations.runs.progress"), render: (_, item) => <Progress size="small" percent={item.recipient_count > 0 ? Math.min(100, Math.round(item.evaluated_count / item.recipient_count * 100)) : 0} /> },
    { title: t("notification_operations.metrics.accepted"), dataIndex: "accepted_count", responsive: ["md"] },
    { title: t("notification_operations.metrics.failed"), dataIndex: "failed_count", responsive: ["md"] },
    { title: t("notification_operations.runs.created"), dataIndex: "created_at", responsive: ["lg"], render: (value: string) => formatter.format(new Date(value)) },
    { title: t("notifications.columns.actions"), render: (_, item) => <Button size="small" onClick={() => { onView(item.id); }}>{t("common.actions.view")}</Button> },
  ];
  return <div className="ak-table-scroll" role="region" tabIndex={0} aria-label={t("notification_operations.tabs.runs")}><Table rowKey="id" loading={loading} columns={columns} dataSource={data} scroll={{ x: 900 }} pagination={{ current: filters.page, pageSize: filters.page_size, total, showSizeChanger: true, onChange: onPage }} /></div>;
}

function TasksTable({ data, total, loading, filters, formatter, canRetry, onPage, onView, onRetry, t }: { data: AdminNotificationTask[]; total: number; loading: boolean; filters: OperationsFilters; formatter: Intl.DateTimeFormat; canRetry: boolean; onPage: (page: number, size: number) => void; onView: (id: string) => void; onRetry: (item: AdminNotificationTask) => void; t: Translator }) {
  const columns: TableColumnsType<AdminNotificationTask> = [
    { title: t("notification_operations.tasks.kind"), dataIndex: "task_kind", render: (value: string) => t(`notification_operations.task_kind.${value}`) },
    { title: t("notification_operations.tasks.status"), dataIndex: "status", render: (value: string) => statusTag(value, t) },
    { title: t("notification_operations.tasks.attempts"), render: (_, item) => `${String(item.attempt_count)} / ${String(item.max_attempts)}` },
    { title: t("notification_operations.tasks.next"), dataIndex: "next_retry_at", responsive: ["md"], render: (value?: string | null) => value ? formatter.format(new Date(value)) : "—" },
    { title: t("notification_operations.tasks.error"), dataIndex: "last_error_code", ellipsis: true, responsive: ["lg"], render: (value?: string) => value ?? "—" },
    { title: t("notifications.columns.actions"), width: 180, render: (_, item) => <Space><Button size="small" onClick={() => { onView(item.id); }}>{t("common.actions.view")}</Button>{canRetry && item.retryable ? <Button size="small" type="primary" onClick={() => { onRetry(item); }}>{t("notification_operations.retry.action")}</Button> : null}</Space> },
  ];
  return <div className="ak-table-scroll" role="region" tabIndex={0} aria-label={t("notification_operations.tabs.tasks")}><Table rowKey="id" loading={loading} columns={columns} dataSource={data} scroll={{ x: 920 }} pagination={{ current: filters.page, pageSize: filters.page_size, total, showSizeChanger: true, onChange: onPage }} /></div>;
}

function FailuresTable({ data, total, loading, filters, formatter, selected, canRetry, onSelected, onPage, onView, onRetry, t }: { data: AdminNotificationFailure[]; total: number; loading: boolean; filters: OperationsFilters; formatter: Intl.DateTimeFormat; selected: string[]; canRetry: boolean; onSelected: (ids: string[]) => void; onPage: (page: number, size: number) => void; onView: (id: string) => void; onRetry: (item: AdminNotificationFailure) => void; t: Translator }) {
  const columns: TableColumnsType<AdminNotificationFailure> = [
    { title: t("notification_operations.failures.resource"), render: (_, item) => item.message_title ?? t(`notification_operations.task_kind.${item.task_kind}`) },
    { title: t("notification_operations.filters.provider"), dataIndex: "provider", render: (value?: string) => value ?? "—", responsive: ["md"] },
    { title: t("notification_operations.tasks.error"), render: (_, item) => <Space orientation="vertical" size={0}><Typography.Text code>{item.last_error_code ?? "—"}</Typography.Text><Typography.Text type="secondary" ellipsis>{localizedResult(item.last_result_class, item.last_error_summary, t)}</Typography.Text></Space> },
    { title: t("notification_operations.failures.time"), dataIndex: "finalized_at", responsive: ["lg"], render: (value?: string | null) => value ? formatter.format(new Date(value)) : "—" },
    { title: t("notifications.columns.actions"), width: 180, render: (_, item) => <Space><Button size="small" onClick={() => { onView(item.id); }}>{t("common.actions.view")}</Button>{canRetry && item.retryable ? <Button size="small" type="primary" onClick={() => { onRetry(item); }}>{t("notification_operations.retry.action")}</Button> : null}</Space> },
  ];
  return <div className="ak-table-scroll" role="region" tabIndex={0} aria-label={t("notification_operations.tabs.failures")}><Table rowKey="id" loading={loading} {...(canRetry ? { rowSelection: { selectedRowKeys: selected, getCheckboxProps: (item: AdminNotificationFailure) => ({ disabled: !item.retryable }), onChange: (keys: Key[]) => { onSelected(keys.map(String)); } } } : {})} columns={columns} dataSource={data} scroll={{ x: 920 }} pagination={{ current: filters.page, pageSize: filters.page_size, total, showSizeChanger: true, onChange: onPage }} /></div>;
}

function TaskDrawer({ open, task, loading, formatter, onClose, t }: { open: boolean; task: AdminNotificationTask | undefined; loading: boolean; formatter: Intl.DateTimeFormat; onClose: () => void; t: Translator }) {
  return <Drawer open={open} onClose={onClose} size={720} title={t("notification_operations.task_detail.title")}><Skeleton loading={loading} active>{task ? <Space orientation="vertical" size="large" style={{ width: "100%" }}><Descriptions bordered column={1} size="small" items={[{ key: "status", label: t("notification_operations.tasks.status"), children: statusTag(task.status, t) }, { key: "kind", label: t("notification_operations.tasks.kind"), children: t(`notification_operations.task_kind.${task.task_kind}`) }, { key: "attempts", label: t("notification_operations.tasks.attempts"), children: `${String(task.attempt_count)} / ${String(task.max_attempts)}` }, { key: "resource", label: t("notification_operations.task_detail.resource"), children: task.resource_id ?? "—" }, { key: "trace", label: t("notification_operations.task_detail.trace"), children: task.trace_id ?? "—" }, { key: "error", label: t("notification_operations.tasks.error"), children: localizedResult(task.last_result_class, task.last_error_summary, t) }]} /><Typography.Title level={3}>{t("notification_operations.task_detail.attempts")}</Typography.Title>{task.attempts?.length ? <Table rowKey="attempt_number" pagination={false} size="small" scroll={{ x: 700 }} dataSource={task.attempts} columns={[{ title: "#", dataIndex: "attempt_number" }, { title: t("notification_operations.tasks.status"), dataIndex: "status", render: (value: string) => statusTag(value, t) }, { title: t("notification_operations.task_detail.duration"), dataIndex: "duration_ms", render: (value?: number | null) => value === null || value === undefined ? "—" : `${String(value)} ms` }, { title: t("notification_operations.tasks.error"), render: (_, attempt) => localizedResult(attempt.result_class, attempt.error_summary, t), ellipsis: true }, { title: t("notification_operations.failures.time"), dataIndex: "started_at", render: (value: string) => formatter.format(new Date(value)) }]} /> : <Empty description={t("common.states.empty")} />}</Space> : null}</Skeleton></Drawer>;
}

function RunDrawer({ open, run, loading, formatter, onClose, t }: { open: boolean; run: AdminNotificationRun | undefined; loading: boolean; formatter: Intl.DateTimeFormat; onClose: () => void; t: Translator }) {
  return <Drawer open={open} onClose={onClose} size={640} title={t("notification_operations.run_detail.title")}><Skeleton loading={loading} active>{run ? <Descriptions bordered column={1} size="small" items={[{ key: "message", label: t("notification_operations.runs.message"), children: run.message_title }, { key: "status", label: t("notification_operations.runs.status"), children: statusTag(run.status, t) }, { key: "recipient", label: t("notification_operations.run_detail.recipients"), children: run.recipient_count }, { key: "evaluated", label: t("notification_operations.run_detail.evaluated"), children: run.evaluated_count }, { key: "delivery", label: t("notification_operations.run_detail.deliveries"), children: run.delivery_count }, { key: "accepted", label: t("notification_operations.metrics.accepted"), children: run.accepted_count }, { key: "failed", label: t("notification_operations.metrics.failed"), children: run.failed_count }, { key: "skipped", label: t("notification_operations.metrics.skipped"), children: run.skipped_count }, { key: "opened", label: t("notification_operations.metrics.opened"), children: run.opened_count }, { key: "time", label: t("notification_operations.runs.created"), children: formatter.format(new Date(run.created_at)) }]} /> : null}</Skeleton></Drawer>;
}
