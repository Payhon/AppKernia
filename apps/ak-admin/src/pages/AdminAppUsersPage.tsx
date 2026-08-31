import { Alert, Avatar, Button, Card, Drawer, Form, Grid, Input, Modal, Select, Space, Table, Tag, Typography, type TableColumnsType } from "antd";
import { Controller, useForm } from "react-hook-form";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AppSelectionRequiredState } from "../components/AppSelectionRequiredState";
import { useAuthStore } from "../features/auth/store";
import { useAppMemberAvatar, useAppMembers, useApplicationMutations } from "../features/apps/hooks";
import { appMemberCreateInputSchema, appMemberPasswordResetSchema, appMemberUpdateInputSchema, type AppMember, type AppMemberCreateInput, type AppMemberPasswordResetInput, type AppMemberUpdateInput } from "../features/apps/model";
import { AppScopeContext, type AppScope } from "../features/apps/scope";

type Editor = AppMember | "new" | null;
const defaults = (): AppMemberCreateInput => ({ email: "", display_name: "", locale: "zh-CN", password: "" });
const fromMember = (member: AppMember): AppMemberUpdateInput => ({ display_name: member.display_name, lock_version: member.lock_version });

export function AdminAppUsersPage() { return <AppScopeContext>{(scope) => <AppUsersContents scope={scope} />}</AppScopeContext>; }

