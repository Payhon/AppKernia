import { PublicPagePreviewModal } from "../components/PublicPagePreviewModal";
import { useTenantKey } from "../features/tenants/hooks";
import { PublicPageActions } from "../components/PublicPageActions";
import { Alert, Button, Card, Drawer, Form, Grid, Input, Modal, Select, Space, Table, Tag, Tooltip, Typography, type TableColumnsType } from "antd";
import { Controller, useForm, type FieldPath } from "react-hook-form";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AppSelectionRequiredState } from "../components/AppSelectionRequiredState";
import { AkLocalizedFormTabs } from "../components/AkLocalizedFormTabs";
import { useAuthStore } from "../features/auth/store";
import { useAppPages, useApplicationMutations } from "../features/apps/hooks";
import { appPageEditorInputSchema, getAppPageTitle, hasPublishableAppPageDraft, shouldShowAppPagePublish, toAppPageEditorInput, toAppPageInput, type AppPageEditorInput, type AppPageLocale, type ManagedPage } from "../features/apps/model";
import { AppScopeContext, type AppScope } from "../features/apps/scope";
import { findFirstInvalidLanguage, useSystemLanguages, type SystemLanguagesState } from "../features/settings/system-languages";
import type { AdminLocale } from "../shared/i18n";

type Editor = ManagedPage | "new" | null;
const reserved = new Set(["privacy-policy", "terms-of-service", "about-us", "faq", "contact-support"]);
const defaults = (): AppPageEditorInput => ({ slug: "", page_type: "custom", publish: false, translations: { "zh-CN": { title: "", body_format: "markdown", body: "" }, "en-US": { title: "", body_format: "markdown", body: "" } } });
const fromPage = (page: ManagedPage): AppPageEditorInput => toAppPageEditorInput(page);

export function AdminAppPagesPage() { return <AppScopeContext>{(scope) => <AppPagesContents scope={scope} />}</AppScopeContext>; }

