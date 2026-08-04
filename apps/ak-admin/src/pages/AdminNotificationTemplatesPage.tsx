import {
  Alert,
  Button,
  Card,
  Drawer,
  Form,
  Input,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { Controller, useForm } from "react-hook-form";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import type {
  AdminNotificationTemplate,
  AdminNotificationTemplateRequest,
} from "../generated/api/types.gen";
import { useAuthStore } from "../features/auth/store";
import {
  useNotificationTemplates,
  useNotificationTemplateMutations,
} from "../features/notifications/hooks";

interface Filters {
  q: string;
  status: string;
  channel: string;
  locale: string;
  page: number;
  page_size: number;
}
interface Values {
  code: string;
  name: string;
  channel: AdminNotificationTemplateRequest["channel"];
  locale: "zh-CN" | "en-US" | "global";
  subject_template: string;
  body_template: string;
  variables_schema: string;
  status: "active" | "disabled";
}
const schema = z.object({
  code: z.string().regex(/^[a-z][a-z0-9_.-]{1,95}$/),
  name: z.string().trim().min(1).max(160),
  channel: z.enum(["in_app", "email", "sms", "push", "webhook"]),
  locale: z.enum(["zh-CN", "en-US", "global"]),
  subject_template: z.string().max(500),
  body_template: z.string().trim().min(1).max(100_000),
  variables_schema: z.string().refine((raw) => {
    try {
      const value: unknown = JSON.parse(raw);
      return (
        typeof value === "object" && value !== null && !Array.isArray(value)
      );
    } catch {
      return false;
    }
  }),
  status: z.enum(["active", "disabled"]),
});
const empty: Values = {
  code: "",
  name: "",
  channel: "in_app",
  locale: "zh-CN",
  subject_template: "",
  body_template: "",
  variables_schema: '{\n  "type": "object",\n  "properties": {}\n}',
  status: "active",
};
const fromTemplate = (value: AdminNotificationTemplate): Values => ({
  code: value.code,
  name: value.name,
  channel: value.channel,
  locale: value.locale ?? "global",
  subject_template: value.subject_template ?? "",
  body_template: value.body_template,
  variables_schema: JSON.stringify(value.variables_schema, null, 2),
  status: value.status,
});
const toRequest = (value: Values): AdminNotificationTemplateRequest => ({
  code: value.code.trim(),
  name: value.name.trim(),
  channel: value.channel,
  locale: value.locale === "global" ? null : value.locale,
  subject_template: value.subject_template.trim() || null,
  body_template: value.body_template.trim(),
  variables_schema: JSON.parse(value.variables_schema) as Record<
    string,
    unknown
  >,
  status: value.status,
});
const read = (): Filters => {
  const p = new URLSearchParams(location.search);
  return {
    q: p.get("q") ?? "",
    status: p.get("status") ?? "",
    channel: p.get("channel") ?? "",
    locale: p.get("locale") ?? "",
    page: Number(p.get("page") ?? 1),
    page_size: Number(p.get("page_size") ?? 20),
  };
};
const persist = (f: Filters) => {
  const p = new URLSearchParams();
  Object.entries(f).forEach(([k, v]) => {
    if (
      v !== "" &&
      !(k === "page" && v === 1) &&
      !(k === "page_size" && v === 20)
    )
      p.set(k, String(v));
  });
  history.replaceState(
    history.state,
    "",
    `${location.pathname}${p.size ? `?${p}` : ""}`,
  );
};

export function AdminNotificationTemplatesPage() {
  const { t } = useTranslation();
  const permissions = new Set(
    useAuthStore((state) => state.context?.permissions ?? []),
  );
  const [filters, setState] = useState(read);
  const setFilters = (next: Filters) => {
    setState(next);
    persist(next);
  };
  const query = useNotificationTemplates(filters);
  const mutations = useNotificationTemplateMutations();
  const [editing, setEditing] = useState<
    AdminNotificationTemplate | "new" | null
  >(null);
  const [feedback, setFeedback] = useState<{
    key: string;
    error?: boolean;
  } | null>(null);
  const form = useForm<Values>({ defaultValues: empty });
  const open = (value: AdminNotificationTemplate | "new") => {
    setEditing(value);
    form.reset(value === "new" ? empty : fromTemplate(value));
  };
  const submit = form.handleSubmit(async (values) => {
    const parsed = schema.safeParse(values);
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0];
        if (typeof field === "string")
          form.setError(field as keyof Values, {
            message: t("notifications.editor.invalid"),
          });
      }
      return;
    }
    try {
      if (editing === "new")
        await mutations.create.mutateAsync(toRequest(parsed.data));
      else if (editing)
        await mutations.update.mutateAsync({
          id: editing.id,
          input: toRequest(parsed.data),
        });
      setEditing(null);
      setFeedback({ key: "notifications.feedback.saved" });
    } catch {
      setFeedback({ key: "notifications.feedback.save_error", error: true });
    }
  });
  const columns: TableColumnsType<AdminNotificationTemplate> = [
    {
      title: t("notifications.templates.columns.name"),
      key: "name",
      render: (_, x) => (
        <div>
          <strong>{x.name}</strong>
          <div className="ak-table-secondary">{x.code}</div>
        </div>
      ),
    },
    {
      title: t("notifications.templates.columns.channel"),
      dataIndex: "channel",
      render: (v: string) => t(`notifications.channel.${v}`),
    },
    {
      title: t("notifications.templates.columns.locale"),
      dataIndex: "locale",
      render: (v: string | null) => v ?? t("notifications.locale.global"),
    },
    {
      title: t("notifications.columns.status"),
      dataIndex: "status",
      render: (v: string) => (
        <Tag className={v === "active" ? "ak-status-success" : ""}>
          {t(`notifications.template_status.${v}`)}
        </Tag>
      ),
    },
    {
      title: t("notifications.columns.actions"),
      key: "actions",
      render: (_, x) =>
        permissions.has("notify.template.update") ? (
          <Button
            size="small"
            onClick={() => {
              open(x);
            }}
          >
            {t("common.actions.edit")}
          </Button>
        ) : null,
    },
  ];
  return (
    <div className="ak-page-container">
      <header className="ak-page-heading">
        <div>
          <Typography.Title level={1}>
            {t("notifications.templates.title")}
          </Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("notifications.templates.description")}
          </Typography.Paragraph>
        </div>
        {permissions.has("notify.template.create") ? (
          <Button
            type="primary"
            onClick={() => {
              open("new");
            }}
          >
            {t("notifications.templates.create")}
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
      <Card>
        <div
          className="ak-settings-filters"
          role="search"
          aria-label={t("notifications.filters.landmark")}
        >
          <Input.Search
            allowClear
            aria-label={t("notifications.filters.query")}
            value={filters.q}
            onChange={(e) => {
              setFilters({ ...filters, q: e.target.value, page: 1 });
            }}
          />
          <Select
            allowClear
            aria-label={t("notifications.templates.filters.channel")}
            placeholder={t("notifications.templates.filters.channel")}
            value={filters.channel || undefined}
            onChange={(v) => {
              setFilters({ ...filters, channel: v ?? "", page: 1 });
            }}
            options={["in_app", "email", "sms", "push", "webhook"].map((v) => ({
              value: v,
              label: t(`notifications.channel.${v}`),
            }))}
          />
          <Select
            allowClear
            aria-label={t("notifications.templates.filters.locale")}
            placeholder={t("notifications.templates.filters.locale")}
            value={filters.locale || undefined}
            onChange={(v) => {
              setFilters({ ...filters, locale: v ?? "", page: 1 });
            }}
            options={["zh-CN", "en-US", "global"].map((v) => ({
              value: v,
              label: v === "global" ? t("notifications.locale.global") : v,
            }))}
          />
        </div>
        {query.isError ? (
          <Alert
            type="error"
            showIcon
            title={t("notifications.feedback.load_error")}
            action={
              <Button onClick={() => void query.refetch()}>
                {t("common.actions.retry")}
              </Button>
            }
          />
        ) : null}
        <div className="ak-table-scroll">
          <Table
            rowKey="id"
            columns={columns}
            dataSource={query.data?.items ?? []}
            loading={query.isPending}
            locale={{ emptyText: t("notifications.templates.empty") }}
            {...((query.data?.items.length ?? 0) > 0
              ? { scroll: { x: 760 } }
              : {})}
            pagination={{
              current: filters.page,
              pageSize: filters.page_size,
              total: query.data?.total ?? 0,
              showSizeChanger: true,
              onChange: (page, page_size) => {
                setFilters({ ...filters, page, page_size });
              },
            }}
          />
        </div>
      </Card>
      <Drawer
        rootClassName="ak-notification-drawer"
        destroyOnHidden
        size="large"
        open={editing !== null}
        onClose={() => {
          setEditing(null);
        }}
        title={t(
          editing === "new"
            ? "notifications.templates.editor.create"
            : "notifications.templates.editor.edit",
        )}
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
          <Form.Item
            htmlFor="notification-template-code"
            label={t("notifications.templates.editor.code")}
            {...(form.formState.errors.code
              ? {
                  validateStatus: "error" as const,
                  help: t("notifications.editor.invalid"),
                }
              : {})}
          >
            <Controller
              control={form.control}
              name="code"
              render={({ field }) => <Input {...field} id="notification-template-code" />}
            />
          </Form.Item>
          <Form.Item htmlFor="notification-template-name" label={t("notifications.templates.editor.name")}>
            <Controller
              control={form.control}
              name="name"
              render={({ field }) => <Input {...field} id="notification-template-name" />}
            />
          </Form.Item>
          <Space align="start" wrap>
            <Form.Item htmlFor="notification-template-channel" label={t("notifications.templates.columns.channel")}>
              <Controller
                control={form.control}
                name="channel"
                render={({ field }) => (
                  <Select
                    {...field}
                    id="notification-template-channel"
                    style={{ width: 180 }}
                    options={["in_app", "email", "sms", "push", "webhook"].map(
                      (v) => ({
                        value: v,
                        label: t(`notifications.channel.${v}`),
                      }),
                    )}
                  />
                )}
              />
            </Form.Item>
            <Form.Item htmlFor="notification-template-locale" label={t("notifications.templates.columns.locale")}>
              <Controller
                control={form.control}
                name="locale"
                render={({ field }) => (
                  <Select
                    {...field}
                    id="notification-template-locale"
                    style={{ width: 180 }}
                    options={["zh-CN", "en-US", "global"].map((v) => ({
                      value: v,
                      label:
                        v === "global" ? t("notifications.locale.global") : v,
                    }))}
                  />
                )}
              />
            </Form.Item>
            <Form.Item htmlFor="notification-template-status" label={t("notifications.columns.status")}>
              <Controller
                control={form.control}
                name="status"
                render={({ field }) => (
                  <Select
                    {...field}
                    id="notification-template-status"
                    style={{ width: 160 }}
                    options={["active", "disabled"].map((v) => ({
                      value: v,
                      label: t(`notifications.template_status.${v}`),
                    }))}
                  />
                )}
              />
            </Form.Item>
          </Space>
          <Form.Item htmlFor="notification-template-subject" label={t("notifications.templates.editor.subject")}>
            <Controller
              control={form.control}
              name="subject_template"
              render={({ field }) => <Input {...field} id="notification-template-subject" maxLength={500} />}
            />
          </Form.Item>
          <Form.Item
            htmlFor="notification-template-body"
            label={t("notifications.templates.editor.body")}
            {...(form.formState.errors.body_template
              ? { validateStatus: "error" as const }
              : {})}
          >
            <Controller
              control={form.control}
              name="body_template"
              render={({ field }) => (
                <Input.TextArea {...field} id="notification-template-body" rows={8} maxLength={100000} />
              )}
            />
          </Form.Item>
          <Form.Item
            htmlFor="notification-template-schema"
            label={t("notifications.templates.editor.schema")}
            {...(form.formState.errors.variables_schema
              ? {
                  validateStatus: "error" as const,
                  help: t("notifications.templates.editor.schema_error"),
                }
              : { help: t("notifications.templates.editor.schema_hint") })}
          >
            <Controller
              control={form.control}
              name="variables_schema"
              render={({ field }) => (
                <Input.TextArea
                  {...field}
                  id="notification-template-schema"
                  rows={10}
                  className="ak-code-input"
                />
              )}
            />
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  );
}
