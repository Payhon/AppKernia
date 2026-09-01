/* eslint-disable @typescript-eslint/restrict-template-expressions -- React Hook Form field paths preserve numeric array indices. */
import { AppPublicWebDrawer } from "../components/AppPublicWebDrawer";
import { MoreOutlined, PlusOutlined } from "@ant-design/icons";
import { useNavigate } from "@tanstack/react-router";
import { Alert, Button, Card, Divider, Drawer, Dropdown, Form, Grid, Input, InputNumber, Modal, Select, Space, Switch, Table, Tag, Typography, type TableColumnsType } from "antd";
import { Controller, useFieldArray, useForm, type FieldPath } from "react-hook-form";
import { useEffect, useMemo, useState, type Key } from "react";
import { useTranslation } from "react-i18next";
import { AkFilePicker } from "../components/AkFilePicker";
import { AkLocalizedFormTabs } from "../components/AkLocalizedFormTabs";
import { AppClientConfigurationModal } from "../components/AppClientConfigurationModal";
import { authSession, useAuthStore } from "../features/auth/store";
import { createApplicationActionItems } from "../features/apps/application-actions";
import { useApplicationMutations, useManagedApplications } from "../features/apps/hooks";
import { applicationChannelCodes, applicationInputSchema, type ApplicationInput, type ManagedApplication } from "../features/apps/model";
import { useSystemLanguages } from "../features/settings/system-languages";
import type { AdminLocale } from "../shared/i18n";
import { ApiError } from "../shared/api/error";

type Editor = ManagedApplication | "new" | null;
type PickerTarget = { type: "icon" } | { type: "screenshot" } | { type: "qrcode"; index: number } | { type: "onboarding"; index: number; locale: AdminLocale } | null;

function defaults(tenantId: string): ApplicationInput {
  return {
    appid: "", app_type: "uni_app_x", name: "", description: "", introduction: "", remark: "",
    default_locale: "zh-CN", registration_enabled: true, registration_verification_mode: "email_otp",
    owner_type: "tenant", owner_id: tenantId, icon_file_id: null, managers: [], members: [], screenshot_file_ids: [],
    channels: [], store_listings: [],
    startup: { translations: { "zh-CN": { display_name: "", subtitle: "" }, "en-US": { display_name: "", subtitle: "" } }, onboarding_enabled: false, draft_slides: [] },
  };
}

function fromItem(item: ManagedApplication): ApplicationInput {
  return {
    appid: item.appid, app_type: item.app_type, code: item.code, name: item.name, description: item.description,
    introduction: item.introduction, remark: item.remark, default_locale: item.default_locale,
    registration_enabled: item.registration_enabled, registration_verification_mode: item.registration_verification_mode,
    owner_type: item.owner_type, owner_id: item.owner_id ?? item.tenant_id, icon_file_id: item.icon_file_id ?? null,
    managers: item.managers, members: item.members, screenshot_file_ids: item.screenshots.map((asset) => asset.file_id),
    channels: item.channels.map((channel) => ({ ...channel, url: channel.url ?? null, abm_url: channel.abm_url ?? null, qrcode_file_id: channel.qrcode_file_id ?? null })),
    store_listings: item.store_listings, lock_version: item.lock_version,
    startup: { translations: item.startup.translations, onboarding_enabled: item.startup.onboarding_enabled, draft_slides: item.startup.draft_slides },
  };
}

function commaSeparated(value: string): string[] {
  return [...new Set(value.split(/[\s,]+/).map((part) => part.trim()).filter(Boolean))];
}

function conflictKey(error: unknown): string {
  if (error instanceof ApiError && error.code === "APP.STARTUP.ASSET_REJECTED") return "apps.startup.feedback.asset_rejected";
  return error instanceof ApiError && error.status === 409 ? "apps.feedback.conflict" : "apps.feedback.save_error";
}

type ChannelField = keyof ApplicationInput["channels"][number];
type StoreField = keyof ApplicationInput["store_listings"][number];
const channelPath = <K extends ChannelField>(index: number, field: K): `channels.${number}.${K}` => `channels.${String(index)}.${field}` as `channels.${number}.${K}`;
const storePath = <K extends StoreField>(index: number, field: K): `store_listings.${number}.${K}` => `store_listings.${String(index)}.${field}` as `store_listings.${number}.${K}`;

