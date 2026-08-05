import { Alert, Button, Card, Drawer, Form, Grid, Input, Select, Switch, Table, Tag, Typography, type TableColumnsType } from "antd";
import { useNavigate } from "@tanstack/react-router";
import { Controller, useForm, type FieldPath } from "react-hook-form";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { AdminMobileRelease } from "../generated/api/types.gen";
import { useAuthStore } from "../features/auth/store";
import { useMobileReleaseMutations, useMobileReleases } from "../features/mobile-releases/hooks";
import { defaultMobileReleaseInput, mobileReleaseInputSchema, mobileReleasePlatforms, type MobileReleaseInput, type MobileReleasePlatform } from "../features/mobile-releases/model";
import { ApiError } from "../shared/api/error";

type Editor = AdminMobileRelease | "new" | null;
type Feedback = { key: string; error: boolean } | null;

function readPlatform(): MobileReleasePlatform | "" {
  const value = new URLSearchParams(location.search).get("platform");
  return mobileReleasePlatforms.includes(value as MobileReleasePlatform) ? value as MobileReleasePlatform : "";
}

function releaseInput(item: AdminMobileRelease): MobileReleaseInput {
  return {
    platform: item.platform,
    current_version: item.current_version,
    minimum_version: item.minimum_version,
    upgrade_url: item.upgrade_url ?? "",
    release_notes: item.release_notes,
    active: item.active,
    lock_version: item.lock_version,
  };
}

function errorKey(error: unknown): string {
  return error instanceof ApiError && error.status === 409
    ? "mobile_releases.feedback.conflict"
    : "mobile_releases.feedback.save_error";
}