function AppUsersContents({ scope }: { scope: AppScope }) {
  const { t, i18n } = useTranslation();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("");
  const [editor, setEditor] = useState<Editor>(null);
  const [resetTarget, setResetTarget] = useState<AppMember | null>(null);
  const [feedback, setFeedback] = useState<{ key: string; error: boolean } | null>(null);
  const form = useForm<AppMemberCreateInput & Partial<AppMemberUpdateInput>>({ defaultValues: defaults() });
  const passwordResetForm = useForm<AppMemberPasswordResetInput>({ defaultValues: { new_password: "", confirm_password: "" } });
  const members = useAppMembers(scope.appId, { q, status, page: 1, page_size: 100 });
  const mutations = useApplicationMutations();
  const date = useMemo(() => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }), [i18n.language]);
  const open = (value: AppMember | "new") => { form.reset(value === "new" ? defaults() : fromMember(value)); setEditor(value); };
  const save = form.handleSubmit(async (values) => {
    if (!scope.appId) return;
    const parsed = editor === "new" ? appMemberCreateInputSchema.safeParse(values) : appMemberUpdateInputSchema.safeParse({ display_name: values.display_name, lock_version: editor?.lock_version });
    if (!parsed.success) { setFeedback({ key: "apps.feedback.validation_error", error: true }); return; }
    try { if (editor === "new") await mutations.createMember.mutateAsync({ appId: scope.appId, input: parsed.data as AppMemberCreateInput }); else if (editor) await mutations.updateMember.mutateAsync({ appId: scope.appId, memberId: editor.id, input: parsed.data as AppMemberUpdateInput }); setEditor(null); setFeedback({ key: "apps.feedback.saved", error: false }); } catch { setFeedback({ key: "apps.feedback.save_error", error: true }); }
  });
  const action = (member: AppMember, actionName: "enable" | "disable" | "unlock" | "revoke-sessions") => {
    const appId = scope.appId;
    if (!appId) return;
    Modal.confirm({ title: t(`apps.users.confirm.${actionName}.title`), content: t(`apps.users.confirm.${actionName}.description`, { name: member.display_name }), okText: t(`apps.users.actions.${actionName}`), cancelText: t("common.actions.cancel"), okButtonProps: { danger: actionName === "disable" || actionName === "revoke-sessions" }, onOk: async () => { try { await mutations.memberAction.mutateAsync({ appId, memberId: member.id, action: actionName, lockVersion: member.lock_version }); setFeedback({ key: "apps.feedback.status_saved", error: false }); } catch { setFeedback({ key: "apps.feedback.save_error", error: true }); } } });
  };
  const resetPassword = passwordResetForm.handleSubmit(async (values) => {
    if (!scope.appId || !resetTarget) return;
    const parsed = appMemberPasswordResetSchema.safeParse(values);
    if (!parsed.success) { setFeedback({ key: "apps.feedback.validation_error", error: true }); return; }
    try { await mutations.resetMemberPassword.mutateAsync({ appId: scope.appId, memberId: resetTarget.id, newPassword: parsed.data.new_password, lockVersion: resetTarget.lock_version }); passwordResetForm.reset(); setResetTarget(null); setFeedback({ key: "apps.users.feedback.password_reset", error: false }); } catch { setFeedback({ key: "apps.feedback.save_error", error: true }); }
  });
  const columns: TableColumnsType<AppMember> = [
    { title: t("apps.users.columns.user"), render: (_, item) => <AppMemberIdentity member={item} /> },
    { title: t("apps.users.columns.source"), dataIndex: "source", responsive: ["md"], render: (value: AppMember["source"]) => t(`apps.users.source.${value}`) },
    { title: t("apps.users.columns.status"), dataIndex: "status", render: (value: AppMember["status"]) => <Tag className={value === "active" ? "ak-status-success" : value === "disabled" ? "ak-status-error" : "ak-status-warning"}>{t(`apps.membership.${value}`)}</Tag> },
    { title: t("apps.users.columns.last_sign_in"), dataIndex: "last_sign_in_at", responsive: ["lg"], render: (value: string | null) => value ? date.format(new Date(value)) : t("apps.values.none") },
    { title: t("apps.users.columns.actions"), width: screens.md ? 360 : 120, render: (_, item) => <Space wrap>{permissions.has("app.user.update") ? <Button size="small" onClick={() => { open(item); }}>{t("common.actions.edit")}</Button> : null}{permissions.has(item.status === "active" ? "app.user.disable" : "app.user.enable") ? <Button size="small" danger={item.status === "active"} onClick={() => { action(item, item.status === "active" ? "disable" : "enable"); }}>{t(item.status === "active" ? "apps.actions.disable" : "apps.actions.enable")}</Button> : null}{permissions.has("app.user.unlock") ? <Button size="small" onClick={() => { action(item, "unlock"); }}>{t("apps.users.actions.unlock")}</Button> : null}{permissions.has("app.user.reset_password") ? <Button size="small" onClick={() => { passwordResetForm.reset(); setResetTarget(item); }}>{t("apps.users.actions.reset-password")}</Button> : null}{permissions.has("app.user.revoke_session") ? <Button danger size="small" onClick={() => { action(item, "revoke-sessions"); }}>{t("apps.users.actions.revoke-sessions")}</Button> : null}</Space> },
  ];
  return <div className="ak-page-container">
    <header className="ak-page-heading"><div><Typography.Title level={1}>{t("apps.users.title")}</Typography.Title><Typography.Paragraph type="secondary">{t("apps.users.description")}</Typography.Paragraph></div>{scope.appId && !scope.disabled && permissions.has("app.user.create") ? <Button type="primary" onClick={() => { open("new"); }}>{t("apps.users.actions.create")}</Button> : null}</header>
    {feedback ? <Alert showIcon type={feedback.error ? "error" : "success"} title={t(feedback.key)} /> : null}
    <Card>{scope.appId ? <><div className="ak-content-filters" role="search" aria-label={t("apps.users.filters.landmark")}><Input.Search allowClear aria-label={t("apps.users.filters.q")} onChange={(event) => { setQ(event.target.value); }} placeholder={t("apps.users.filters.q")} value={q} /><Select allowClear aria-label={t("apps.users.filters.status")} onChange={(value) => { setStatus(value ?? ""); }} options={["pending_verification", "active", "disabled"].map((value) => ({ value, label: t(`apps.membership.${value}`) }))} placeholder={t("apps.users.filters.status")} value={status === "" ? undefined : status} /></div>
      {members.isError ? <Alert showIcon type="error" title={t("apps.feedback.load_error")} action={<Button onClick={() => void members.refetch()}>{t("common.actions.retry")}</Button>} /> : null}
      <div className="ak-table-scroll"><Table columns={columns} dataSource={members.data?.items ?? []} loading={members.isPending} locale={{ emptyText: t("apps.users.empty") }} pagination={false} rowKey="id" scroll={{ x: 1000 }} /></div></> : <AppSelectionRequiredState />}
    </Card>
    <AppUserDrawer editor={editor} form={form} fullScreen={!screens.md} saving={mutations.createMember.isPending || mutations.updateMember.isPending} onClose={() => { setEditor(null); }} onSave={() => void save()} />
    <PasswordResetModal form={passwordResetForm} loading={mutations.resetMemberPassword.isPending} member={resetTarget} onCancel={() => { passwordResetForm.reset(); setResetTarget(null); }} onSave={() => void resetPassword()} />
  </div>;
}

