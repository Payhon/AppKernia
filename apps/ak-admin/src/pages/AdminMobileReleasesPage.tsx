import { DeleteOutlined, DownOutlined } from "@ant-design/icons";
import { useNavigate, useRouterState } from "@tanstack/react-router";
import { Alert, Button, Card, Checkbox, Drawer, Dropdown, Form, Grid, Input, Modal, Radio, Select, Space, Switch, Table, Tag, Typography, type MenuProps, type TableColumnsType } from "antd";
import { Controller, useForm, type FieldPath } from "react-hook-form";
import { useEffect, useMemo, useState, type Key } from "react";
import { useTranslation } from "react-i18next";
import { AkFilePicker } from "../components/AkFilePicker";
import { AkLocalizedFormTabs } from "../components/AkLocalizedFormTabs";
import { AppSelectionRequiredState } from "../components/AppSelectionRequiredState";
import { useAppScope } from "../features/apps/scope";
import { useManagedApplications } from "../features/apps/hooks";
import { useAuthStore } from "../features/auth/store";
import { useMobileReleaseMutations, useMobileReleases } from "../features/mobile-releases/hooks";
import { defaultMobileReleaseInput, mobileReleaseInputSchema, mobileReleasePackageTypes, mobileReleasePlatforms, mobileReleasePublishStatuses, releaseInput, type ManagedMobileRelease, type MobileReleaseFilters, type MobileReleaseInput } from "../features/mobile-releases/model";
import { findFirstInvalidLanguage, useSystemLanguages, type SystemLanguagesState } from "../features/settings/system-languages";
import { ApiError } from "../shared/api/error";
import type { AdminLocale } from "../shared/i18n";

type Editor = ManagedMobileRelease | { kind: "new"; packageType: "native_app" | "wgt" } | null;

function filtersFromURL(): MobileReleaseFilters {
  const params = new URLSearchParams(location.search);
  const page = Number(params.get("page")); const pageSize = Number(params.get("page_size"));
  return { q: params.get("q") ?? "", package_type: params.get("package_type") ?? "", platform: params.get("platform") ?? "", publish_status: params.get("publish_status") ?? "", page: Number.isInteger(page) && page > 0 ? page : 1, page_size: Number.isInteger(pageSize) && pageSize > 0 && pageSize <= 100 ? pageSize : 20 };
}

function errorKey(error: unknown): string {
  if (!(error instanceof ApiError)) return "mobile_releases.feedback.save_error";
  if (error.code === "SYS.MOBILE_RELEASE.VERSION_NOT_INCREASING") return "mobile_releases.feedback.version_conflict";
  if (error.code === "SYS.MOBILE_RELEASE.FROZEN") return "mobile_releases.feedback.frozen";
  return error.status === 409 ? "mobile_releases.feedback.conflict" : "mobile_releases.feedback.save_error";
}

export function AdminMobileReleasesPage() {
  return <MobileReleaseCenter />;
}

