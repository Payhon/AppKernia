import { Alert, Button, Card, DatePicker, Descriptions, Drawer, Form, Image, Input, Select, Space, Spin, Table, Tag, Typography } from "antd";
import { useEffect, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useAuthStore } from "../features/auth/store";
import { useAppScope } from "../features/apps/scope";
import { useTenantKey } from "../features/tenants/hooks";
import { AppSelectionRequiredState } from "../components/AppSelectionRequiredState";
import { feedbackImage, getFeedback, listFeedback, replyFeedback, updateFeedback } from "../features/feedback/api";
import { feedbackStatuses, replySchema, type FeedbackSearch, type ReplyForm } from "../features/feedback/model";
import type { Feedback, FeedbackStatus } from "../generated/api/types.gen";

export function AdminFeedbackPage() {
  const { t } = useTranslation(); const scope = useAppScope(); const tenant = useTenantKey();
  return <div className="ak-page-container"><header className="ak-page-heading"><div><Typography.Title level={1}>{t("apps.feedbacks.title")}</Typography.Title><Typography.Paragraph>{t("apps.feedbacks.description")}</Typography.Paragraph></div></header>{scope.appId ? <Contents key={`${tenant}:${scope.appId}`} appId={scope.appId} /> : <AppSelectionRequiredState />}</div>;
}
function Contents({ appId }: { appId: string }) {
  const { t, i18n } = useTranslation(); const tenant = useTenantKey(); const client = useQueryClient();
  const search = useSearch({ from: "/app/feedbacks" }); const navigate = useNavigate({ from: "/app/feedbacks" });
  const [selected, setSelected] = useState<string | null>(null);
  const query = useQuery({ queryKey: ["tenant", tenant, "apps", appId, "feedbacks", search], queryFn: ({ signal }) => listFeedback(appId, search, signal) });
  const filters = (patch: Partial<FeedbackSearch>) => void navigate({ search: (previous) => ({ ...previous, ...patch, page: patch.page ?? 1 }) });
  const date = (value: string) => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
  const invalidate = async () => { await client.invalidateQueries({ queryKey: ["tenant", tenant, "apps", appId, "feedbacks"] }); };
  return <><Card><Space wrap style={{ marginBottom: 16 }} role="search" aria-label={t("apps.feedbacks.filters")}>
    <Input.Search key={search.q} defaultValue={search.q} allowClear maxLength={160} aria-label={t("apps.feedbacks.search")} placeholder={t("apps.feedbacks.search")} onSearch={(q) => { filters({ q }); }} />
    <Select style={{ minWidth: 140 }} aria-label={t("apps.feedbacks.status")} value={search.status} options={[{ value: "", label: t("apps.feedbacks.all") }, ...feedbackStatuses.map((value) => ({ value, label: t(`apps.feedbacks.status.${value}`) }))]} onChange={(status: FeedbackSearch["status"]) => { filters({ status }); }} />
    <DatePicker.RangePicker aria-label={t("apps.feedbacks.dateRange")} value={search.created_from && search.created_to ? [dayjs(search.created_from), dayjs(search.created_to)] : null} onChange={(range) => { filters({ created_from: range?.[0]?.startOf("day").toISOString(), created_to: range?.[1]?.endOf("day").toISOString() }); }} />
  </Space>
  {query.isError ? <Alert type="error" showIcon title={t("apps.feedbacks.failed")} action={<Button onClick={() => void query.refetch()}>{t("common.actions.retry")}</Button>} /> : null}
  <Table<Feedback> rowKey="id" dataSource={query.data?.items ?? []} loading={query.isPending} scroll={{ x: 760 }} locale={{ emptyText: t("apps.feedbacks.empty") }} pagination={{ current: search.page, pageSize: search.page_size, total: query.data?.total ?? 0, onChange: (page, page_size) => { filters({ page, page_size }); }, showSizeChanger: true }} columns={[
    { title: t("apps.feedbacks.content"), key: "description", render: (_, value) => <Typography.Paragraph style={{ maxWidth: 360, whiteSpace: "pre-wrap", marginBottom: 0 }} ellipsis={{ rows: 2 }}>{value.description}</Typography.Paragraph> },
    { title: t("apps.feedbacks.status"), dataIndex: "status", render: (value: FeedbackStatus) => <Tag color={value === "resolved" ? "success" : value === "processing" ? "processing" : "default"}>{t(`apps.feedbacks.status.${value}`)}</Tag> },
    { title: t("apps.feedbacks.platform"), dataIndex: "platform", render: (value: string) => t(`apps.feedbacks.platform.${value}`) },
    { title: t("apps.feedbacks.version"), dataIndex: "app_version" },
    { title: t("apps.feedbacks.created"), dataIndex: "created_at", render: date },
    { title: t("common.actions.view"), key: "action", render: (_, value) => <Button onClick={() => { setSelected(value.id); }}>{t("common.actions.view")}</Button> },
  ]} /></Card>
  <Drawer title={t("apps.feedbacks.detail")} open={selected !== null} onClose={() => { setSelected(null); }} size={640} destroyOnHidden>{selected ? <FeedbackDetail key={selected} appId={appId} id={selected} onSaved={invalidate} /> : null}</Drawer></>;
}
function PrivateImage({ appId, id, fileId }: { appId: string; id: string; fileId: string }) {
  const { t } = useTranslation(); const [url, setUrl] = useState(""); const [error, setError] = useState(false); const [attempt, setAttempt] = useState(0);
  useEffect(() => { const controller = new AbortController(); let objectUrl = ""; setError(false); setUrl(""); void feedbackImage(appId, id, fileId, controller.signal).then((blob) => { if (controller.signal.aborted) return; objectUrl = URL.createObjectURL(blob); setUrl(objectUrl); }).catch(() => { if (!controller.signal.aborted) setError(true); }); return () => { controller.abort(); if (objectUrl) URL.revokeObjectURL(objectUrl); }; }, [appId, id, fileId, attempt]);
  return error ? <Button onClick={() => { setAttempt(attempt + 1); }}>{t("apps.feedbacks.imageRetry")}</Button> : url ? <Image width={150} src={url} alt={t("apps.feedbacks.image")} /> : <Spin aria-label={t("common.loading")} />;
}
function FeedbackDetail({ appId, id, onSaved }: { appId: string; id: string; onSaved: () => Promise<void> }) {
  const { t, i18n } = useTranslation(); const tenant = useTenantKey(); const permissions = useAuthStore((state) => state.context?.permissions ?? []);
  const query = useQuery({ queryKey: ["tenant", tenant, "apps", appId, "feedbacks", "detail", id], queryFn: ({ signal }) => getFeedback(appId, id, signal) });
  const form = useForm<ReplyForm>({ defaultValues: { body: "", status: "processing" } }); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  const replay = useRef({ payload: "", key: "" }); const alive = useRef(true); useEffect(() => { alive.current = true; return () => { alive.current = false; }; }, []);
  const date = (value: string) => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
  const failure = (error: unknown) => error !== null && typeof error === "object" && "status" in error && error.status === 409 ? "apps.feedbacks.conflict" : "apps.feedbacks.failed";
  const change = async (status: FeedbackStatus) => { if (!query.data || busy) return; setBusy(true); setError(""); try { await updateFeedback(appId, id, { status, lock_version: query.data.lock_version }); if (alive.current) await onSaved(); } catch (cause) { if (alive.current) setError(failure(cause)); } finally { if (alive.current) setBusy(false); } };
  const submit = form.handleSubmit(async (value) => {
    if (!query.data || busy) return; const parsed = replySchema.safeParse(value); if (!parsed.success) { form.setError("body", { message: t("apps.feedbacks.replyRequired") }); return; }
    const input = { ...parsed.data, lock_version: query.data.lock_version }; const payload = JSON.stringify(input); if (payload !== replay.current.payload) replay.current = { payload, key: crypto.randomUUID() };
    setBusy(true); setError(""); try { await replyFeedback(appId, id, input, replay.current.key); if (alive.current) { form.reset(); replay.current = { payload: "", key: "" }; await onSaved(); } } catch (cause) { if (alive.current) setError(failure(cause)); } finally { if (alive.current) setBusy(false); }
  });
  if (query.isPending) return <Spin />;
  if (!query.data) return <Alert type="error" title={t("apps.feedbacks.failed")} action={<Button onClick={() => void query.refetch()}>{t("common.actions.retry")}</Button>} />;
  const item = query.data;
  return <Space orientation="vertical" size={20} style={{ width: "100%" }}>
    {error ? <Alert type="error" showIcon title={t(error)} action={<Button onClick={() => void query.refetch()}>{t("apps.feedbacks.refresh")}</Button>} /> : null}
    <Descriptions column={1} items={[
      { key: "status", label: t("apps.feedbacks.status"), children: t(`apps.feedbacks.status.${item.status}`) },
      { key: "user", label: t("apps.feedbacks.user"), children: item.user_id },
      { key: "contact", label: t("apps.feedbacks.contact"), children: item.contact || t("apps.feedbacks.notProvided") },
      { key: "platform", label: t("apps.feedbacks.platform"), children: t(`apps.feedbacks.platform.${item.platform}`) },
      { key: "version", label: t("apps.feedbacks.version"), children: item.app_version },
      { key: "created", label: t("apps.feedbacks.created"), children: date(item.created_at) },
    ]} />
    <Typography.Paragraph style={{ whiteSpace: "pre-wrap" }}>{item.description}</Typography.Paragraph>
    <Space wrap>{item.attachments.map((file) => <PrivateImage key={file.file_id} appId={appId} id={id} fileId={file.file_id} />)}</Space>
    {permissions.includes("app.feedback.update") ? <Space wrap>{feedbackStatuses.filter((status) => status !== item.status).map((status) => <Button disabled={busy} key={status} onClick={() => void change(status)}>{t("apps.feedbacks.markStatus", { status: t(`apps.feedbacks.status.${status}`) })}</Button>)}</Space> : null}
    <Typography.Title level={4}>{t("apps.feedbacks.replies")}</Typography.Title>
    {item.replies.length === 0 ? <Typography.Text type="secondary">{t("apps.feedbacks.noReply")}</Typography.Text> : item.replies.map((reply) => <Card size="small" key={reply.id}><Typography.Paragraph style={{ whiteSpace: "pre-wrap" }}>{reply.body}</Typography.Paragraph><Typography.Text type="secondary">{date(reply.created_at)}</Typography.Text></Card>)}
    {permissions.includes("app.feedback.reply") ? <Form layout="vertical" onFinish={() => void submit()}><Form.Item label={t("apps.feedbacks.reply")} validateStatus={form.formState.errors.body ? "error" : ""} help={form.formState.errors.body?.message}><Controller control={form.control} name="body" render={({ field }) => <Input.TextArea {...field} aria-label={t("apps.feedbacks.reply")} rows={5} maxLength={2000} showCount disabled={busy} />} /></Form.Item><Form.Item label={t("apps.feedbacks.replyStatus")}><Controller control={form.control} name="status" render={({ field }) => <Select {...field} aria-label={t("apps.feedbacks.replyStatus")} disabled={busy} options={feedbackStatuses.map((value) => ({ value, label: t(`apps.feedbacks.status.${value}`) }))} />} /></Form.Item><Button type="primary" htmlType="submit" loading={busy}>{t("apps.feedbacks.sendReply")}</Button></Form> : null}
    {item.events.length > 0 ? <><Typography.Title level={4}>{t("apps.feedbacks.history")}</Typography.Title>{item.events.map((event) => <Typography.Paragraph key={event.id}>{t(`apps.feedbacks.status.${event.from_status}`)} → {t(`apps.feedbacks.status.${event.to_status}`)} · {date(event.created_at)}</Typography.Paragraph>)}</> : null}
  </Space>;
}
