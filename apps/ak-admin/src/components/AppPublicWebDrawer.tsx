/* eslint-disable @typescript-eslint/restrict-template-expressions -- Numeric array indices are required by typed React Hook Form field paths. */
import { Alert, Button, Divider, Drawer, Form, Grid, Input, QRCode, Select, Skeleton, Space, Switch, Typography } from "antd";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import type { AdminPublicWebConfig, AdminPublicWebConfigRequest } from "../generated/api/types.gen";
import { publicWebConfig, publicWebSchema, type PublicWebFormValues } from "../features/apps/public-web";
import { useTenantKey } from "../features/tenants/hooks";
import { useAuthStore } from "../features/auth/store";
import { useSystemLanguages } from "../features/settings/system-languages";
import { ApiError } from "../shared/api/error";
import type { AdminLocale } from "../shared/i18n";
import { AkLocalizedFormTabs } from "./AkLocalizedFormTabs";
import { PublicPageActions } from "./PublicPageActions";

export function AppPublicWebDrawer({ appId, onClose }: { appId: string; onClose: () => void }) {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const tenantId = useTenantKey();
  const query = useQuery({ queryKey: ["tenant", tenantId, "public-web", appId], queryFn: () => publicWebConfig(appId) });
  return <Drawer open size={screens.md ? "large" : "100%"} title={t("apps.public_web.title")} onClose={onClose} destroyOnHidden>
    {query.isPending ? <Skeleton active /> : query.isError ? <Alert type="error" showIcon title={t("apps.feedback.load_error")} action={<Button onClick={() => { void query.refetch(); }}>{t("common.actions.retry")}</Button>} /> : <PublicWebForm key={appId} data={query.data} />}
  </Drawer>;
}
function PublicWebForm({ data }: { data: AdminPublicWebConfig }) {
  const { t, i18n } = useTranslation();
  const context = useAuthStore(state => state.context);
  const canUpdate = context?.permissions.includes("app.public_web.update") ?? false;
  const languages = useSystemLanguages();
  const [locale, setLocale] = useState<AdminLocale>("zh-CN");
  const [feedback, setFeedback] = useState("");
  const [saved, setSaved] = useState(data);
  const [invalid, setInvalid] = useState(false);
  const client = useQueryClient();
  const tenant = useTenantKey();
  const form = useForm<PublicWebFormValues>({ defaultValues: { enabled: data.enabled, apk_enabled: data.apk_enabled, promotion_enabled: data.promotion_enabled, lock_version: data.lock_version, translations: data.translations, stores: data.stores } });
  const mutation = useMutation({ mutationFn: (input: AdminPublicWebConfigRequest) => publicWebConfig(data.app_id, input) });
  const submit = form.handleSubmit(async value => {
    const parsed = publicWebSchema.safeParse(value);
    if (!parsed.success) { setInvalid(true); setFeedback("apps.public_web.validation"); return; }
    setInvalid(false);
    try {
      const result = await mutation.mutateAsync(parsed.data);
      form.reset({ enabled: result.enabled, apk_enabled: result.apk_enabled, promotion_enabled: result.promotion_enabled, lock_version: result.lock_version, translations: result.translations, stores: result.stores });
      setSaved(result); setFeedback("apps.feedback.saved"); client.setQueryData(["tenant", tenant, "public-web", data.app_id], result);
    } catch (error) { setFeedback(error instanceof ApiError && error.status === 409 ? "apps.feedback.conflict" : "apps.feedback.save_error"); }
  });
  return <Space orientation="vertical" size="large" className="ak-full-width">
    <Alert type="info" showIcon title={t("apps.public_web.hint")} />
    {!data.download_page_url ? <Alert type="warning" showIcon title={t("apps.public_web.origin_missing")} /> : null}
    {feedback ? <Alert showIcon type={feedback === "apps.feedback.saved" ? "success" : "error"} title={t(feedback)} /> : null}
    <Form layout="vertical" onFinish={() => { void submit(); }}>
      <fieldset disabled={!canUpdate || mutation.isPending} className="ak-public-web-fields">
        <Form.Item label={t("apps.public_web.enabled")}><Controller control={form.control} name="enabled" render={({ field }) => <Switch checked={field.value} onChange={field.onChange} disabled={!data.download_page_url} aria-label={t("apps.public_web.enabled")} />} /></Form.Item>
        <Form.Item label={t("apps.public_web.apk_enabled")}><Controller control={form.control} name="apk_enabled" render={({ field }) => <Switch checked={field.value} onChange={field.onChange} aria-label={t("apps.public_web.apk_enabled")} />} /></Form.Item>
        <Divider />
        <Typography.Title level={4}>{t("apps.public_web.promotion_section")}</Typography.Title>
        <Typography.Paragraph type="secondary">{t("apps.public_web.promotion_hint")}</Typography.Paragraph>
        <Form.Item label={t("apps.public_web.promotion_enabled")}><Controller control={form.control} name="promotion_enabled" render={({ field }) => <Switch checked={field.value} onChange={field.onChange} aria-label={t("apps.public_web.promotion_enabled")} />} /></Form.Item>
        <AkLocalizedFormTabs activeLocale={locale} languages={languages} errorLocales={{ "zh-CN": invalid, "en-US": invalid }} onActiveLocaleChange={setLocale} renderFields={(lang, label) => <>
          <Form.Item label={t("apps.public_web.name")}><Controller control={form.control} name={`translations.${lang}.name`} render={({ field }) => <Input {...field} maxLength={160} aria-label={`${label} ${t("apps.public_web.name")}`} />} /></Form.Item>
          <Form.Item label={t("apps.public_web.introduction")}><Controller control={form.control} name={`translations.${lang}.introduction`} render={({ field }) => <Input.TextArea {...field} maxLength={20000} rows={5} aria-label={`${label} ${t("apps.public_web.introduction")}`} />} /></Form.Item>
          <Form.Item label={t("apps.public_web.promotion_title")}><Controller control={form.control} name={`translations.${lang}.promotion_title`} render={({ field }) => <Input {...field} maxLength={160} aria-label={`${label} ${t("apps.public_web.promotion_title")}`} />} /></Form.Item>
          <Form.Item label={t("apps.public_web.promotion_description")}><Controller control={form.control} name={`translations.${lang}.promotion_description`} render={({ field }) => <Input.TextArea {...field} maxLength={500} rows={3} aria-label={`${label} ${t("apps.public_web.promotion_description")}`} />} /></Form.Item>
          <Form.Item label={t("apps.public_web.promotion_button_label")}><Controller control={form.control} name={`translations.${lang}.promotion_button_label`} render={({ field }) => <Input {...field} maxLength={80} aria-label={`${label} ${t("apps.public_web.promotion_button_label")}`} />} /></Form.Item>
        </>} />
        <Divider />
        {data.stores.map((store, index) => <section key={store.id}><Typography.Title level={4}>{store.name}</Typography.Title>
          <Form.Item label={t("apps.public_web.platform")}><Controller control={form.control} name={`stores.${index}.platform`} render={({ field }) => <Select {...field} aria-label={`${store.name} ${t("apps.public_web.platform")}`} options={[{ value: "", label: t("apps.public_web.unset") }, ...["android", "ios", "harmony"].map(value => ({ value, label: t(`mobile_releases.platform.${value}`) }))]} />} /></Form.Item>
          <Form.Item label={t("apps.public_web.web_url")}><Controller control={form.control} name={`stores.${index}.web_url`} render={({ field }) => <Input {...field} maxLength={2048} aria-label={`${store.name} ${t("apps.public_web.web_url")}`} />} /></Form.Item>
        </section>)}
      </fieldset>
      {canUpdate ? <Button type="primary" htmlType="submit" loading={mutation.isPending}>{t("common.actions.save")}</Button> : null}
    </Form>
    {saved.enabled && saved.download_page_url ? <><PublicPageActions url={saved.download_page_url} title={saved.translations[i18n.language === "en-US" ? "en-US" : "zh-CN"].name} /><QRCode value={saved.download_page_url} aria-label={t("apps.public_web.qr")} /></> : null}
  </Space>;
}
