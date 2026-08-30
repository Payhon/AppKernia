import {
  Alert,
  Button,
  Card,
  Drawer,
  Form,
  Grid,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { Controller, useForm } from "react-hook-form";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import { z } from "zod";
import type {
  AdminNotificationMessage,
  AdminNotificationMessageRequest,
} from "../generated/api/types.gen";
import { authSession } from "../features/auth/store";
import { useAuthStore } from "../features/auth/store";
import {
  type NotificationKind,
  useNotificationMessages,
  useNotificationMessageMutations,
} from "../features/notifications/hooks";
import { sanitizeNotificationHtml } from "../features/notifications/sanitize";
import { useAdminUsers } from "../features/users/hooks";

interface Filters {
  q: string;
  status: string;
  message_type: string;
  page: number;
  page_size: number;
}
interface EditorValues {
  message_type: AdminNotificationMessageRequest["message_type"];
  title: string;
  body: string;
  body_format: AdminNotificationMessageRequest["body_format"];
  audience_scope: AdminNotificationMessageRequest["audience_scope"];
  audience_user_ids: string[];
  scheduled_at: string;
  expires_at: string;
  push_category: AdminNotificationMessageRequest["push_category"];
  push_ttl_seconds: number;
  push_collapse_key: string;
  push_route_key: NonNullable<AdminNotificationMessageRequest["push_route_key"]>;
}
const editorSchema = z
  .object({
    message_type: z.enum(["system", "notice", "private", "marketing", "security"]),
    title: z.string().trim().min(1).max(300),
    body: z.string().trim().min(1).max(100_000),
    body_format: z.enum(["plain", "markdown", "html"]),
    audience_scope: z.enum(["all", "selected"]),
    audience_user_ids: z.array(z.uuid()).max(500),
    scheduled_at: z.string(),
    expires_at: z.string(),
    push_category: z.enum(["service_security", "news_operations"]),
    push_ttl_seconds: z.number().int().min(300).max(604800),
    push_collapse_key: z.string().max(128).regex(/^[A-Za-z0-9._:-]*$/),
    push_route_key: z.enum(["", "notification.detail"]),
  })
  .refine(
    (value) =>
      value.audience_scope === "all" || value.audience_user_ids.length > 0,
    { path: ["audience_user_ids"], message: "required" },
  );
const defaults = (kind: NotificationKind): EditorValues => ({
  message_type: kind === "notices" ? "notice" : "system",
  title: "",
  body: "",
  body_format: "plain",
  audience_scope: "all",
  audience_user_ids: [],
  scheduled_at: "",
  expires_at: "",
  push_category: "service_security",
  push_ttl_seconds: 86400,
  push_collapse_key: "",
  push_route_key: "notification.detail",
});
const localDate = (value?: string | null) =>
  value ? new Date(value).toISOString().slice(0, 16) : "";
const inputFrom = (value: AdminNotificationMessage): EditorValues => ({
  message_type: value.message_type,
  title: value.title,
  body: value.body,
  body_format: value.body_format,
  audience_scope: value.audience_scope,
  audience_user_ids: value.audience_user_ids,
  scheduled_at: localDate(value.scheduled_at),
  expires_at: localDate(value.expires_at),
  push_category: value.push_category,
  push_ttl_seconds: value.push_ttl_seconds,
  push_collapse_key: value.push_collapse_key ?? "",
  push_route_key: value.push_route_key ?? "notification.detail",
});
const toRequest = (value: EditorValues): AdminNotificationMessageRequest => ({
  message_type: value.message_type,
  title: value.title.trim(),
  body: value.body.trim(),
  body_format: value.body_format,
  audience_scope: value.audience_scope,
  audience_user_ids:
    value.audience_scope === "selected" ? value.audience_user_ids : [],
  scheduled_at: value.scheduled_at
    ? new Date(value.scheduled_at).toISOString()
    : null,
  expires_at: value.expires_at
    ? new Date(value.expires_at).toISOString()
    : null,
  push_category: value.push_category,
  push_ttl_seconds: value.push_ttl_seconds,
  push_collapse_key: value.push_collapse_key,
  push_route_key: value.push_route_key,
  push_route_params: {},
});
const readFilters = (): Filters => {
  const params = new URLSearchParams(location.search);
  return {
    q: params.get("q") ?? "",
    status: params.get("status") ?? "",
    message_type: params.get("message_type") ?? "",
    page: Number(params.get("page") ?? 1),
    page_size: Number(params.get("page_size") ?? 20),
  };
};
const persistFilters = (filters: Filters) => {
  const params = new URLSearchParams();
  Object.entries(filters).forEach(([key, value]) => {
    if (value !== "" && !(key === "page" && value === 1) && !(key === "page_size" && value === 20))
      params.set(key, String(value));
  });
  history.replaceState(
    history.state,
    "",
    `${location.pathname}${params.size ? `?${params}` : ""}`,
  );
};

export function AdminNotificationMessagesPage({ kind }: { kind: NotificationKind }) {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(
    useAuthStore((state) => state.context?.permissions ?? []),
  );
  const prefix = kind === "notices" ? "notice" : "message";
  const [filters, setFiltersState] = useState(readFilters);
  const setFilters = (next: Filters) => {
    setFiltersState(next);
    persistFilters(next);
  };
  const query = useNotificationMessages(kind, filters);
  const mutations = useNotificationMessageMutations(kind);
  const users = useAdminUsers({ status: "active", page: 1, page_size: 100 });
  const [editing, setEditing] = useState<AdminNotificationMessage | "new" | null>(null);
  const [feedback, setFeedback] = useState<{ key: string; error?: boolean } | null>(null);
  const formatter = useMemo(
    () => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }),
    [i18n.language],
  );
  const form = useForm<EditorValues>({ defaultValues: defaults(kind) });
  const audienceScope = form.watch("audience_scope");
  const body = form.watch("body");
  const bodyFormat = form.watch("body_format");
  const openEditor = (message: AdminNotificationMessage | "new") => {
    setEditing(message);
    form.reset(message === "new" ? defaults(kind) : inputFrom(message));
  };
  const submit = form.handleSubmit(async (values) => {
    const parsed = editorSchema.safeParse(values);
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0];
        if (typeof field === "string")
          form.setError(field as keyof EditorValues, {
            message: t("notifications.editor.required"),
          });
      }
      return;
    }
    try {
      if (editing === "new") await mutations.create.mutateAsync(toRequest(parsed.data));
      else if (editing) await mutations.update.mutateAsync({ id: editing.id, input: toRequest(parsed.data) });
      setEditing(null);
      setFeedback({ key: "notifications.feedback.saved" });
    } catch {
      setFeedback({ key: "notifications.feedback.save_error", error: true });
    }
  });
  const publish = async (message: AdminNotificationMessage) => {
    try {
      const preview = await authSession.previewAdminNotificationRecipients(kind, message.id);
      Modal.confirm({
        title: t("notifications.publish.title"),
        content: (
          <div>
            <Typography.Paragraph>
              {t("notifications.publish.confirm", { title: message.title, count: preview.count })}
            </Typography.Paragraph>
            <ul>
              {preview.items.slice(0, 5).map((recipient) => (
                <li key={recipient.user_id}>{recipient.display_name} · {recipient.email_hint}</li>
              ))}
            </ul>
          </div>
        ),
        okText: t("notifications.actions.publish"),
        cancelText: t("common.actions.cancel"),
        onOk: async () => {
          await mutations.publish.mutateAsync(message.id);
          setFeedback({ key: "notifications.feedback.published" });
        },
      });
    } catch {
      setFeedback({ key: "notifications.feedback.preview_error", error: true });
    }
  };
  const cancel = (message: AdminNotificationMessage) => {
    Modal.confirm({
      title: t("notifications.cancel.title"),
      content: t("notifications.cancel.confirm", { title: message.title }),
      okText: t("notifications.actions.cancel_message"),
      okButtonProps: { danger: true },
      cancelText: t("common.actions.cancel"),
      onOk: async () => {
        await mutations.cancel.mutateAsync(message.id);
        setFeedback({ key: "notifications.feedback.cancelled" });
      },
    });
  };
  const columns: TableColumnsType<AdminNotificationMessage> = [
    { title: t("notifications.columns.title"), dataIndex: "title", render: (value: string) => <strong>{value}</strong> },
    ...(kind === "messages" ? [{ title: t("notifications.columns.type"), dataIndex: "message_type", render: (value: string) => t(`notifications.type.${value}`) }] : []),
    { title: t("notifications.columns.status"), dataIndex: "status", render: (value: string) => <Tag className={value === "published" ? "ak-status-success" : value === "cancelled" ? "ak-status-error" : "ak-status-warning"}>{t(`notifications.status.${value}`)}</Tag> },
    { title: t("notifications.columns.audience"), dataIndex: "audience_scope", render: (value: string, row) => t(`notifications.audience.${value}`, { count: row.audience_user_ids.length }), responsive: ["md"] },
    { title: t("notifications.columns.created"), dataIndex: "created_at", render: (value: string) => formatter.format(new Date(value)), responsive: ["lg"] },
    { title: t("notifications.columns.actions"), key: "actions", width: screens.md ? 330 : 150, render: (_, row) => <Space wrap><Button size="small" onClick={() => { void navigate({ to: kind === "notices" ? "/system/notifications/notices/$noticeId" : "/system/notifications/messages/$messageId", params: kind === "notices" ? { noticeId: row.id } : { messageId: row.id } }); }}>{t("common.actions.view")}</Button>{["draft", "scheduled"].includes(row.status) && permissions.has(`notify.${prefix}.update`) ? <Button size="small" onClick={() => { openEditor(row); }}>{t("common.actions.edit")}</Button> : null}{["draft", "scheduled"].includes(row.status) && permissions.has(`notify.${prefix}.publish`) && (row.push_category !== "news_operations" || permissions.has("notify.operations.publish")) ? <Button type="primary" size="small" onClick={() => void publish(row)}>{t("notifications.actions.publish")}</Button> : null}{row.status !== "cancelled" && permissions.has(`notify.${prefix}.cancel`) ? <Button danger size="small" onClick={() => { cancel(row); }}>{t("notifications.actions.cancel_message")}</Button> : null}</Space> },
  ];
  return (
    <div className="ak-page-container">
      <header className="ak-page-heading"><div><Typography.Title level={1}>{t(`notifications.${prefix}.title`)}</Typography.Title><Typography.Paragraph type="secondary">{t(`notifications.${prefix}.description`)}</Typography.Paragraph></div>{permissions.has(`notify.${prefix}.create`) ? <Button type="primary" onClick={() => { openEditor("new"); }}>{t("notifications.actions.create")}</Button> : null}</header>
      {feedback ? <div className={feedback.error ? "ak-form-error" : "ak-org-feedback"} role={feedback.error ? "alert" : "status"}>{t(feedback.key)}</div> : null}
      <Card>
        <div className="ak-settings-filters" role="search" aria-label={t("notifications.filters.landmark")}><Input.Search allowClear aria-label={t("notifications.filters.query")} value={filters.q} onChange={(event) => { setFilters({ ...filters, q: event.target.value, page: 1 }); }}/><Select allowClear aria-label={t("notifications.filters.status")} placeholder={t("notifications.filters.status")} value={filters.status || undefined} onChange={(value) => { setFilters({ ...filters, status: value ?? "", page: 1 }); }} options={["draft", "scheduled", "published", "cancelled"].map((value) => ({ value, label: t(`notifications.status.${value}`) }))}/>{kind === "messages" ? <Select allowClear aria-label={t("notifications.filters.type")} placeholder={t("notifications.filters.type")} value={filters.message_type || undefined} onChange={(value) => { setFilters({ ...filters, message_type: value ?? "", page: 1 }); }} options={["system", "private", "marketing", "security"].map((value) => ({ value, label: t(`notifications.type.${value}`) }))}/> : null}</div>
        {query.isError ? <Alert type="error" showIcon title={t("notifications.feedback.load_error")} action={<Button onClick={() => void query.refetch()}>{t("common.actions.retry")}</Button>}/> : null}
        <div className="ak-table-scroll"><Table rowKey="id" columns={columns} dataSource={query.data?.items ?? []} loading={query.isPending} locale={{ emptyText: t(`notifications.${prefix}.empty`) }} {...((query.data?.items.length ?? 0) > 0 ? { scroll: { x: 920 } } : {})} pagination={{ current: filters.page, pageSize: filters.page_size, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (page, page_size) => { setFilters({ ...filters, page, page_size }); } }}/></div>
      </Card>
      <Drawer rootClassName="ak-notification-drawer" destroyOnHidden size={screens.md ? "large" : "100%"} open={editing !== null} title={t(editing === "new" ? "notifications.editor.create_title" : "notifications.editor.edit_title")} onClose={() => { setEditing(null); }} extra={<Button type="primary" loading={mutations.create.isPending || mutations.update.isPending} onClick={() => void submit()}>{t("common.actions.save")}</Button>}>
        <Form layout="vertical" onFinish={() => void submit()}>
          {kind === "messages" ? <Form.Item htmlFor="notification-message-type" label={t("notifications.editor.type")} {...(form.formState.errors.message_type ? { validateStatus: "error" as const } : {})}><Controller control={form.control} name="message_type" render={({ field }) => <Select {...field} id="notification-message-type" options={["system", "private", "marketing", "security"].map((value) => ({ value, label: t(`notifications.type.${value}`) }))}/>} /></Form.Item> : null}
          <Form.Item htmlFor="notification-title" label={t("notifications.editor.title")} {...(form.formState.errors.title ? { validateStatus: "error" as const, help: t("notifications.editor.required") } : {})}><Controller control={form.control} name="title" render={({ field }) => <Input {...field} id="notification-title" maxLength={300} showCount/>}/></Form.Item>
          <Form.Item htmlFor="notification-body-format" label={t("notifications.editor.format")}><Controller control={form.control} name="body_format" render={({ field }) => <Select {...field} id="notification-body-format" options={["plain", "markdown", "html"].map((value) => ({ value, label: t(`notifications.format.${value}`) }))}/>} /></Form.Item>
          <Form.Item htmlFor="notification-body" label={t("notifications.editor.body")} {...(form.formState.errors.body ? { validateStatus: "error" as const, help: t("notifications.editor.required") } : { help: t("notifications.editor.sanitization_hint") })}><Controller control={form.control} name="body" render={({ field }) => <Input.TextArea {...field} id="notification-body" rows={10} maxLength={100000}/>}/></Form.Item>
          <Card size="small" title={t("notifications.editor.preview")}><div className="ak-notification-preview" aria-live="polite">{bodyFormat === "html" ? <div dangerouslySetInnerHTML={{ __html: sanitizeNotificationHtml(body) }}/> : <pre>{body}</pre>}</div></Card>
          <Form.Item htmlFor="notification-audience" label={t("notifications.editor.audience")}><Controller control={form.control} name="audience_scope" render={({ field }) => <Select {...field} id="notification-audience" options={["all", "selected"].map((value) => ({ value, label: t(`notifications.audience.${value}`, { count: 0 }) }))}/>} /></Form.Item>
          {audienceScope === "selected" ? <Form.Item htmlFor="notification-recipients" label={t("notifications.editor.recipients")} {...(form.formState.errors.audience_user_ids ? { validateStatus: "error" as const, help: t("notifications.editor.required") } : { help: t("notifications.editor.recipient_hint") })}><Controller control={form.control} name="audience_user_ids" render={({ field }) => <Select {...field} id="notification-recipients" mode="multiple" showSearch={{ optionFilterProp: "label" }} loading={users.isPending} options={(users.data?.items ?? []).map((user) => ({ value: user.id, label: `${user.display_name} · ${user.email}` }))}/>} /></Form.Item> : null}
          <Form.Item htmlFor="notification-scheduled-at" label={t("notifications.editor.scheduled_at")}><Controller control={form.control} name="scheduled_at" render={({ field }) => <Input {...field} id="notification-scheduled-at" type="datetime-local"/>}/></Form.Item>
          <Form.Item htmlFor="notification-expires-at" label={t("notifications.editor.expires_at")}><Controller control={form.control} name="expires_at" render={({ field }) => <Input {...field} id="notification-expires-at" type="datetime-local"/>}/></Form.Item>
          <Card size="small" title={t("notifications.editor.push_section")}>
            <div className="ak-form-grid-2"><Form.Item label={t("notifications.editor.push_category")}><Controller control={form.control} name="push_category" render={({ field }) => <Select {...field} options={["service_security", "news_operations"].map((value) => ({ value, label: t(`notifications.push_category.${value}`) }))}/>} /></Form.Item><Form.Item label={t("notifications.editor.push_ttl")}><Controller control={form.control} name="push_ttl_seconds" render={({ field }) => <InputNumber className="ak-full-width" min={300} max={604800} step={300} value={field.value} onChange={(value) => { field.onChange(value ?? 86400); }}/>} /></Form.Item></div>
            <div className="ak-form-grid-2"><Form.Item label={t("notifications.editor.push_collapse_key")}><Controller control={form.control} name="push_collapse_key" render={({ field }) => <Input {...field} maxLength={128}/>} /></Form.Item><Form.Item label={t("notifications.editor.push_route")}><Controller control={form.control} name="push_route_key" render={({ field }) => <Select {...field} options={[{ value: "notification.detail", label: t("notifications.push_route.notification_detail") }, { value: "", label: t("notifications.push_route.none") }]}/>} /></Form.Item></div>
          </Card>
        </Form>
      </Drawer>
    </div>
  );
}