export function AdminMobileReleasesPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [platform, setPlatform] = useState<MobileReleasePlatform | "">(readPlatform);
  const [editor, setEditor] = useState<Editor>(null);
  const [feedback, setFeedback] = useState<Feedback>(null);
  const [editorFeedback, setEditorFeedback] = useState<string | null>(null);
  const query = useMobileReleases();
  const mutations = useMobileReleaseMutations();
  const form = useForm<MobileReleaseInput>({ defaultValues: defaultMobileReleaseInput() });
  const date = useMemo(() => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }), [i18n.language]);
  const items = useMemo(() => (query.data ?? []).filter((item) => !platform || item.platform === platform), [platform, query.data]);

  useEffect(() => {
    void navigate({ to: "/system/mobile/releases", search: { platform }, replace: true });
  }, [navigate, platform]);

  const openEditor = (value: AdminMobileRelease | "new") => {
    const defaults = value === "new" ? { ...defaultMobileReleaseInput(), ...(platform ? { platform } : {}) } : releaseInput(value);
    form.reset(defaults);
    setEditorFeedback(null);
    setEditor(value);
  };

  const save = form.handleSubmit(async (values) => {
    form.clearErrors();
    const parsed = mobileReleaseInputSchema.safeParse(values);
    if (!parsed.success) {
      parsed.error.issues.forEach((issue) => {
        form.setError(issue.path.join(".") as FieldPath<MobileReleaseInput>, { message: issue.message });
      });
      setEditorFeedback("mobile_releases.feedback.validation_error");
      return;
    }
    try {
      if (editor === "new") await mutations.create.mutateAsync(parsed.data);
      else if (editor) await mutations.update.mutateAsync({ id: editor.id, input: parsed.data });
      setEditor(null);
      setEditorFeedback(null);
      setFeedback({ key: "mobile_releases.feedback.saved", error: false });
    } catch (error) {
      setEditorFeedback(errorKey(error));
    }
  });

  const columns: TableColumnsType<AdminMobileRelease> = [
    {
      title: t("mobile_releases.columns.platform"),
      dataIndex: "platform",
      width: 130,
      render: (value: MobileReleasePlatform) => <Tag>{t(`mobile_releases.platform.${value}`)}</Tag>,
    },
    {
      title: t("mobile_releases.columns.current"),
      dataIndex: "current_version",
      width: 145,
      render: (value: string) => <code className="ak-version-value">{value}</code>,
    },
    {
      title: t("mobile_releases.columns.minimum"),
      dataIndex: "minimum_version",
      width: 145,
      responsive: ["sm"],
      render: (value: string) => <code className="ak-version-value">{value}</code>,
    },
    {
      title: t("mobile_releases.columns.status"),
      dataIndex: "active",
      width: 110,
      render: (value: boolean) => <Tag className={value ? "ak-status-success" : ""}>{t(value ? "mobile_releases.status.active" : "mobile_releases.status.inactive")}</Tag>,
    },
    {
      title: t("mobile_releases.columns.upgrade_url"),
      dataIndex: "upgrade_url",
      responsive: ["lg"],
      render: (value: string | null | undefined) => value
        ? <a className="ak-release-url" href={value} target="_blank" rel="noreferrer" aria-label={t("mobile_releases.actions.open_upgrade_url")}>{value}</a>
        : <span className="ak-table-secondary">{t("mobile_releases.values.none")}</span>,
    },
    {
      title: t("mobile_releases.columns.updated"),
      dataIndex: "updated_at",
      width: 190,
      responsive: ["lg"],
      render: (value: string) => date.format(new Date(value)),
    },
    {
      title: t("mobile_releases.columns.actions"),
      width: 100,
      render: (_, item) => permissions.has("mobile.release.update")
        ? <Button size="small" onClick={() => { openEditor(item); }}>{t("common.actions.edit")}</Button>
        : null,
    },
  ];

  return <div className="ak-page-container">
    <header className="ak-page-heading">
      <div>
        <Typography.Title level={1}>{t("mobile_releases.title")}</Typography.Title>
        <Typography.Paragraph type="secondary">{t("mobile_releases.description")}</Typography.Paragraph>
      </div>
      {permissions.has("mobile.release.create") ? <Button type="primary" onClick={() => { openEditor("new"); }}>{t("mobile_releases.actions.create")}</Button> : null}
    </header>
    {feedback ? <div className={feedback.error ? "ak-form-error" : "ak-org-feedback"} role={feedback.error ? "alert" : "status"}>{t(feedback.key)}</div> : null}
    <Card>
      <div className="ak-release-filters" role="search" aria-label={t("mobile_releases.filters.landmark")}>
        <Select allowClear value={platform || undefined} placeholder={t("mobile_releases.filters.platform")} aria-label={t("mobile_releases.filters.platform")} onChange={(value) => { setPlatform(value ?? ""); }} options={mobileReleasePlatforms.map((value) => ({ value, label: t(`mobile_releases.platform.${value}`) }))} />
      </div>
      {query.isError ? <Alert showIcon type="error" title={t("mobile_releases.feedback.load_error")} action={<Button onClick={() => void query.refetch()}>{t("common.actions.retry")}</Button>} /> : null}
      {screens.md ? <div className="ak-table-scroll">
        <Table rowKey="id" loading={query.isPending} columns={columns} dataSource={items} pagination={false} locale={{ emptyText: t("mobile_releases.empty") }} scroll={{ x: 980 }} />
      </div> : <div className="ak-mobile-release-list">
        {query.isPending ? <Card loading size="small" /> : null}
        {!query.isPending && items.length === 0 ? <div className="ak-mobile-release-empty">{t("mobile_releases.empty")}</div> : null}
        {!query.isPending ? items.map((item) => <Card
          className="ak-mobile-release-card"
          key={item.id}
          size="small"
          title={<Tag>{t(`mobile_releases.platform.${item.platform}`)}</Tag>}
          extra={<Tag className={item.active ? "ak-status-success" : ""}>{t(item.active ? "mobile_releases.status.active" : "mobile_releases.status.inactive")}</Tag>}
        >
          <dl className="ak-mobile-release-facts">
            <div><dt>{t("mobile_releases.columns.current")}</dt><dd><code className="ak-version-value">{item.current_version}</code></dd></div>
            <div><dt>{t("mobile_releases.columns.minimum")}</dt><dd><code className="ak-version-value">{item.minimum_version}</code></dd></div>
          </dl>
          {permissions.has("mobile.release.update") ? <Button block onClick={() => { openEditor(item); }}>{t("common.actions.edit")}</Button> : null}
        </Card>) : null}
      </div>}
      {query.isFetching && !query.isPending ? <div className="ak-content-stale" role="status">{t("mobile_releases.states.refreshing")}</div> : null}
    </Card>
    <MobileReleaseDrawer editor={editor} form={form} open={editor !== null} fullScreen={!screens.md} feedbackKey={editorFeedback} saving={mutations.create.isPending || mutations.update.isPending} onClose={() => { setEditor(null); setEditorFeedback(null); }} onSave={() => void save()} />
  </div>;
}

