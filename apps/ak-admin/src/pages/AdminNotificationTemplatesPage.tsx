import {
  Alert,
  Button,
  Card,
  Checkbox,
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
  AdminSmsTemplateBindingRequest,
  AdminNotificationTemplate,
  AdminNotificationTemplateRequest,
} from "../generated/api/types.gen";
import { useAuthStore } from "../features/auth/store";
import {
  useNotificationTemplates,
  useNotificationTemplateMutations,
  useSmsTemplateBindings,
} from "../features/notifications/hooks";
import { useAdminDictionary } from "../features/settings/hooks";

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
  body_format: "plain" | "html";
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
  body_format: z.enum(["plain", "html"]),
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
  body_format: "plain",
  variables_schema: '{\n  "type": "object",\n  "properties": {}\n}',
  status: "active",
};
type SmsProvider = "tencent" | "aliyun";
const isSmsProvider = (value: string): value is SmsProvider =>
  value === "tencent" || value === "aliyun";
const fromTemplate = (value: AdminNotificationTemplate): Values => ({
  code: value.code,
  name: value.name,
  channel: value.channel,
  locale: value.locale ?? "global",
  subject_template: value.subject_template ?? "",
  body_template: value.body_template,
  body_format: value.body_format,
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
  body_format: value.body_format,
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
  const channel = form.watch("channel");
  const smsEvents = useAdminDictionary("notification.sms.template_event");
  const emailEvents = useAdminDictionary("notification.email.template_event");
  const smsProviders = useAdminDictionary("sms.provider");
  const smsProviderOptions = (smsProviders.data?.items ?? [])
    .filter((item) => isSmsProvider(item.value))
    .map((item) => ({ value: item.value as SmsProvider, label: item.label }));
  const defaultSmsProvider = smsProviderOptions[0]?.value ?? "tencent";
  const [bindingTarget, setBindingTarget] = useState<AdminNotificationTemplate | null>(null);
  const bindings = useSmsTemplateBindings(bindingTarget?.id ?? null);
  const bindingForm = useForm<{
    provider: SmsProvider;
    external_template_id: string;
    sign_name: string;
    parameter_order: string;
    status: "active" | "disabled";
  }>({ defaultValues: { provider: "tencent", external_template_id: "", sign_name: "", parameter_order: "", status: "active" } });
  const [testTarget, setTestTarget] = useState<AdminNotificationTemplate | null>(null);
  const testForm = useForm<{
    target: string;
    provider: "smtp" | "tencent" | "aliyun";
    variables: string;
    confirm_billable: boolean;
  }>({ defaultValues: { target: "", provider: "smtp", variables: "{}", confirm_billable: false } });
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
  const submitBinding = bindingForm.handleSubmit(async (values) => {
    if (!bindingTarget) return;
    const input: AdminSmsTemplateBindingRequest = {
      external_template_id: values.external_template_id.trim(),
      sign_name: values.sign_name.trim(),
      parameter_order: values.parameter_order.split(",").map((value) => value.trim()).filter(Boolean),
      status: values.status,
      version: 1,
    };
    try {
      await mutations.upsertBinding.mutateAsync({ id: bindingTarget.id, provider: values.provider, input });
      setFeedback({ key: "notifications.templates.bindings.saved" });
      void bindings.refetch();
    } catch {
      setFeedback({ key: "notifications.feedback.save_error", error: true });
    }
  });
  const submitTest = testForm.handleSubmit(async (values) => {
    if (!testTarget) return;
    try {
      const variables = JSON.parse(values.variables) as Record<string, string>;
      await mutations.test.mutateAsync({
        id: testTarget.id,
        input: {
          target: values.target.trim(),
          provider: testTarget.channel === "email" ? "smtp" : values.provider,
          variables,
          confirm_billable: testTarget.channel === "sms" ? values.confirm_billable : false,
        },
      });
      setTestTarget(null);
      setFeedback({ key: "notifications.templates.test.queued" });
    } catch {
      setFeedback({ key: "notifications.templates.test.error", error: true });
    }
  });
  const columns: TableColumnsType<AdminNotificationTemplate> = [
    {
      title: t("notifications.templates.columns.name"),
      key: "name",
      render: (_, x) => (
        <div>
          <strong>{x.name}</strong>
          <div className="ak-table-secondary">
            {x.code} {x.is_locked ? <Tag>{t("settings.common.system_locked")}</Tag> : null}
          </div>
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
      render: (_, x) => (
        <Space wrap>
          {permissions.has("notify.template.update") && !x.is_locked ? (
            <Button size="small" onClick={() => { open(x); }}>{t("common.actions.edit")}</Button>
          ) : null}
          {x.channel === "sms" && permissions.has("notify.template.update") ? (
            <Button size="small" onClick={() => {
              setBindingTarget(x);
              bindingForm.reset({ provider: defaultSmsProvider, external_template_id: "", sign_name: "", parameter_order: "", status: "active" });
            }}>{t("notifications.templates.bindings.action")}</Button>
          ) : null}
          {(x.channel === "sms" || x.channel === "email") && permissions.has("notify.template.test") ? (
            <Button size="small" danger={x.channel === "sms"} onClick={() => {
              setTestTarget(x);
              testForm.reset({ target: "", provider: x.channel === "email" ? "smtp" : defaultSmsProvider, variables: JSON.stringify(Object.fromEntries(Object.keys((x.variables_schema["properties"] as Record<string, unknown> | undefined) ?? {}).map((key) => [key, ""])), null, 2), confirm_billable: false });
            }}>{t("notifications.templates.test.action")}</Button>
          ) : null}
        </Space>
      ),
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
              render={({ field }) =>
                channel === "email" || channel === "sms" ? (
                  <Select
                    {...field}
                    id="notification-template-code"
                    loading={channel === "email" ? emailEvents.isPending : smsEvents.isPending}
                    options={(channel === "email" ? emailEvents.data?.items : smsEvents.data?.items)?.map((item) => ({ value: item.value, label: item.label })) ?? []}
                  />
                ) : (
                  <Input {...field} id="notification-template-code" />
                )
              }
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
                    onChange={(value) => {
                      field.onChange(value);
                      if (value === "sms") form.setValue("body_format", "plain");
                      if (value === "email" || value === "sms") form.setValue("code", "");
                    }}
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
            <Form.Item htmlFor="notification-template-body-format" label={t("notifications.templates.editor.body_format")}>
              <Controller
                control={form.control}
                name="body_format"
                render={({ field }) => (
                  <Select
                    {...field}
                    disabled={channel === "sms"}
                    id="notification-template-body-format"
                    style={{ width: 180 }}
                    options={["plain", "html"].map((value) => ({ value, label: t(`notifications.format.${value}`) }))}
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
      <Drawer
        destroyOnHidden
        open={bindingTarget !== null}
        onClose={() => { setBindingTarget(null); }}
        title={t("notifications.templates.bindings.title")}
        extra={<Button type="primary" loading={mutations.upsertBinding.isPending} onClick={() => void submitBinding()}>{t("common.actions.save")}</Button>}
      >
        <Alert showIcon type="info" title={t("notifications.templates.bindings.reviewed_title")} description={t("notifications.templates.bindings.reviewed_description")} />
        <Form layout="vertical" onFinish={() => void submitBinding()}>
          <Controller control={bindingForm.control} name="provider" render={({ field }) => (
            <Form.Item htmlFor="notification-sms-binding-provider" label={t("notifications.templates.bindings.provider")}>
              <Select {...field} id="notification-sms-binding-provider" disabled={smsProviders.isPending || smsProviders.isError} loading={smsProviders.isPending} options={smsProviderOptions} onChange={(provider: SmsProvider) => {
                field.onChange(provider);
                const existing = bindings.data?.find((item) => item.provider === provider);
                bindingForm.reset({ provider, external_template_id: existing?.external_template_id ?? "", sign_name: existing?.sign_name ?? "", parameter_order: existing?.parameter_order.join(", ") ?? "", status: existing?.status ?? "active" });
              }} />
            </Form.Item>
          )} />
          <Controller control={bindingForm.control} name="external_template_id" render={({ field }) => <Form.Item htmlFor="notification-sms-binding-template-id" label={t("notifications.templates.bindings.template_id")} required><Input {...field} id="notification-sms-binding-template-id" /></Form.Item>} />
          <Controller control={bindingForm.control} name="sign_name" render={({ field }) => <Form.Item htmlFor="notification-sms-binding-sign" label={t("notifications.templates.bindings.sign_name")}><Input {...field} id="notification-sms-binding-sign" /></Form.Item>} />
          <Controller control={bindingForm.control} name="parameter_order" render={({ field }) => <Form.Item htmlFor="notification-sms-binding-order" label={t("notifications.templates.bindings.parameter_order")} help={t("notifications.templates.bindings.parameter_order_hint")}><Input {...field} id="notification-sms-binding-order" placeholder="code, expires_minutes" /></Form.Item>} />
          <Controller control={bindingForm.control} name="status" render={({ field }) => <Form.Item htmlFor="notification-sms-binding-status" label={t("notifications.columns.status")}><Select {...field} id="notification-sms-binding-status" options={["active", "disabled"].map((value) => ({ value, label: t(`notifications.template_status.${value}`) }))} /></Form.Item>} />
          {bindings.data?.some((item) => item.provider === bindingForm.watch("provider")) ? (
            <Button danger loading={mutations.deleteBinding.isPending} onClick={() => {
              if (!bindingTarget) return;
              void mutations.deleteBinding.mutateAsync({ id: bindingTarget.id, provider: bindingForm.getValues("provider") }).then(() => bindings.refetch());
            }}>{t("common.actions.delete")}</Button>
          ) : null}
        </Form>
      </Drawer>
      <Drawer
        destroyOnHidden
        open={testTarget !== null}
        onClose={() => { setTestTarget(null); }}
        title={t("notifications.templates.test.title")}
        extra={<Button danger={testTarget?.channel === "sms"} type="primary" loading={mutations.test.isPending} onClick={() => void submitTest()}>{t("notifications.templates.test.send")}</Button>}
      >
        {testTarget?.channel === "sms" ? <Alert showIcon type="warning" title={t("notifications.templates.test.billable_title")} description={t("notifications.templates.test.billable_description")} /> : null}
        <Form layout="vertical" onFinish={() => void submitTest()}>
          {testTarget?.channel === "sms" ? <Controller control={testForm.control} name="provider" render={({ field }) => <Form.Item htmlFor="notification-template-test-provider" label={t("notifications.templates.bindings.provider")}><Select {...field} id="notification-template-test-provider" disabled={smsProviders.isPending || smsProviders.isError} loading={smsProviders.isPending} options={smsProviderOptions} /></Form.Item>} /> : null}
          <Controller control={testForm.control} name="target" render={({ field }) => <Form.Item htmlFor="notification-template-test-target" label={t("notifications.templates.test.target")} required><Input {...field} id="notification-template-test-target" type={testTarget?.channel === "email" ? "email" : "tel"} /></Form.Item>} />
          <Controller control={testForm.control} name="variables" render={({ field }) => <Form.Item htmlFor="notification-template-test-variables" label={t("notifications.templates.test.variables")}><Input.TextArea {...field} id="notification-template-test-variables" className="ak-code-input" rows={10} /></Form.Item>} />
          {testTarget?.channel === "sms" ? <Controller control={testForm.control} name="confirm_billable" render={({ field }) => <Form.Item><Checkbox id="notification-template-test-confirm" checked={field.value} onChange={(event) => { field.onChange(event.target.checked); }}>{t("notifications.templates.test.confirm_billable")}</Checkbox></Form.Item>} /> : null}
        </Form>
      </Drawer>
    </div>
  );
}