function AppPagesContents({ scope }: { scope: AppScope }) {
  const { t, i18n } = useTranslation();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [q, setQ] = useState("");
  const status = "";
  const tenant = useTenantKey();
  const scopeKey = `${tenant}:${scope.appId ?? "none"}`;
  const [preview, setPreview] = useState<{ scope: string; url: string; title: string } | null>(null);
  useEffect(() => { setPreview(null); }, [scopeKey]);
  const [editor, setEditor] = useState<Editor>(null);
  const [feedback, setFeedback] = useState<{ key: string; error: boolean } | null>(null);
  const [activeLocale, setActiveLocale] = useState<AdminLocale>("zh-CN");
  const languages = useSystemLanguages();
  const form = useForm<AppPageEditorInput>({ defaultValues: defaults() });
  const pages = useAppPages(scope.appId, { q, status, page: 1, page_size: 100 });
  const mutations = useApplicationMutations();
  const date = useMemo(() => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }), [i18n.language]);
  const pageLocale: AppPageLocale = i18n.language === "en-US" ? "en-US" : "zh-CN";
  const pageTitle = (page: ManagedPage) => getAppPageTitle(page, pageLocale) ?? t(`apps.pages.type.${page.page_type}`);
  const open = (value: ManagedPage | "new") => { form.reset(value === "new" ? defaults() : fromPage(value)); setActiveLocale(languages.preferredLocale); setEditor(value); };
  const save = form.handleSubmit(async (values) => {
    if (!scope.appId) return;
    const parsed = appPageEditorInputSchema.safeParse(values);
    if (!parsed.success) { parsed.error.issues.forEach((issue) => { form.setError(issue.path.join(".") as FieldPath<AppPageEditorInput>, { message: t("apps.feedback.validation_error") }); }); const invalidLocale = findFirstInvalidLanguage(languages.languages, parsed.error.issues); if (invalidLocale) setActiveLocale(invalidLocale); setFeedback({ key: "apps.feedback.validation_error", error: true }); return; }
    try { const input = toAppPageInput(parsed.data); if (editor === "new") await mutations.createPage.mutateAsync({ appId: scope.appId, input }); else if (editor) await mutations.updatePage.mutateAsync({ appId: scope.appId, pageId: editor.slug, input }); setEditor(null); setFeedback({ key: "apps.feedback.saved", error: false }); } catch { setFeedback({ key: "apps.feedback.save_error", error: true }); }
  });
  const publish = async (page: ManagedPage) => { if (!scope.appId) return; try { await mutations.publishPage.mutateAsync({ appId: scope.appId, pageId: page.slug, lockVersion: page.lock_version }); setFeedback({ key: "apps.pages.feedback.published", error: false }); } catch { setFeedback({ key: "apps.feedback.save_error", error: true }); } };
  const deletePage = (page: ManagedPage) => { const appId = scope.appId; if (!appId || reserved.has(page.page_type)) return; Modal.confirm({ title: t("apps.pages.delete.title"), content: t("apps.pages.delete.description", { name: pageTitle(page) }), okText: t("common.actions.delete"), cancelText: t("common.actions.cancel"), okButtonProps: { danger: true }, onOk: async () => { try { await mutations.deletePage.mutateAsync({ appId, pageId: page.slug, lockVersion: page.lock_version }); setFeedback({ key: "apps.feedback.deleted", error: false }); } catch { setFeedback({ key: "apps.feedback.save_error", error: true }); } } }); };
  const columns: TableColumnsType<ManagedPage> = [
    { title: t("apps.pages.columns.title"), render: (_, item) => <div><strong>{pageTitle(item)}</strong><div className="ak-content-slug">/{item.slug}</div></div> },
    { title: t("apps.pages.columns.type"), dataIndex: "page_type", responsive: ["md"], render: (value: string) => <Tag>{t(`apps.pages.type.${value}`)}</Tag> },
    { title: t("apps.pages.columns.status"), dataIndex: "status", render: (value: ManagedPage["status"]) => <Tag className={value === "published" ? "ak-status-success" : "ak-status-warning"}>{t(`apps.pages.status.${value}`)}</Tag> },
    { title: t("apps.pages.columns.updated"), dataIndex: "updated_at", responsive: ["lg"], render: (value: string) => date.format(new Date(value)) },
    { key: "actions", title: t("apps.pages.columns.actions"), width: screens.md ? 240 : 100, render: (_, item) => <Space wrap>{item.status === "published" ? <PublicPageActions url={item.public_url} title={pageTitle(item)} onPreview={(url, title) => { setPreview({ scope: scopeKey, url, title }); }} /> : null}{permissions.has("app.content.update") ? <Button size="small" onClick={() => { open(item); }}>{t("common.actions.edit")}</Button> : null}{shouldShowAppPagePublish(item) && permissions.has("app.content.publish") ? <Tooltip title={hasPublishableAppPageDraft(item) ? undefined : t("apps.pages.feedback.save_before_publish")}><span><Button disabled={!hasPublishableAppPageDraft(item) || mutations.publishPage.isPending} loading={mutations.publishPage.isPending} size="small" type="primary" onClick={() => void publish(item)}>{t("apps.pages.actions.publish")}</Button></span></Tooltip> : null}{!reserved.has(item.page_type) && permissions.has("app.content.delete") ? <Button danger size="small" onClick={() => { deletePage(item); }}>{t("common.actions.delete")}</Button> : <Typography.Text type="secondary">{t("apps.pages.reserved")}</Typography.Text>}</Space> },
  ];
  return <div className="ak-page-container"><header className="ak-page-heading"><div><Typography.Title level={1}>{t("apps.pages.title")}</Typography.Title><Typography.Paragraph type="secondary">{t("apps.pages.description")}</Typography.Paragraph></div>{scope.appId && !scope.disabled && permissions.has("app.content.create") ? <Button type="primary" onClick={() => { open("new"); }}>{t("apps.pages.actions.create")}</Button> : null}</header>{feedback ? <Alert showIcon type={feedback.error ? "error" : "success"} title={t(feedback.key)} /> : null}<Card>{scope.appId ? <><div className="ak-content-filters" role="search" aria-label={t("apps.pages.filters.landmark")}><Input.Search allowClear aria-label={t("apps.pages.filters.q")} onChange={(event) => { setQ(event.target.value); }} placeholder={t("apps.pages.filters.q")} value={q} /></div>{pages.isError ? <Alert showIcon type="error" title={t("apps.feedback.load_error")} action={<Button onClick={() => void pages.refetch()}>{t("common.actions.retry")}</Button>} /> : null}<div className="ak-table-scroll"><Table columns={columns} dataSource={pages.data?.items ?? []} loading={pages.isPending} locale={{ emptyText: t("apps.pages.empty") }} pagination={false} rowKey="id" scroll={{ x: 880 }} /></div></> : <AppSelectionRequiredState />}</Card><PublicPagePreviewModal open={preview !== null && preview.scope === scopeKey} url={preview?.url} title={preview?.title ?? t("apps.public_web.preview")} onClose={() => {
    const url = preview?.url;
    setPreview(null);
    if (url) requestAnimationFrame(() => document.querySelector<HTMLButtonElement>(`[data-public-preview-url="${CSS.escape(url)}"]`)?.focus());
  }} /><PageDrawer activeLocale={activeLocale} editor={editor} form={form} fullScreen={!screens.md} languages={languages} saving={mutations.createPage.isPending || mutations.updatePage.isPending} onActiveLocaleChange={setActiveLocale} onClose={() => { setEditor(null); }} onSave={() => void save()} /></div>;
}

