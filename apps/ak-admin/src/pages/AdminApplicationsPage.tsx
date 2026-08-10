import { DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import { useNavigate } from "@tanstack/react-router";
import { Alert, Button, Card, Divider, Drawer, Form, Grid, Input, InputNumber, Modal, Select, Space, Switch, Table, Tag, Typography, type TableColumnsType } from "antd";
import { Controller, useFieldArray, useForm, type FieldPath } from "react-hook-form";
import { useMemo, useState, type Key } from "react";
import { useTranslation } from "react-i18next";
import { AkFilePicker } from "../components/AkFilePicker";
import { useAuthStore } from "../features/auth/store";
import { useApplicationMutations, useManagedApplications } from "../features/apps/hooks";
import { applicationChannelCodes, applicationInputSchema, type ApplicationInput, type ManagedApplication } from "../features/apps/model";
import { ApiError } from "../shared/api/error";

type Editor = ManagedApplication | "new" | null;
type PickerTarget = { type: "icon" } | { type: "screenshot" } | { type: "qrcode"; index: number } | null;

function defaults(tenantId: string): ApplicationInput {
  return {
    appid: "", app_type: "uni_app_x", name: "", description: "", introduction: "", remark: "",
    default_locale: "zh-CN", registration_enabled: true, registration_verification_mode: "email_otp",
    owner_type: "tenant", owner_id: tenantId, icon_file_id: null, managers: [], members: [], screenshot_file_ids: [],
    channels: [], store_listings: [],
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
  };
}

function commaSeparated(value: string): string[] {
  return [...new Set(value.split(/[\s,]+/).map((part) => part.trim()).filter(Boolean))];
}

function conflictKey(error: unknown): string {
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
  const applicationActions = (item: ManagedApplication) => <Space size="small" wrap>
    {permissions.has("app.application.update") ? <Button size="small" onClick={() => { open(item); }}>{t("common.actions.edit")}</Button> : null}
    <Button size="small" onClick={() => void navigate({ to: "/app/upgrade-center", search: { app_id: item.id, q: "", package_type: "", platform: "", publish_status: "", page: 1, page_size: 20 } })}>{t("apps.application.actions.upgrade_center")}</Button>
    <Button size="small" onClick={() => void navigate({ to: "/app/content/articles", search: { app_id: item.id } })}>{t("apps.application.actions.content")}</Button>
    {permissions.has("app.application.disable") ? <Button size="small" danger={item.status === "active"} loading={mutations.status.isPending} onClick={() => void changeStatus(item)}>{t(item.status === "active" ? "apps.actions.disable" : "apps.actions.enable")}</Button> : null}
    {permissions.has("app.application.delete") && item.status === "disabled" && !item.is_default ? <Button danger icon={<DeleteOutlined />} size="small" onClick={() => { deleteItems([item.id]); }}>{t("common.actions.delete")}</Button> : null}
  </Space>;
  const columns: TableColumnsType<ManagedApplication> = [
    { title: t("apps.application.columns.appid"), dataIndex: "appid", width: 190, render: (value: string) => value ? <code className="ak-version-value">{value}</code> : <Tag className="ak-status-warning">{t("apps.application.values.appid_pending")}</Tag> },
    { title: t("apps.application.columns.type"), dataIndex: "app_type", width: 120, render: (value: ManagedApplication["app_type"]) => <Tag>{t(`apps.application.types.${value}`)}</Tag> },
    { title: t("apps.application.columns.name"), width: 220, render: (_, item) => <div><strong>{item.name}</strong><div className="ak-content-slug">{item.code}</div></div> },
    { title: t("apps.application.columns.description"), dataIndex: "description", ellipsis: true, responsive: ["md"], render: (value: string) => value || <span className="ak-table-secondary">{t("apps.application.values.none")}</span> },
    { title: t("apps.application.columns.remark"), dataIndex: "remark", ellipsis: true, responsive: ["lg"], render: (value: string) => value || <span className="ak-table-secondary">{t("apps.application.values.none")}</span> },
    { title: t("apps.application.columns.status"), dataIndex: "status", width: 110, render: (value: ManagedApplication["status"]) => <Tag className={value === "active" ? "ak-status-success" : "ak-status-error"}>{t(`apps.status.${value}`)}</Tag> },
    { title: t("apps.application.columns.created"), dataIndex: "created_at", width: 180, responsive: ["xl"], render: (value: string) => date.format(new Date(value)) },
    { title: t("apps.application.columns.actions"), ...(screens.lg ? { fixed: "right" as const } : {}), width: 300, render: (_, item) => applicationActions(item) },
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
      {screens.md ? <div aria-label={t("apps.application.title")} className="ak-table-scroll" role="region" tabIndex={0}><Table columns={columns} dataSource={applications.data?.items ?? []} loading={applications.isPending} locale={{ emptyText: t("apps.application.empty") }} pagination={{ current: page, pageSize, total: applications.data?.total ?? 0, showSizeChanger: true, onChange: (nextPage, nextPageSize) => { setPage(nextPage); setPageSize(nextPageSize); } }} rowKey="id" {...(permissions.has("app.application.delete") ? { rowSelection: { selectedRowKeys: selected, onChange: setSelected, getCheckboxProps: (item: ManagedApplication) => ({ disabled: item.status !== "disabled" || item.is_default }) } } : {})} scroll={{ x: 1500 }} size="middle" /></div> : <div className="ak-mobile-application-list">
        {applications.isPending ? <Card loading size="small" /> : null}
        {!applications.isPending && (applications.data?.items.length ?? 0) === 0 ? <div className="ak-mobile-release-empty">{t("apps.application.empty")}</div> : null}
        {(applications.data?.items ?? []).map((item) => <Card className="ak-mobile-application-card" key={item.id} size="small" title={item.name} extra={<Tag className={item.status === "active" ? "ak-status-success" : "ak-status-error"}>{t(`apps.status.${item.status}`)}</Tag>}>
          <Space orientation="vertical" size="small" className="ak-full-width"><code className="ak-version-value">{item.appid || t("apps.application.values.appid_pending")}</code><Tag>{t(`apps.application.types.${item.app_type}`)}</Tag>{item.description ? <Typography.Paragraph ellipsis={{ rows: 2 }}>{item.description}</Typography.Paragraph> : null}{applicationActions(item)}</Space>
        </Card>)}
      </div>}
    </Card>
    <ApplicationDrawer editor={editor} form={form} fullScreen={!screens.md} picker={picker} setPicker={setPicker} saving={mutations.create.isPending || mutations.update.isPending} onClose={() => { setEditor(null); }} onSave={() => void save()} />
  </div>;
}

function ApplicationDrawer({ editor, form, fullScreen, picker, setPicker, saving, onClose, onSave }: { editor: Editor; form: ReturnType<typeof useForm<ApplicationInput>>; fullScreen: boolean; picker: PickerTarget; setPicker: (value: PickerTarget) => void; saving: boolean; onClose: () => void; onSave: () => void }) {
  const { t } = useTranslation();
  const channels = useFieldArray({ control: form.control, name: "channels", keyName: "formKey" });
  const stores = useFieldArray({ control: form.control, name: "store_listings", keyName: "formKey" });
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
  return <>
    <Drawer destroyOnHidden extra={<Button loading={saving} type="primary" onClick={onSave}>{t("common.actions.save")}</Button>} onClose={onClose} open={editor !== null} size={fullScreen ? "100%" : "large"} title={t(editor === "new" ? "apps.application.editor.create" : "apps.application.editor.edit")}>
      <Form layout="vertical" className="ak-application-form">
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
          <div><Button onClick={() => { setPicker({ type: "icon" }); }}>{t("apps.application.actions.choose_icon")}</Button> {form.watch("icon_file_id") ? <code className="ak-content-slug">{form.watch("icon_file_id")}</code> : <Typography.Text type="secondary">{t("apps.application.values.none")}</Typography.Text>}</div>
          <div><Button onClick={() => { setPicker({ type: "screenshot" }); }}>{t("apps.application.actions.add_screenshot")}</Button></div>
          {screenshots.map((fileId, index) => <Space key={`${fileId}-${String(index)}`} wrap><Tag>{index + 1}</Tag><code className="ak-content-slug">{fileId}</code><Button disabled={index === 0} size="small" onClick={() => { moveScreenshot(index, -1); }}>{t("apps.application.actions.move_up")}</Button><Button disabled={index === screenshots.length - 1} size="small" onClick={() => { moveScreenshot(index, 1); }}>{t("apps.application.actions.move_down")}</Button><Button danger size="small" onClick={() => { form.setValue("screenshot_file_ids", screenshots.filter((_, current) => current !== index), { shouldDirty: true }); }}>{t("common.actions.delete")}</Button></Space>)}
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
      setPicker(null);
    }} />
  </>;
}