function MobileReleaseCenter() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate(); const screens = Grid.useBreakpoint();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const scope = useAppScope();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [filters, setFilters] = useState<MobileReleaseFilters>(filtersFromURL);
  const [editor, setEditor] = useState<Editor>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [selected, setSelected] = useState<Key[]>([]);
  const [feedback, setFeedback] = useState<{ key: string; error: boolean } | null>(null);
  const [activeLocale, setActiveLocale] = useState<AdminLocale>("zh-CN");
  const languages = useSystemLanguages();
  const query = useMobileReleases(scope.appId, filters);
  const mutations = useMobileReleaseMutations(scope.appId);
  const apps = useManagedApplications({ q: "", page: 1, page_size: 100 });
  const selectedApp = apps.data?.items.find((app) => app.id === scope.appId);
  const form = useForm<MobileReleaseInput>({ defaultValues: defaultMobileReleaseInput() });
  const date = useMemo(() => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }), [i18n.language]);
  const isUpgradeCenter = pathname === "/app/upgrade-center";

  useEffect(() => {
    void navigate({ to: pathname as never, search: { app_id: scope.appId ?? undefined, ...filters } as never, replace: true });
  }, [filters, navigate, pathname, scope.appId]);

  const updateFilter = (patch: Partial<MobileReleaseFilters>) => { setFilters((current) => ({ ...current, ...patch, page: patch.page ?? 1 })); };
  const openNew = (packageType: "native_app" | "wgt") => { form.reset(defaultMobileReleaseInput(packageType)); setActiveLocale(languages.preferredLocale); setFeedback(null); setEditor({ kind: "new", packageType }); };
  const openExisting = (item: ManagedMobileRelease) => { form.reset(releaseInput(item)); setActiveLocale(languages.preferredLocale); setFeedback(null); setEditor(item); };
  const isNew = editor !== null && "kind" in editor;
  const frozen = editor !== null && !("kind" in editor) && editor.ever_published_at != null;
  const save = form.handleSubmit(async (values) => {
    form.clearErrors();
    const parsed = mobileReleaseInputSchema.safeParse(values);
    if (!parsed.success) {
      parsed.error.issues.forEach((issue) => { form.setError(issue.path.join(".") as FieldPath<MobileReleaseInput>, { message: issue.message }); });
      const invalidLocale = findFirstInvalidLanguage(languages.languages, parsed.error.issues);
      if (invalidLocale) setActiveLocale(invalidLocale);
      setFeedback({ key: "mobile_releases.feedback.validation_error", error: true }); return;
    }
    try {
      if (isNew) await mutations.create.mutateAsync(parsed.data);
      else if (editor && !frozen) await mutations.update.mutateAsync({ id: editor.id, input: parsed.data });
      setEditor(null); setFeedback({ key: "mobile_releases.feedback.saved", error: false });
    } catch (error) { setFeedback({ key: errorKey(error), error: true }); }
  });
  const lifecycle = async (item: ManagedMobileRelease, action: "publish" | "unpublish") => {
    try {
      if (action === "publish") await mutations.publish.mutateAsync({ id: item.id, lockVersion: item.lock_version });
      else await mutations.unpublish.mutateAsync({ id: item.id, lockVersion: item.lock_version });
      setFeedback({ key: action === "publish" ? "mobile_releases.feedback.published" : "mobile_releases.feedback.unpublished", error: false });
    } catch (error) { setFeedback({ key: errorKey(error), error: true }); }
  };
  const deleteItems = (ids: string[]) => Modal.confirm({ title: t("mobile_releases.delete.title"), content: t("mobile_releases.delete.description", { count: ids.length }), okText: t("common.actions.delete"), okButtonProps: { danger: true }, onOk: async () => {
    try { if (ids.length === 1) await mutations.delete.mutateAsync(ids[0] ?? ""); else await mutations.batchDelete.mutateAsync(ids); setSelected([]); setFeedback({ key: "mobile_releases.feedback.deleted", error: false }); }
    catch (error) { setFeedback({ key: errorKey(error), error: true }); }
  } });
  const menu: MenuProps["items"] = [{ key: "native_app", label: t("mobile_releases.package_type.native_app") }, { key: "wgt", label: t("mobile_releases.package_type.wgt") }];
  const releaseActions = (item: ManagedMobileRelease) => <Space wrap size="small">
    <Button size="small" onClick={() => { openExisting(item); }}>{t(item.publish_status === "draft" ? "common.actions.edit" : "common.actions.view")}</Button>
    {item.publish_status === "draft" && permissions.has("mobile.release.publish") ? <Button size="small" type="primary" onClick={() => void lifecycle(item, "publish")}>{t("mobile_releases.actions.publish")}</Button> : null}
    {["online", "partial"].includes(item.publish_status) && permissions.has("mobile.release.publish") ? <Button size="small" onClick={() => void lifecycle(item, "unpublish")}>{t("mobile_releases.actions.unpublish")}</Button> : null}
    {item.publish_status === "offline" && permissions.has("mobile.release.publish") ? <Button size="small" onClick={() => void lifecycle(item, "publish")}>{t("mobile_releases.actions.republish")}</Button> : null}
    {item.publish_status === "draft" && permissions.has("mobile.release.delete") ? <Button danger icon={<DeleteOutlined />} size="small" onClick={() => deleteItems([item.id])}>{t("common.actions.delete")}</Button> : null}
  </Space>;
  const columns: TableColumnsType<ManagedMobileRelease> = [
    { title: t("mobile_releases.columns.version"), dataIndex: "version", width: 130, render: (value: string) => <code className="ak-version-value">{value}</code> },
    { title: t("mobile_releases.columns.package_type"), dataIndex: "package_type", width: 150, render: (value: string) => <Tag>{t(`mobile_releases.package_type.${value}`)}</Tag> },
    { title: t("mobile_releases.columns.platform"), dataIndex: "platforms", width: 220, render: (values: string[]) => <Space size={[0, 4]} wrap>{values.map((value) => <Tag key={value}>{t(`mobile_releases.platform.${value}`)}</Tag>)}</Space> },
    { title: t("mobile_releases.columns.title"), responsive: ["md"], render: (_, item) => item.titles[i18n.language === "en-US" ? "en-US" : "zh-CN"] },
    { title: t("mobile_releases.columns.status"), dataIndex: "publish_status", width: 130, render: (value: string) => <Tag className={value === "online" ? "ak-status-success" : value === "partial" ? "ak-status-warning" : value === "offline" ? "ak-status-error" : ""}>{t(`mobile_releases.publish_status.${value}`)}</Tag> },
    { title: t("mobile_releases.columns.published_platforms"), dataIndex: "published_platforms", width: 190, responsive: ["lg"], render: (values: string[]) => values.length ? values.map((value) => t(`mobile_releases.platform.${value}`)).join(" / ") : t("mobile_releases.values.none") },
    { title: t("mobile_releases.columns.created"), dataIndex: "created_at", width: 180, responsive: ["xl"], render: (value: string) => date.format(new Date(value)) },
    { title: t("mobile_releases.columns.actions"), ...(screens.lg ? { fixed: "right" as const } : {}), width: 230, render: (_, item) => releaseActions(item) },
  ];

  return <div className="ak-page-container">
    <header className="ak-page-heading"><div><Typography.Title level={1}>{t(isUpgradeCenter ? "mobile_releases.upgrade_center.title" : "mobile_releases.title")}</Typography.Title><Typography.Paragraph type="secondary">{t(isUpgradeCenter ? "mobile_releases.upgrade_center.description" : "mobile_releases.description")}</Typography.Paragraph></div><Space wrap>
      {selected.length > 0 && permissions.has("mobile.release.delete") ? <Button danger onClick={() => deleteItems(selected.map(String))}>{t("mobile_releases.actions.batch_delete", { count: selected.length })}</Button> : null}
      {scope.appId && permissions.has("mobile.release.create") ? <Dropdown menu={{ items: menu, onClick: ({ key }) => { openNew(key as "native_app" | "wgt"); } }}><Button disabled={Boolean(selectedApp?.appid_pending)} type="primary">{t("mobile_releases.actions.release_new")} <DownOutlined /></Button></Dropdown> : null}
    </Space></header>
    {selectedApp?.appid_pending ? <Alert showIcon type="warning" title={t("mobile_releases.states.appid_pending")} /> : null}
    {feedback ? <Alert showIcon type={feedback.error ? "error" : "success"} title={t(feedback.key)} /> : null}
    <Card>{scope.appId ? <>
      <div className="ak-release-filters" role="search" aria-label={t("mobile_releases.filters.landmark")}>
        <Input.Search allowClear value={filters.q} placeholder={t("mobile_releases.filters.query")} onChange={(event) => { updateFilter({ q: event.target.value }); }} />
        <Select allowClear aria-label={t("mobile_releases.filters.package_type")} value={filters.package_type || undefined} placeholder={t("mobile_releases.filters.package_type")} onChange={(value) => { updateFilter({ package_type: value ?? "" }); }} options={mobileReleasePackageTypes.map((value) => ({ value, label: t(`mobile_releases.package_type.${value}`) }))} />
        <Select allowClear aria-label={t("mobile_releases.filters.platform")} value={filters.platform || undefined} placeholder={t("mobile_releases.filters.platform")} onChange={(value) => { updateFilter({ platform: value ?? "" }); }} options={mobileReleasePlatforms.map((value) => ({ value, label: t(`mobile_releases.platform.${value}`) }))} />
        <Select allowClear aria-label={t("mobile_releases.filters.publish_status")} value={filters.publish_status || undefined} placeholder={t("mobile_releases.filters.publish_status")} onChange={(value) => { updateFilter({ publish_status: value ?? "" }); }} options={mobileReleasePublishStatuses.map((value) => ({ value, label: t(`mobile_releases.publish_status.${value}`) }))} />
      </div>
      {query.isError ? <Alert showIcon type="error" title={t("mobile_releases.feedback.load_error")} action={<Button onClick={() => void query.refetch()}>{t("common.actions.retry")}</Button>} /> : null}
      {screens.md ? <div aria-label={t("mobile_releases.title")} className="ak-table-scroll" role="region" tabIndex={0}><Table rowKey="id" loading={query.isPending} columns={columns} dataSource={query.data?.items ?? []} locale={{ emptyText: t("mobile_releases.empty") }} {...(permissions.has("mobile.release.delete") ? { rowSelection: { selectedRowKeys: selected, onChange: setSelected, getCheckboxProps: (item: ManagedMobileRelease) => ({ disabled: item.publish_status !== "draft" }) } } : {})} pagination={{ current: filters.page, pageSize: filters.page_size, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (page, pageSize) => { setFilters((current) => ({ ...current, page, page_size: pageSize })); } }} scroll={{ x: 1350 }} /></div> : <div className="ak-mobile-release-list">
        {query.isPending ? <Card loading size="small" /> : null}
        {!query.isPending && (query.data?.items.length ?? 0) === 0 ? <div className="ak-mobile-release-empty">{t("mobile_releases.empty")}</div> : null}
        {(query.data?.items ?? []).map((item) => <Card className="ak-mobile-release-card" key={item.id} size="small" title={<Space><code>{item.version}</code><Tag>{t(`mobile_releases.package_type.${item.package_type}`)}</Tag></Space>} extra={<Tag className={item.publish_status === "online" ? "ak-status-success" : item.publish_status === "partial" ? "ak-status-warning" : item.publish_status === "offline" ? "ak-status-error" : ""}>{t(`mobile_releases.publish_status.${item.publish_status}`)}</Tag>}><Typography.Paragraph>{item.platforms.map((value) => t(`mobile_releases.platform.${value}`)).join(" / ")}</Typography.Paragraph>{releaseActions(item)}</Card>)}
      </div>}</> : <AppSelectionRequiredState />}
    </Card>
    <ReleaseDrawer activeLocale={activeLocale} editor={editor} form={form} frozen={frozen} fullScreen={!screens.md} languages={languages} stores={selectedApp?.store_listings ?? []} feedback={feedback?.error ? feedback.key : null} saving={mutations.create.isPending || mutations.update.isPending} onActiveLocaleChange={setActiveLocale} onClose={() => { setEditor(null); }} onSave={() => void save()} onPickFile={() => { setPickerOpen(true); }} />
    <AkFilePicker open={pickerOpen} onClose={() => { setPickerOpen(false); }} onSelect={(file) => { form.setValue("package_file_id", file.id, { shouldDirty: true }); setPickerOpen(false); }} />
  </div>;
}