function PageDrawer({ activeLocale, editor, form, fullScreen, languages, saving, onActiveLocaleChange, onClose, onSave }: { activeLocale: AdminLocale; editor: Editor; form: ReturnType<typeof useForm<AppPageEditorInput>>; fullScreen: boolean; languages: SystemLanguagesState; saving: boolean; onActiveLocaleChange: (locale: AdminLocale) => void; onClose: () => void; onSave: () => void }) {
  const { t } = useTranslation();
  const pageType = form.watch("page_type");
  const protectedType = editor !== "new" && (reserved.has(pageType));
  const errors = form.formState.errors;
  const validation = (message?: string) => message ? { validateStatus: "error" as const, help: message } : {};
  return <Drawer destroyOnHidden extra={<Button disabled={!languages.isReady} loading={saving} type="primary" onClick={onSave}>{t("common.actions.save")}</Button>} onClose={onClose} open={editor !== null} size={fullScreen ? "100%" : "large"} title={t(editor === "new" ? "apps.pages.editor.create" : "apps.pages.editor.edit")}><Form layout="vertical"><Form.Item label={t("apps.pages.fields.slug")} {...validation(errors.slug?.message)}><Controller control={form.control} name="slug" render={({ field }) => <Input {...field} aria-label={t("apps.pages.fields.slug")} disabled={protectedType} />} /></Form.Item><Form.Item label={t("apps.pages.fields.type")} {...validation(errors.page_type?.message)}><Controller control={form.control} name="page_type" render={({ field }) => <Select {...field} aria-label={t("apps.pages.fields.type")} disabled={protectedType} options={["privacy-policy", "terms-of-service", "about-us", "faq", "contact-support", "custom"].map((value) => ({ value, label: t(`apps.pages.type.${value}`) }))} />} /></Form.Item>{protectedType ? <Alert showIcon type="info" title={t("apps.pages.reserved_hint")} /> : null}<AkLocalizedFormTabs activeLocale={activeLocale} errorLocales={{ "en-US": Boolean(errors.translations?.["en-US"]), "zh-CN": Boolean(errors.translations?.["zh-CN"]) }} languages={languages} onActiveLocaleChange={onActiveLocaleChange} renderFields={(locale, label) => <><Form.Item label={t("apps.pages.fields.title")} {...validation(errors.translations?.[locale]?.title?.message)}><Controller control={form.control} name={`translations.${locale}.title`} render={({ field }) => <Input {...field} aria-label={`${label} ${t("apps.pages.fields.title")}`} />} /></Form.Item><Form.Item label={t("apps.pages.fields.body_format")} {...validation(errors.translations?.[locale]?.body_format?.message)}><Controller control={form.control} name={`translations.${locale}.body_format`} render={({ field }) => <Select {...field} aria-label={`${label} ${t("apps.pages.fields.body_format")}`} options={["markdown", "blocks"].map((value) => ({ value, label: t(`apps.pages.body_format.${value}`) }))} />} /></Form.Item><Form.Item label={t(`apps.pages.fields.${form.watch(`translations.${locale}.body_format`) === "blocks" ? "blocks" : "body"}`)} extra={form.watch(`translations.${locale}.body_format`) === "blocks" ? t("apps.pages.fields.blocks_hint") : undefined} {...validation(errors.translations?.[locale]?.body?.message)}><Controller control={form.control} name={`translations.${locale}.body`} render={({ field }) => <Input.TextArea {...field} aria-label={`${label} ${t("apps.pages.fields.body")}`} maxLength={100000} rows={16} showCount />} /></Form.Item></>} /></Form></Drawer>;
}
