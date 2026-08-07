import { Alert, Button, Card, Drawer, Form, Grid, Input, Select, Space, Switch, Table, Tag, Typography, type TableColumnsType } from "antd";
import { Controller, useForm } from "react-hook-form";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAuthStore } from "../features/auth/store";
import { useApplicationMutations, useManagedApplications } from "../features/apps/hooks";
import { applicationInputSchema, type ApplicationInput, type ManagedApplication } from "../features/apps/model";

type Editor = ManagedApplication | "new" | null;
const defaults = (): ApplicationInput => ({ name: "", default_locale: "zh-CN", registration_enabled: true, registration_verification_mode: "email_otp" });
const fromItem = (item: ManagedApplication): ApplicationInput => ({ name: item.name, default_locale: item.default_locale, registration_enabled: item.registration_enabled, registration_verification_mode: item.registration_verification_mode, lock_version: item.lock_version });

export function AdminApplicationsPage() {
  const { t, i18n } = useTranslation();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [q, setQ] = useState("");
  const [editor, setEditor] = useState<Editor>(null);
  const [feedback, setFeedback] = useState<{ key: string; error: boolean } | null>(null);
  const form = useForm<ApplicationInput>({ defaultValues: defaults() });
  const applications = useManagedApplications({ q, page: 1, page_size: 100 });
  const mutations = useApplicationMutations();
  const date = useMemo(() => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }), [i18n.language]);
  const open = (value: ManagedApplication | "new") => { form.reset(value === "new" ? defaults() : fromItem(value)); setEditor(value); };
  const save = form.handleSubmit(async (values) => {
    const parsed = applicationInputSchema.safeParse(values);
    if (!parsed.success) { setFeedback({ key: "apps.feedback.validation_error", error: true }); return; }
    try { if (editor === "new") await mutations.create.mutateAsync(parsed.data); else if (editor) await mutations.update.mutateAsync({ id: editor.id, input: parsed.data }); setEditor(null); setFeedback({ key: "apps.feedback.saved", error: false }); } catch { setFeedback({ key: "apps.feedback.save_error", error: true }); }
  });
  const setStatus = async (item: ManagedApplication) => {
    try { await mutations.status.mutateAsync({ id: item.id, action: item.status === "active" ? "disable" : "enable", lockVersion: item.lock_version }); setFeedback({ key: "apps.feedback.status_saved", error: false }); } catch { setFeedback({ key: "apps.feedback.save_error", error: true }); }
  };
  const columns: TableColumnsType<ManagedApplication> = [
    { title: t("apps.application.columns.name"), render: (_, item) => <div><strong>{item.name}</strong><div className="ak-content-slug">{item.id}</div></div> },
    { title: t("apps.application.columns.registration"), responsive: ["md"], render: (_, item) => <Space wrap><Tag>{t(item.registration_enabled ? "apps.registration.enabled" : "apps.registration.disabled")}</Tag><Tag>{t(`apps.registration.verification.${item.registration_verification_mode}`)}</Tag></Space> },
    { title: t("apps.application.columns.status"), dataIndex: "status", width: 130, render: (value: ManagedApplication["status"]) => <Tag className={value === "active" ? "ak-status-success" : "ak-status-error"}>{t(`apps.status.${value}`)}</Tag> },
    { title: t("apps.application.columns.updated"), dataIndex: "updated_at", responsive: ["lg"], render: (value: string) => date.format(new Date(value)) },
    { title: t("apps.application.columns.actions"), width: screens.md ? 190 : 110, render: (_, item) => <Space wrap>{permissions.has("app.application.update") ? <Button size="small" onClick={() => { open(item); }}>{t("common.actions.edit")}</Button> : null}{permissions.has("app.application.disable") ? <Button size="small" danger={item.status === "active"} loading={mutations.status.isPending} onClick={() => void setStatus(item)}>{t(item.status === "active" ? "apps.actions.disable" : "apps.actions.enable")}</Button> : null}</Space> },
  ];
  return <div className="ak-page-container">
    <header className="ak-page-heading"><div><Typography.Title level={1}>{t("apps.application.title")}</Typography.Title><Typography.Paragraph type="secondary">{t("apps.application.description")}</Typography.Paragraph></div>{permissions.has("app.application.create") ? <Button type="primary" onClick={() => { open("new"); }}>{t("apps.application.actions.create")}</Button> : null}</header>
    {feedback ? <Alert showIcon type={feedback.error ? "error" : "success"} title={t(feedback.key)} /> : null}
    <Card><Input.Search allowClear aria-label={t("apps.application.filters.q")} onChange={(event) => { setQ(event.target.value); }} placeholder={t("apps.application.filters.q")} value={q} />
      {applications.isError ? <Alert showIcon type="error" title={t("apps.feedback.load_error")} action={<Button onClick={() => void applications.refetch()}>{t("common.actions.retry")}</Button>} /> : null}
      <div className="ak-table-scroll"><Table columns={columns} dataSource={applications.data?.items ?? []} loading={applications.isPending} locale={{ emptyText: t("apps.application.empty") }} pagination={false} rowKey="id" scroll={{ x: 820 }} /></div>
    </Card>
    <ApplicationDrawer editor={editor} form={form} fullScreen={!screens.md} saving={mutations.create.isPending || mutations.update.isPending} onClose={() => { setEditor(null); }} onSave={() => void save()} />
  </div>;
}

function ApplicationDrawer({ editor, form, fullScreen, saving, onClose, onSave }: { editor: Editor; form: ReturnType<typeof useForm<ApplicationInput>>; fullScreen: boolean; saving: boolean; onClose: () => void; onSave: () => void }) {
  const { t } = useTranslation();
  return <Drawer destroyOnHidden extra={<Button loading={saving} type="primary" onClick={onSave}>{t("common.actions.save")}</Button>} onClose={onClose} open={editor !== null} size={fullScreen ? "100%" : "large"} title={t(editor === "new" ? "apps.application.editor.create" : "apps.application.editor.edit")}>
    <Form layout="vertical">
      <Form.Item label={t("apps.application.fields.name")}><Controller control={form.control} name="name" render={({ field }) => <Input {...field} aria-label={t("apps.application.fields.name")} autoComplete="off" />} /></Form.Item>
      <Form.Item label={t("apps.application.fields.default_locale")}><Controller control={form.control} name="default_locale" render={({ field }) => <Select {...field} aria-label={t("apps.application.fields.default_locale")} options={["zh-CN", "en-US"].map((value) => ({ value, label: t(`apps.locale.${value}`) }))} />} /></Form.Item>
      <Form.Item label={t("apps.application.fields.registration_enabled")}><Controller control={form.control} name="registration_enabled" render={({ field }) => <Switch aria-label={t("apps.application.fields.registration_enabled")} checked={field.value} onChange={field.onChange} />} /></Form.Item>
      <Form.Item label={t("apps.application.fields.registration_verification_mode")}><Controller control={form.control} name="registration_verification_mode" render={({ field }) => <Select {...field} aria-label={t("apps.application.fields.registration_verification_mode")} options={["none", "email_otp"].map((value) => ({ value, label: t(`apps.registration.verification.${value}`) }))} />} /></Form.Item>
    </Form>
  </Drawer>;
}