function AppMemberIdentity({ member, size = 40 }: { member: AppMember; size?: number }) {
  const { t } = useTranslation();
  const avatar = useAppMemberAvatar(member.avatar_url);
  const [source, setSource] = useState<string | null>(null);
  useEffect(() => {
    if (!avatar.data) { setSource(null); return; }
    const next = URL.createObjectURL(avatar.data); setSource(next);
    return () => { URL.revokeObjectURL(next); };
  }, [avatar.data]);
  return <Space align="center"><Avatar alt={t("apps.users.avatar_alt", { name: member.display_name })} size={size} src={source ?? undefined}>{member.display_name.slice(0, 1).toUpperCase()}</Avatar><div><strong>{member.display_name}</strong><div className="ak-content-slug">{member.email}</div></div></Space>;
}

function AppUserDrawer({ editor, form, fullScreen, saving, onClose, onSave }: { editor: Editor; form: ReturnType<typeof useForm<AppMemberCreateInput & Partial<AppMemberUpdateInput>>>; fullScreen: boolean; saving: boolean; onClose: () => void; onSave: () => void }) {
  const { t } = useTranslation();
  return <Drawer destroyOnHidden extra={<Button loading={saving} type="primary" onClick={onSave}>{t("common.actions.save")}</Button>} onClose={onClose} open={editor !== null} size={fullScreen ? "100%" : "large"} title={t(editor === "new" ? "apps.users.editor.create" : "apps.users.editor.edit")}>
    {editor && editor !== "new" ? <Card size="small" style={{ marginBottom: 16 }}><AppMemberIdentity member={editor} size={48} /><Typography.Paragraph style={{ marginBottom: 0, marginTop: 8 }} type="secondary">{t("apps.users.avatar_self_managed")}</Typography.Paragraph></Card> : null}
    <Form layout="vertical"><Form.Item label={t("apps.users.fields.email")}><Controller control={form.control} name="email" render={({ field }) => <Input {...field} aria-label={t("apps.users.fields.email")} autoComplete="email" disabled={editor !== "new"} inputMode="email" />} /></Form.Item><Form.Item label={t("apps.users.fields.display_name")}><Controller control={form.control} name="display_name" render={({ field }) => <Input {...field} aria-label={t("apps.users.fields.display_name")} autoComplete="name" />} /></Form.Item>{editor === "new" ? <><Form.Item label={t("apps.users.fields.locale")}><Controller control={form.control} name="locale" render={({ field }) => <Select {...field} aria-label={t("apps.users.fields.locale")} options={["zh-CN", "en-US"].map((value) => ({ value, label: t(`apps.locale.${value}`) }))} />} /></Form.Item><Form.Item label={t("apps.users.fields.password")} extra={t("apps.users.fields.password_hint")}><Controller control={form.control} name="password" render={({ field }) => <Input.Password {...field} aria-label={t("apps.users.fields.password")} autoComplete="new-password" />} /></Form.Item></> : null}</Form>
  </Drawer>;
}

function PasswordResetModal({ member, form, loading, onCancel, onSave }: { member: AppMember | null; form: ReturnType<typeof useForm<AppMemberPasswordResetInput>>; loading: boolean; onCancel: () => void; onSave: () => void }) {
  const { t } = useTranslation();
  return <Modal confirmLoading={loading} okText={t("apps.users.actions.reset-password")} onCancel={onCancel} onOk={onSave} open={member !== null} title={t("apps.users.password_reset.title")}><Typography.Paragraph type="secondary">{t("apps.users.password_reset.description", { name: member?.display_name ?? "" })}</Typography.Paragraph><Form layout="vertical"><Form.Item label={t("apps.users.password_reset.new_password")} extra={t("apps.users.fields.password_hint")}><Controller control={form.control} name="new_password" render={({ field }) => <Input.Password {...field} aria-label={t("apps.users.password_reset.new_password")} autoComplete="new-password" />} /></Form.Item><Form.Item label={t("apps.users.password_reset.confirm_password")}><Controller control={form.control} name="confirm_password" render={({ field }) => <Input.Password {...field} aria-label={t("apps.users.password_reset.confirm_password")} autoComplete="new-password" />} /></Form.Item></Form></Modal>;
}