function MobileReleaseDrawer({ editor, form, open, fullScreen, feedbackKey, saving, onClose, onSave }: { editor: Editor; form: ReturnType<typeof useForm<MobileReleaseInput>>; open: boolean; fullScreen: boolean; feedbackKey: string | null; saving: boolean; onClose: () => void; onSave: () => void }) {
  const { t } = useTranslation();
  const errors = form.formState.errors;
  const help = (message: string | undefined, fallback: string) => message === "minimum_newer"
    ? t("mobile_releases.validation.minimum_order")
    : message === "active_url_required"
      ? t("mobile_releases.validation.active_https_url")
      : message ? t(fallback) : undefined;
  const validation = (message: string | undefined, fallback: string) => message ? { validateStatus: "error" as const, help: help(message, fallback) } : {};
  return <Drawer open={open} size={fullScreen ? "100%" : "large"} destroyOnHidden title={t(editor === "new" ? "mobile_releases.editor.create" : "mobile_releases.editor.edit")} onClose={onClose} extra={<Button type="primary" loading={saving} onClick={onSave}>{t("common.actions.save")}</Button>}>
    {feedbackKey ? <Alert className="ak-release-drawer-alert" showIcon type="error" title={t(feedbackKey)} /> : null}
    <Form layout="vertical">
      <Form.Item label={t("mobile_releases.fields.platform")} {...validation(errors.platform?.message, "mobile_releases.validation.required")}>
        <Controller control={form.control} name="platform" render={({ field }) => <Select {...field} aria-label={t("mobile_releases.fields.platform")} disabled={editor !== "new"} options={mobileReleasePlatforms.map((value) => ({ value, label: t(`mobile_releases.platform.${value}`) }))} />} />
      </Form.Item>
      <div className="ak-mobile-release-version-grid">
        <Form.Item label={t("mobile_releases.fields.current_version")} {...validation(errors.current_version?.message, "mobile_releases.validation.semver")}>
          <Controller control={form.control} name="current_version" render={({ field }) => <Input {...field} aria-label={t("mobile_releases.fields.current_version")} autoComplete="off" placeholder="1.2.3" />} />
        </Form.Item>
        <Form.Item label={t("mobile_releases.fields.minimum_version")} {...validation(errors.minimum_version?.message, "mobile_releases.validation.semver")}>
          <Controller control={form.control} name="minimum_version" render={({ field }) => <Input {...field} aria-label={t("mobile_releases.fields.minimum_version")} autoComplete="off" placeholder="1.0.0" />} />
        </Form.Item>
      </div>
      <Form.Item label={t("mobile_releases.fields.upgrade_url")} {...validation(errors.upgrade_url?.message, "mobile_releases.validation.https_url")}>
        <Controller control={form.control} name="upgrade_url" render={({ field }) => <Input {...field} aria-label={t("mobile_releases.fields.upgrade_url")} inputMode="url" placeholder="https://" />} />
      </Form.Item>
      <Form.Item label={t("mobile_releases.fields.active")} extra={t("mobile_releases.fields.active_hint")}>
        <Controller control={form.control} name="active" render={({ field }) => <Switch aria-label={t("mobile_releases.fields.active")} checked={field.value} onChange={field.onChange} />} />
      </Form.Item>
      {(["zh-CN", "en-US"] as const).map((locale) => <Card className="ak-content-locale-card" key={locale} size="small" title={t(`mobile_releases.locales.${locale}`)}>
        <Form.Item label={t("mobile_releases.fields.release_notes")} {...validation(errors.release_notes?.[locale]?.message, "mobile_releases.validation.required")}>
          <Controller control={form.control} name={`release_notes.${locale}`} render={({ field }) => <Input.TextArea {...field} aria-label={`${t(`mobile_releases.locales.${locale}`)} ${t("mobile_releases.fields.release_notes")}`} rows={7} showCount maxLength={10_000} />} />
        </Form.Item>
      </Card>)}
    </Form>
  </Drawer>;
}