function ReleaseDrawer({ activeLocale, editor, form, frozen, fullScreen, languages, stores, feedback, saving, onActiveLocaleChange, onClose, onSave, onPickFile }: { activeLocale: AdminLocale; editor: Editor; form: ReturnType<typeof useForm<MobileReleaseInput>>; frozen: boolean; fullScreen: boolean; languages: SystemLanguagesState; stores: { id?: string; name: string; enabled: boolean }[]; feedback: string | null; saving: boolean; onActiveLocaleChange: (locale: AdminLocale) => void; onClose: () => void; onSave: () => void; onPickFile: () => void }) {
  const { t } = useTranslation(); const errors = form.formState.errors; const packageType = form.watch("package_type"); const sourceType = form.watch("source_type");
  const validation = (message?: string) => message ? { validateStatus: "error" as const, help: t(`mobile_releases.validation.${message}`) } : {};
  return <Drawer open={editor !== null} size={fullScreen ? "100%" : "large"} destroyOnHidden title={t(editor && "kind" in editor ? `mobile_releases.editor.create_${editor.packageType}` : frozen ? "mobile_releases.editor.detail" : "mobile_releases.editor.edit")} onClose={onClose} extra={!frozen ? <Button disabled={!languages.isReady} type="primary" loading={saving} onClick={onSave}>{t("common.actions.save")}</Button> : null}>
    {feedback ? <Alert showIcon type="error" title={t(feedback)} /> : null}
    <Form layout="vertical" disabled={frozen} className="ak-release-form">
      <div className="ak-form-grid-2"><Form.Item label={t("mobile_releases.fields.package_type")}><Controller control={form.control} name="package_type" render={({ field }) => <Select {...field} aria-label={t("mobile_releases.fields.package_type")} disabled options={mobileReleasePackageTypes.map((value) => ({ value, label: t(`mobile_releases.package_type.${value}`) }))} />} /></Form.Item><Form.Item label={t("mobile_releases.fields.version")} {...validation(errors.version?.message)}><Controller control={form.control} name="version" render={({ field }) => <Input {...field} aria-label={t("mobile_releases.fields.version")} placeholder="1.2.3" />} /></Form.Item></div>
      <Form.Item label={t("mobile_releases.fields.platforms")} {...validation(errors.platforms?.message)}><Controller control={form.control} name="platforms" render={({ field }) => <Checkbox.Group {...field} aria-label={t("mobile_releases.fields.platforms")} options={mobileReleasePlatforms.map((value) => ({ value, label: t(`mobile_releases.platform.${value}`), disabled: packageType === "native_app" && field.value.length === 1 && !field.value.includes(value) }))} />} /></Form.Item>
      {packageType === "wgt" ? <Form.Item label={t("mobile_releases.fields.minimum_native_version")} {...validation(errors.minimum_native_version?.message)}><Controller control={form.control} name="minimum_native_version" render={({ field }) => <Input {...field} aria-label={t("mobile_releases.fields.minimum_native_version")} placeholder="1.0.0" />} /></Form.Item> : null}
      <AkLocalizedFormTabs
        activeLocale={activeLocale}
        errorLocales={{
          "en-US": Boolean(errors.titles?.["en-US"] ?? errors.contents?.["en-US"]),
          "zh-CN": Boolean(errors.titles?.["zh-CN"] ?? errors.contents?.["zh-CN"]),
        }}
        languages={languages}
        onActiveLocaleChange={onActiveLocaleChange}
        renderFields={(locale, label) => <><Form.Item label={t("mobile_releases.fields.title")} {...validation(errors.titles?.[locale]?.message)}><Controller control={form.control} name={`titles.${locale}`} render={({ field }) => <Input {...field} aria-label={`${label} ${t("mobile_releases.fields.title")}`} maxLength={200} showCount />} /></Form.Item><Form.Item label={t("mobile_releases.fields.contents")} {...validation(errors.contents?.[locale]?.message)}><Controller control={form.control} name={`contents.${locale}`} render={({ field }) => <Input.TextArea {...field} aria-label={`${label} ${t("mobile_releases.fields.contents")}`} rows={7} maxLength={10_000} showCount />} /></Form.Item></>}
      />
      <Form.Item label={t("mobile_releases.fields.source_type")}><Controller control={form.control} name="source_type" render={({ field }) => <Radio.Group {...field} aria-label={t("mobile_releases.fields.source_type")} optionType="button" options={["internal", "external"].map((value) => ({ value, label: t(`mobile_releases.source_type.${value}`) }))} />} /></Form.Item>
      {sourceType === "internal" ? <Form.Item label={t("mobile_releases.fields.package_file")} {...validation(errors.package_file_id?.message)}><Space wrap><Button onClick={onPickFile}>{t("mobile_releases.actions.choose_file")}</Button>{form.watch("package_file_id") ? <code className="ak-content-slug">{form.watch("package_file_id")}</code> : <Typography.Text type="secondary">{t("mobile_releases.values.none")}</Typography.Text>}</Space></Form.Item> : <Form.Item label={t("mobile_releases.fields.external_url")} {...validation(errors.external_url?.message)}><Controller control={form.control} name="external_url" render={({ field }) => <Input {...field} aria-label={t("mobile_releases.fields.external_url")} placeholder="https://" />} /></Form.Item>}
      <Form.Item label={t("mobile_releases.fields.store_listings")}><Controller control={form.control} name="store_listing_ids" render={({ field }) => <Select {...field} aria-label={t("mobile_releases.fields.store_listings")} mode="multiple" options={stores.flatMap((store) => store.enabled && store.id ? [{ value: store.id, label: store.name }] : [])} placeholder={t("mobile_releases.fields.store_listings_placeholder")} />} /></Form.Item>
      <div className="ak-form-grid-2">{packageType === "wgt" ? <Form.Item label={t("mobile_releases.fields.is_silently")}><Controller control={form.control} name="is_silently" render={({ field }) => <Switch aria-label={t("mobile_releases.fields.is_silently")} checked={field.value} onChange={field.onChange} />} /></Form.Item> : <span />}<Form.Item label={t("mobile_releases.fields.is_mandatory")}><Controller control={form.control} name="is_mandatory" render={({ field }) => <Switch aria-label={t("mobile_releases.fields.is_mandatory")} checked={field.value} onChange={field.onChange} />} /></Form.Item></div>
      {editor && "kind" in editor ? <Form.Item label={t("mobile_releases.fields.publish_now")} extra={t("mobile_releases.fields.publish_now_hint")}><Controller control={form.control} name="publish_now" render={({ field }) => <Switch aria-label={t("mobile_releases.fields.publish_now")} checked={field.value} onChange={field.onChange} />} /></Form.Item> : null}
    </Form>
  </Drawer>;
}