export function AdminApplicationsPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const screens = Grid.useBreakpoint();
  const context = useAuthStore((state) => state.context);
  const tenantId = context?.active_tenant.id ?? "";
  const permissions = new Set(context?.permissions ?? []);
  const [q, setQ] = useState("");
  const [status, setStatusFilter] = useState("");
  const [appType, setAppType] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selected, setSelected] = useState<Key[]>([]);
  const [editor, setEditor] = useState<Editor>(null);
  const [picker, setPicker] = useState<PickerTarget>(null);
  const [publicWebAppId, setPublicWebAppId] = useState<string | null>(null);
  const [clientConfigApp, setClientConfigApp] = useState<ManagedApplication | null>(null);
  const [feedback, setFeedback] = useState<{ key: string; error: boolean } | null>(null);
  const form = useForm<ApplicationInput>({ defaultValues: defaults(tenantId) });
  const applications = useManagedApplications({ q, status, app_type: appType, page, page_size: pageSize });
  const mutations = useApplicationMutations();
  const date = useMemo(() => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }), [i18n.language]);
  const open = (value: ManagedApplication | "new") => { form.reset(value === "new" ? defaults(tenantId) : fromItem(value)); setEditor(value); setFeedback(null); };
  const save = form.handleSubmit(async (values) => {
    form.clearErrors();
    const parsed = applicationInputSchema.safeParse(values);
    if (!parsed.success) {
      parsed.error.issues.forEach((issue) => { form.setError(issue.path.join(".") as FieldPath<ApplicationInput>, { message: issue.message }); });
      setFeedback({ key: "apps.feedback.validation_error", error: true });
      return;
    }
    try {
      if (editor === "new") await mutations.create.mutateAsync(parsed.data);
      else if (editor) await mutations.update.mutateAsync({ id: editor.id, input: parsed.data });
      setEditor(null); setFeedback({ key: "apps.feedback.saved", error: false });
    } catch (error) { setFeedback({ key: conflictKey(error), error: true }); }
  });
  const publishOnboarding = async () => {
    if (!editor || editor === "new") return;
    try {
      const updated = await mutations.publishOnboarding.mutateAsync({ id: editor.id, expectedPublishedVersion: editor.startup.published_version });
      form.reset(fromItem(updated)); setEditor(updated); setFeedback({ key: "apps.startup.feedback.published", error: false });
    } catch (error) { setFeedback({ key: conflictKey(error), error: true }); }
  };
  const changeStatus = async (item: ManagedApplication) => {
    try {
      await mutations.status.mutateAsync({ id: item.id, action: item.status === "active" ? "disable" : "enable", lockVersion: item.lock_version });
      setFeedback({ key: "apps.feedback.status_saved", error: false });
    } catch (error) { setFeedback({ key: conflictKey(error), error: true }); }
  };
  const deleteItems = (ids: string[]) => {
    Modal.confirm({
      title: t("apps.application.delete.title"), content: t("apps.application.delete.description", { count: ids.length }), okButtonProps: { danger: true }, okText: t("common.actions.delete"),
      onOk: async () => {
        try {
          if (ids.length === 1) await mutations.delete.mutateAsync(ids[0] ?? ""); else await mutations.batchDelete.mutateAsync(ids);
          setSelected([]); setFeedback({ key: "apps.feedback.deleted", error: false });
        } catch (error) { setFeedback({ key: conflictKey(error), error: true }); }
      },
    });
  };
  const applicationActions = (item: ManagedApplication) => <Dropdown
    menu={{ items: [...createApplicationActionItems(item, permissions, {
      edit: t("common.actions.edit"),
      upgradeCenter: t("apps.application.actions.upgrade_center"),
      content: t("apps.application.actions.content"),
      clientConfig: t("apps.client_config.actions.open"),
      enable: t("apps.actions.enable"),
      disable: t("apps.actions.disable"),
      delete: t("common.actions.delete"),
    }, {
      edit: () => { open(item); },
      upgradeCenter: () => { void navigate({ to: "/app/upgrade-center", search: { app_id: item.id, q: "", package_type: "", platform: "", publish_status: "", page: 1, page_size: 20 } }); },
      content: () => { void navigate({ to: "/app/content/articles", search: { app_id: item.id } }); },
      clientConfig: () => { setClientConfigApp(item); },
      changeStatus: () => { void changeStatus(item); },
      delete: () => { deleteItems([item.id]); },
    }, mutations.status.isPending), ...(permissions.has("app.public_web.read") ? [{ key: "public-web", label: t("apps.public_web.title"), onClick: () => { setPublicWebAppId(item.id); } }] : [])] }}
    trigger={["click"]}
  >
    <Button aria-label={t("apps.application.actions.menu_for", { name: item.name })} icon={<MoreOutlined />} size="small">{t("apps.application.actions.menu")}</Button>
  </Dropdown>;
  const columns: TableColumnsType<ManagedApplication> = [
    { title: t("apps.application.columns.appid"), dataIndex: "appid", width: 190, render: (value: string) => value ? <code className="ak-version-value">{value}</code> : <Tag className="ak-status-warning">{t("apps.application.values.appid_pending")}</Tag> },
    { title: t("apps.application.columns.type"), dataIndex: "app_type", width: 120, render: (value: ManagedApplication["app_type"]) => <Tag>{t(`apps.application.types.${value}`)}</Tag> },
    { title: t("apps.application.columns.name"), width: 220, render: (_, item) => <div><strong>{item.name}</strong><div className="ak-content-slug">{item.code}</div></div> },
    { title: t("apps.application.columns.description"), dataIndex: "description", ellipsis: true, responsive: ["md"], render: (value: string) => value || <span className="ak-table-secondary">{t("apps.application.values.none")}</span> },
    { title: t("apps.application.columns.remark"), dataIndex: "remark", ellipsis: true, responsive: ["lg"], render: (value: string) => value || <span className="ak-table-secondary">{t("apps.application.values.none")}</span> },
    { title: t("apps.application.columns.status"), dataIndex: "status", width: 110, render: (value: ManagedApplication["status"]) => <Tag className={value === "active" ? "ak-status-success" : "ak-status-error"}>{t(`apps.status.${value}`)}</Tag> },
    { title: t("apps.application.columns.created"), dataIndex: "created_at", width: 180, responsive: ["xl"], render: (value: string) => date.format(new Date(value)) },
    { title: t("apps.application.columns.actions"), ...(screens.lg ? { fixed: "right" as const } : {}), width: 112, render: (_, item) => applicationActions(item) },
  ];
  return <div className="ak-page-container">
    <header className="ak-page-heading"><div><Typography.Title level={1}>{t("apps.application.title")}</Typography.Title><Typography.Paragraph type="secondary">{t("apps.application.description")}</Typography.Paragraph></div><Space wrap>
      {permissions.has("app.application.delete") && selected.length > 0 ? <Button danger loading={mutations.batchDelete.isPending} onClick={() => { deleteItems(selected.map(String)); }}>{t("apps.application.actions.batch_delete", { count: selected.length })}</Button> : null}
      {permissions.has("app.application.create") ? <Button type="primary" onClick={() => { open("new"); }}>{t("apps.application.actions.create")}</Button> : null}
    </Space></header>
    {feedback ? <Alert showIcon type={feedback.error ? "error" : "success"} title={t(feedback.key)} /> : null}
    <Card>
      <div className="ak-app-list-filters" role="search" aria-label={t("apps.application.filters.landmark")}>
        <Input.Search allowClear aria-label={t("apps.application.filters.q")} onChange={(event) => { setQ(event.target.value); setPage(1); }} placeholder={t("apps.application.filters.q")} value={q} />
        <Select allowClear aria-label={t("apps.application.filters.type")} onChange={(value) => { setAppType(value ?? ""); setPage(1); }} options={["uni_app", "uni_app_x"].map((value) => ({ value, label: t(`apps.application.types.${value}`) }))} placeholder={t("apps.application.filters.type")} value={appType || undefined} />
        <Select allowClear aria-label={t("apps.application.filters.status")} onChange={(value) => { setStatusFilter(value ?? ""); setPage(1); }} options={["active", "disabled"].map((value) => ({ value, label: t(`apps.status.${value}`) }))} placeholder={t("apps.application.filters.status")} value={status || undefined} />
      </div>
      {applications.isError ? <Alert showIcon type="error" title={t("apps.feedback.load_error")} action={<Button onClick={() => void applications.refetch()}>{t("common.actions.retry")}</Button>} /> : null}
      {screens.md ? <div aria-label={t("apps.application.title")} className="ak-table-scroll" role="region" tabIndex={0}><Table columns={columns} dataSource={applications.data?.items ?? []} loading={applications.isPending} locale={{ emptyText: t("apps.application.empty") }} pagination={{ current: page, pageSize, total: applications.data?.total ?? 0, showSizeChanger: true, onChange: (nextPage, nextPageSize) => { setPage(nextPage); setPageSize(nextPageSize); } }} rowKey="id" {...(permissions.has("app.application.delete") ? { rowSelection: { selectedRowKeys: selected, onChange: setSelected, getCheckboxProps: (item: ManagedApplication) => ({ disabled: item.status !== "disabled" || item.is_default }) } } : {})} scroll={{ x: 1320 }} size="middle" /></div> : <div className="ak-mobile-application-list">
        {applications.isPending ? <Card loading size="small" /> : null}
        {!applications.isPending && (applications.data?.items.length ?? 0) === 0 ? <div className="ak-mobile-release-empty">{t("apps.application.empty")}</div> : null}
        {(applications.data?.items ?? []).map((item) => <Card className="ak-mobile-application-card" key={item.id} size="small" title={item.name} extra={<Tag className={item.status === "active" ? "ak-status-success" : "ak-status-error"}>{t(`apps.status.${item.status}`)}</Tag>}>
          <Space orientation="vertical" size="small" className="ak-full-width"><code className="ak-version-value">{item.appid || t("apps.application.values.appid_pending")}</code><Tag>{t(`apps.application.types.${item.app_type}`)}</Tag>{item.description ? <Typography.Paragraph ellipsis={{ rows: 2 }}>{item.description}</Typography.Paragraph> : null}{applicationActions(item)}</Space>
        </Card>)}
      </div>}
    </Card>
    <ApplicationDrawer canPublish={permissions.has("app.onboarding.publish")} editor={editor} form={form} fullScreen={!screens.md} picker={picker} setPicker={setPicker} publishing={mutations.publishOnboarding.isPending} saving={mutations.create.isPending || mutations.update.isPending} onClose={() => { setEditor(null); }} onPublish={() => { Modal.confirm({ title: t("apps.startup.publish.title"), content: t("apps.startup.publish.description"), okText: t("apps.startup.actions.publish"), onOk: publishOnboarding }); }} onSave={() => void save()} />
    {publicWebAppId ? <AppPublicWebDrawer key={publicWebAppId} appId={publicWebAppId} onClose={() => { setPublicWebAppId(null); }} /> : null}
    <AppClientConfigurationModal app={clientConfigApp} permissions={permissions} onClose={() => { setClientConfigApp(null); }} />
  </div>;
}

function ApplicationDrawer({ canPublish, editor, form, fullScreen, picker, setPicker, publishing, saving, onClose, onPublish, onSave }: { canPublish: boolean; editor: Editor; form: ReturnType<typeof useForm<ApplicationInput>>; fullScreen: boolean; picker: PickerTarget; setPicker: (value: PickerTarget) => void; publishing: boolean; saving: boolean; onClose: () => void; onPublish: () => void; onSave: () => void }) {
  const { t } = useTranslation();
  const [activeLocale, setActiveLocale] = useState<AdminLocale>("zh-CN");
  const languages = useSystemLanguages();
  const channels = useFieldArray({ control: form.control, name: "channels", keyName: "formKey" });
  const stores = useFieldArray({ control: form.control, name: "store_listings", keyName: "formKey" });
  const onboarding = useFieldArray({ control: form.control, name: "startup.draft_slides", keyName: "formKey" });
  const screenshots = form.watch("screenshot_file_ids");
  const moveScreenshot = (index: number, delta: number) => {
    const next = [...screenshots]; const target = index + delta;
    if (target < 0 || target >= next.length) return;
    const currentFile = next[index];
    const targetFile = next[target];
    if (currentFile === undefined || targetFile === undefined) return;
    next[index] = targetFile;
    next[target] = currentFile;
    form.setValue("screenshot_file_ids", next, { shouldDirty: true });
  };
  const drawerActions = <Space wrap>{editor && editor !== "new" && canPublish ? <Button disabled={onboarding.fields.length === 0 || !editor.startup.draft_changed} loading={publishing} onClick={onPublish}>{t("apps.startup.actions.publish")}</Button> : null}<Button loading={saving} type="primary" onClick={onSave}>{t("common.actions.save")}</Button></Space>;
  return <>
    <Drawer destroyOnHidden extra={fullScreen ? null : drawerActions} onClose={onClose} open={editor !== null} size={fullScreen ? "100%" : "large"} title={t(editor === "new" ? "apps.application.editor.create" : "apps.application.editor.edit")}>
      <Form layout="vertical" className="ak-application-form">
        {fullScreen ? <div className="ak-application-mobile-actions">{drawerActions}</div> : null}
        <Divider titlePlacement="start">{t("apps.application.sections.basic")}</Divider>
        <div className="ak-form-grid-2">
          <Form.Item label={t("apps.application.fields.appid")} {...(form.formState.errors.appid ? { validateStatus: "error" as const } : {})} help={form.formState.errors.appid ? t("apps.application.validation.appid") : t("apps.application.fields.appid_hint")}><Controller control={form.control} name="appid" render={({ field }) => <Input {...field} aria-label={t("apps.application.fields.appid")} disabled={editor !== "new" && editor?.appid_pending === false} placeholder="__UNI__APPKERNIA" />} /></Form.Item>
          <Form.Item label={t("apps.application.fields.app_type")}><Controller control={form.control} name="app_type" render={({ field }) => <Select {...field} aria-label={t("apps.application.fields.app_type")} disabled={editor !== "new"} options={["uni_app", "uni_app_x"].map((value) => ({ value, label: t(`apps.application.types.${value}`) }))} />} /></Form.Item>
          <Form.Item label={t("apps.application.fields.name")}><Controller control={form.control} name="name" render={({ field }) => <Input {...field} aria-label={t("apps.application.fields.name")} autoComplete="off" />} /></Form.Item>
          <Form.Item label={t("apps.application.fields.default_locale")}><Controller control={form.control} name="default_locale" render={({ field }) => <Select {...field} aria-label={t("apps.application.fields.default_locale")} options={["zh-CN", "en-US"].map((value) => ({ value, label: t(`apps.locale.${value}`) }))} />} /></Form.Item>
        </div>
        <Form.Item label={t("apps.application.fields.description")}><Controller control={form.control} name="description" render={({ field }) => <Input.TextArea {...field} aria-label={t("apps.application.fields.description")} rows={3} showCount maxLength={4000} />} /></Form.Item>
        <Form.Item label={t("apps.application.fields.introduction")}><Controller control={form.control} name="introduction" render={({ field }) => <Input.TextArea {...field} aria-label={t("apps.application.fields.introduction")} rows={3} showCount maxLength={1000} />} /></Form.Item>
        <Form.Item label={t("apps.application.fields.remark")}><Controller control={form.control} name="remark" render={({ field }) => <Input.TextArea {...field} aria-label={t("apps.application.fields.remark")} rows={3} showCount maxLength={4000} />} /></Form.Item>
        <div className="ak-form-grid-2"><Form.Item label={t("apps.application.fields.registration_enabled")}><Controller control={form.control} name="registration_enabled" render={({ field }) => <Switch aria-label={t("apps.application.fields.registration_enabled")} checked={field.value} onChange={field.onChange} />} /></Form.Item><Form.Item label={t("apps.application.fields.registration_verification_mode")}><Controller control={form.control} name="registration_verification_mode" render={({ field }) => <Select {...field} aria-label={t("apps.application.fields.registration_verification_mode")} options={["none", "email_otp"].map((value) => ({ value, label: t(`apps.registration.verification.${value}`) }))} />} /></Form.Item></div>

        <Divider titlePlacement="start">{t("apps.application.sections.assets")}</Divider>
        <Space orientation="vertical" size="middle" className="ak-full-width">
          <div><Button onClick={() => { setPicker({ type: "icon" }); }}>{t("apps.application.actions.choose_icon")}</Button> {form.watch("icon_file_id") ? <Space><PrivateFileThumbnail alt={t("apps.startup.icon_preview")} fileId={form.watch("icon_file_id") ?? ""} /><code className="ak-content-slug">{form.watch("icon_file_id")}</code></Space> : <Typography.Text type="secondary">{t("apps.application.values.none")}</Typography.Text>}</div>
          <div><Button onClick={() => { setPicker({ type: "screenshot" }); }}>{t("apps.application.actions.add_screenshot")}</Button></div>
          {screenshots.map((fileId, index) => <Space key={`${fileId}-${String(index)}`} wrap><Tag>{index + 1}</Tag><code className="ak-content-slug">{fileId}</code><Button disabled={index === 0} size="small" onClick={() => { moveScreenshot(index, -1); }}>{t("apps.application.actions.move_up")}</Button><Button disabled={index === screenshots.length - 1} size="small" onClick={() => { moveScreenshot(index, 1); }}>{t("apps.application.actions.move_down")}</Button><Button danger size="small" onClick={() => { form.setValue("screenshot_file_ids", screenshots.filter((_, current) => current !== index), { shouldDirty: true }); }}>{t("common.actions.delete")}</Button></Space>)}
        </Space>

        <Divider titlePlacement="start">{t("apps.application.sections.startup")}</Divider>
        <Alert showIcon type="info" title={t("apps.startup.bundle_export_hint")} />
        <AkLocalizedFormTabs activeLocale={activeLocale} errorLocales={{ "zh-CN": Boolean(form.formState.errors.startup?.translations?.["zh-CN"]), "en-US": Boolean(form.formState.errors.startup?.translations?.["en-US"]) }} languages={languages} onActiveLocaleChange={setActiveLocale} renderFields={(locale, label) => <div className="ak-form-grid-2"><Form.Item label={t("apps.startup.fields.display_name")}><Controller control={form.control} name={`startup.translations.${locale}.display_name`} render={({ field }) => <Input {...field} aria-label={`${label} ${t("apps.startup.fields.display_name")}`} maxLength={120} />} /></Form.Item><Form.Item label={t("apps.startup.fields.subtitle")}><Controller control={form.control} name={`startup.translations.${locale}.subtitle`} render={({ field }) => <Input {...field} aria-label={`${label} ${t("apps.startup.fields.subtitle")}`} maxLength={240} />} /></Form.Item></div>} />
        <div className="ak-startup-status-row"><Form.Item label={t("apps.startup.fields.enabled")}><Controller control={form.control} name="startup.onboarding_enabled" render={({ field }) => <Switch aria-label={t("apps.startup.fields.enabled")} checked={field.value} onChange={field.onChange} />} /></Form.Item>{editor && editor !== "new" ? <Space wrap><Tag>{t("apps.startup.published_version", { version: editor.startup.published_version })}</Tag><Tag className={editor.startup.draft_changed ? "ak-status-warning" : "ak-status-success"}>{t(editor.startup.draft_changed ? "apps.startup.draft_changed" : "apps.startup.draft_synced")}</Tag>{editor.startup.published_at ? <Typography.Text type="secondary">{new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(editor.startup.published_at))}</Typography.Text> : null}</Space> : null}</div>
        <Space orientation="vertical" className="ak-full-width" size="middle">
          {onboarding.fields.map((slide, index) => <Card className="ak-application-nested-card" key={slide.formKey} size="small" title={t("apps.startup.slide", { index: index + 1 })} extra={<Space><Button disabled={index === 0} size="small" onClick={() => { onboarding.swap(index, index - 1); }}>{t("apps.application.actions.move_up")}</Button><Button disabled={index === onboarding.fields.length - 1} size="small" onClick={() => { onboarding.swap(index, index + 1); }}>{t("apps.application.actions.move_down")}</Button><Button danger size="small" onClick={() => { onboarding.remove(index); }}>{t("common.actions.delete")}</Button></Space>}>
            {(["zh-CN", "en-US"] as const).map((locale) => { const fileId = form.watch(`startup.draft_slides.${index}.assets.${locale}.file_id`); return <div className="ak-startup-locale-asset" key={locale}><Typography.Text strong>{t(`apps.locale.${locale}`)}</Typography.Text><Space wrap><Button onClick={() => { setPicker({ type: "onboarding", index, locale }); }}>{t("apps.startup.actions.choose_image")}</Button>{fileId ? <><PrivateFileThumbnail alt={t("apps.startup.image_preview", { index: index + 1 })} fileId={fileId} /><code className="ak-content-slug">{fileId}</code></> : <Typography.Text type="danger">{t("apps.startup.image_required")}</Typography.Text>}</Space><Form.Item label={t("apps.startup.fields.accessibility_label")}><Controller control={form.control} name={`startup.draft_slides.${index}.assets.${locale}.accessibility_label`} render={({ field }) => <Input {...field} aria-label={`${t(`apps.locale.${locale}`)} ${t("apps.startup.fields.accessibility_label")}`} maxLength={500} />} /></Form.Item></div>; })}
          </Card>)}
          <Button block disabled={onboarding.fields.length >= 10} icon={<PlusOutlined />} onClick={() => { const position = onboarding.fields.length; onboarding.append({ position, assets: { "zh-CN": { file_id: "", accessibility_label: "" }, "en-US": { file_id: "", accessibility_label: "" } } }); }}>{t("apps.startup.actions.add_slide")}</Button>
        </Space>

        <Divider titlePlacement="start">{t("apps.application.sections.owner_team")}</Divider>
        <div className="ak-form-grid-2"><Form.Item label={t("apps.application.fields.owner_type")}><Controller control={form.control} name="owner_type" render={({ field }) => <Select {...field} aria-label={t("apps.application.fields.owner_type")} options={["tenant", "user"].map((value) => ({ value, label: t(`apps.application.owner_types.${value}`) }))} />} /></Form.Item><Form.Item label={t("apps.application.fields.owner_id")}><Controller control={form.control} name="owner_id" render={({ field }) => <Input {...field} aria-label={t("apps.application.fields.owner_id")} />} /></Form.Item></div>
        <Form.Item label={t("apps.application.fields.managers")} extra={t("apps.application.fields.uuid_list_hint")}><Controller control={form.control} name="managers" render={({ field }) => <Input.TextArea aria-label={t("apps.application.fields.managers")} rows={2} value={field.value.join(", ")} onChange={(event) => { field.onChange(commaSeparated(event.target.value)); }} />} /></Form.Item>
        <Form.Item label={t("apps.application.fields.members")} extra={t("apps.application.fields.uuid_list_hint")}><Controller control={form.control} name="members" render={({ field }) => <Input.TextArea aria-label={t("apps.application.fields.members")} rows={2} value={field.value.join(", ")} onChange={(event) => { field.onChange(commaSeparated(event.target.value)); }} />} /></Form.Item>

        <Divider titlePlacement="start">{t("apps.application.sections.channels")}</Divider>
        {channels.fields.map((channel, index) => <Card className="ak-application-nested-card" key={channel.formKey} size="small" title={t("apps.application.channel_card", { index: index + 1 })} extra={<Button danger size="small" onClick={() => { channels.remove(index); }}>{t("common.actions.delete")}</Button>}>
          <div className="ak-form-grid-2"><Form.Item label={t("apps.application.fields.channel_code")}><Controller control={form.control} name={channelPath(index, "channel_code")} render={({ field }) => <Select {...field} aria-label={`${t("apps.application.fields.channel_code")} ${String(index + 1)}`} options={applicationChannelCodes.map((value) => ({ value, label: t(`apps.application.channels.${value}`) }))} />} /></Form.Item><Form.Item label={t("apps.application.fields.channel_name")}><Controller control={form.control} name={channelPath(index, "name")} render={({ field }) => <Input {...field} aria-label={`${t("apps.application.fields.channel_name")} ${String(index + 1)}`} />} /></Form.Item></div>
          <Form.Item label={t("apps.application.fields.channel_url")}><Controller control={form.control} name={channelPath(index, "url")} render={({ field }) => <Input {...field} aria-label={`${t("apps.application.fields.channel_url")} ${String(index + 1)}`} value={field.value ?? ""} onChange={(event) => { field.onChange(event.target.value || null); }} placeholder="https://" />} /></Form.Item>
          <Form.Item label={t("apps.application.fields.abm_url")}><Controller control={form.control} name={channelPath(index, "abm_url")} render={({ field }) => <Input {...field} aria-label={`${t("apps.application.fields.abm_url")} ${String(index + 1)}`} value={field.value ?? ""} onChange={(event) => { field.onChange(event.target.value || null); }} placeholder="https://" />} /></Form.Item>
          <Space wrap><Controller control={form.control} name={channelPath(index, "enabled")} render={({ field }) => <Switch aria-label={`${t("apps.application.fields.channel_code")} ${String(index + 1)} ${t("common.states.enabled")}`} checked={field.value} checkedChildren={t("common.states.enabled")} unCheckedChildren={t("common.states.disabled")} onChange={field.onChange} />} /><Button onClick={() => { setPicker({ type: "qrcode", index }); }}>{t("apps.application.actions.choose_qrcode")}</Button><code className="ak-content-slug">{form.watch(channelPath(index, "qrcode_file_id"))}</code></Space>
        </Card>)}
        <Button block icon={<PlusOutlined />} onClick={() => { channels.append({ channel_code: "android", name: "", url: null, abm_url: null, qrcode_file_id: null, enabled: true }); }}>{t("apps.application.actions.add_channel")}</Button>

        <Divider titlePlacement="start">{t("apps.application.sections.stores")}</Divider>
        {stores.fields.map((store, index) => <Card className="ak-application-nested-card" key={store.formKey} size="small" title={t("apps.application.store_card", { index: index + 1 })} extra={<Button danger size="small" onClick={() => { stores.remove(index); }}>{t("common.actions.delete")}</Button>}><div className="ak-form-grid-2"><Form.Item label={t("apps.application.fields.store_name")}><Controller control={form.control} name={storePath(index, "name")} render={({ field }) => <Input {...field} aria-label={`${t("apps.application.fields.store_name")} ${String(index + 1)}`} />} /></Form.Item><Form.Item label={t("apps.application.fields.store_scheme")}><Controller control={form.control} name={storePath(index, "scheme")} render={({ field }) => <Input {...field} aria-label={`${t("apps.application.fields.store_scheme")} ${String(index + 1)}`} placeholder="appkernia://" />} /></Form.Item><Form.Item label={t("apps.application.fields.store_priority")}><Controller control={form.control} name={storePath(index, "priority")} render={({ field }) => <InputNumber aria-label={`${t("apps.application.fields.store_priority")} ${String(index + 1)}`} className="ak-full-width" min={-100000} max={100000} value={field.value} onChange={(value) => { field.onChange(value ?? 0); }} />} /></Form.Item><Form.Item label={t("apps.application.fields.store_enabled")}><Controller control={form.control} name={storePath(index, "enabled")} render={({ field }) => <Switch aria-label={`${t("apps.application.fields.store_enabled")} ${String(index + 1)}`} checked={field.value} onChange={field.onChange} />} /></Form.Item></div></Card>)}
        <Button block icon={<PlusOutlined />} onClick={() => { stores.append({ name: "", scheme: "", enabled: true, priority: 0 }); }}>{t("apps.application.actions.add_store")}</Button>
      </Form>
    </Drawer>
    <AkFilePicker open={picker !== null} onClose={() => { setPicker(null); }} onSelect={(file) => {
      if (picker?.type === "icon") form.setValue("icon_file_id", file.id, { shouldDirty: true });
      if (picker?.type === "screenshot") form.setValue("screenshot_file_ids", [...screenshots, file.id], { shouldDirty: true });
      if (picker?.type === "qrcode") form.setValue(channelPath(picker.index, "qrcode_file_id"), file.id, { shouldDirty: true });
      if (picker?.type === "onboarding") form.setValue(`startup.draft_slides.${picker.index}.assets.${picker.locale}.file_id`, file.id, { shouldDirty: true });
      setPicker(null);
    }} />
  </>;
}

function PrivateFileThumbnail({ fileId, alt }: { fileId: string; alt: string }) {
  const [source, setSource] = useState<string | null>(null);
  useEffect(() => {
    if (!fileId) return;
    let live = true; let objectUrl = "";
    void authSession.adminRequest(`/files/${encodeURIComponent(fileId)}/content`).then(async (response) => {
      if (!response.ok) return;
      objectUrl = URL.createObjectURL(await response.blob());
      if (live) setSource(objectUrl);
    });
    return () => { live = false; if (objectUrl) URL.revokeObjectURL(objectUrl); };
  }, [fileId]);
  return source ? <img alt={alt} className="ak-startup-thumbnail" src={source} /> : <span aria-label={alt} className="ak-startup-thumbnail-placeholder" />;
}
