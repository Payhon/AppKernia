import { Alert, Button, Card, Form, Select, Space, Switch, Tag, Typography, message } from "antd";
import { Controller, useForm } from "react-hook-form";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import {
  useAppLoginProviderBindingMutation,
  useAppLoginProviderBindings,
  useLoginProviderCatalog,
  useLoginProviderConfigs,
} from "../features/login-providers/hooks";
import {
  appLoginProviderBindingsWriteSchema,
  catalogMatchesDefinition,
  loginProviderCodes,
  loginProviderDefinitions,
  type AppLoginProviderBinding,
  type AppLoginProviderBindingWriteItem,
} from "../features/login-providers/model";
import { ApiError } from "../shared/api/error";
import { LoginProviderIcon } from "./LoginProviderIcon";

interface Props {
  appId: string;
  canOpenConfigurationPage: boolean;
  canUpdate: boolean;
  onDirtyChange: (dirty: boolean) => void;
}

export interface BindingFormValues {
  bindings: (Omit<AppLoginProviderBindingWriteItem, "login_provider_config_id"> & { login_provider_config_id: string })[];
}

export function loginProviderBindingFormValues(bindings: readonly AppLoginProviderBinding[]): BindingFormValues {
  const current = new Map(bindings.map((binding) => [binding.provider_code, binding]));
  return {
    bindings: loginProviderCodes.map((providerCode, index) => {
      const binding = current.get(providerCode);
      return {
        provider_code: providerCode,
        login_provider_config_id: binding?.login_provider_config_id ?? "",
        enabled: binding?.enabled ?? false,
        sort_order: binding?.sort_order ?? (index + 1) * 10,
        lock_version: binding?.lock_version ?? 0,
      };
    }),
  };
}

export function buildLoginProviderBindingInput(formValues: BindingFormValues) {
  return {
    bindings: formValues.bindings.map((item) => ({
      ...item,
      login_provider_config_id: item.login_provider_config_id || null,
      enabled: item.login_provider_config_id ? item.enabled : false,
    })),
  };
}

export function mergeLoginProviderBindingConflictValues(
  editableValues: BindingFormValues,
  latestBindings: readonly AppLoginProviderBinding[],
): BindingFormValues {
  const latestByProvider = new Map(latestBindings.map((binding) => [binding.provider_code, binding]));
  return {
    bindings: editableValues.bindings.map((item) => ({
      ...item,
      lock_version: latestByProvider.get(item.provider_code)?.lock_version ?? item.lock_version,
    })),
  };
}

export function AppLoginProviderConfigurationPanel({ appId, canOpenConfigurationPage, canUpdate, onDirtyChange }: Props) {
  const { t } = useTranslation();
  const [messageApi, holder] = message.useMessage();
  const [conflict, setConflict] = useState(false);
  const [conflictRefreshing, setConflictRefreshing] = useState(false);
  const lastAppliedAppId = useRef<string | null>(null);
  const catalog = useLoginProviderCatalog();
  const configs = useLoginProviderConfigs({ q: "", status: "active", page: 1, page_size: 100 });
  const bindings = useAppLoginProviderBindings(appId);
  const mutation = useAppLoginProviderBindingMutation(appId);
  const form = useForm<BindingFormValues>({ defaultValues: loginProviderBindingFormValues([]) });
  const dirty = form.formState.isDirty;

  useEffect(() => { onDirtyChange(dirty); }, [dirty, onDirtyChange]);
  useEffect(() => {
    if (bindings.data) {
      const appChanged = lastAppliedAppId.current !== appId;
      lastAppliedAppId.current = appId;
      if (appChanged || !form.formState.isDirty) {
        form.reset(loginProviderBindingFormValues(bindings.data));
        setConflict(false);
      }
    }
  }, [appId, bindings.data, form, form.formState.isDirty]);

  const catalogByProvider = useMemo(() => new Map((catalog.data ?? []).map((item) => [item.provider_code, item])), [catalog.data]);
  const configsByProvider = useMemo(() => new Map(loginProviderCodes.map((providerCode) => [
    providerCode,
    (configs.data?.items ?? []).filter((item) => item.provider_code === providerCode && item.status === "active" && item.last_preflight_status === "ready"),
  ])), [configs.data?.items]);
  const currentBindings = useMemo(() => new Map((bindings.data ?? []).map((item) => [item.provider_code, item])), [bindings.data]);
  const values = form.watch("bindings");
  const invalid = values.some((item) => item.enabled && item.login_provider_config_id === "");
  const catalogMismatch = loginProviderCodes.some((providerCode) => {
    const item = catalogByProvider.get(providerCode);
    return item === undefined || !catalogMatchesDefinition(item);
  });

  const save = form.handleSubmit(async (formValues) => {
    if (!canUpdate) return;
    const input = buildLoginProviderBindingInput(formValues);
    const parsed = appLoginProviderBindingsWriteSchema.safeParse(input);
    if (!parsed.success) {
      messageApi.error(t("login_providers.binding.validation"));
      return;
    }
    try {
      const saved = await mutation.mutateAsync(parsed.data);
      form.reset(loginProviderBindingFormValues(saved));
      setConflict(false);
      messageApi.success(t("login_providers.binding.saved"));
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        setConflict(true);
        return;
      }
      messageApi.error(t("login_providers.feedback.error"));
    }
  });

  const refreshConflict = async () => {
    const editableValues = form.getValues();
    setConflictRefreshing(true);
    try {
      const refreshed = await bindings.refetch();
      if (!refreshed.data) throw new Error("LOGIN_PROVIDER_BINDINGS_REFRESH_EMPTY");
      form.reset(
        mergeLoginProviderBindingConflictValues(editableValues, refreshed.data),
        { keepDefaultValues: true, keepTouched: true },
      );
      setConflict(false);
      messageApi.info(t("login_providers.binding.conflict_refreshed"));
    } catch {
      messageApi.error(t("login_providers.binding.conflict_refresh_error"));
    } finally {
      setConflictRefreshing(false);
    }
  };

  const loading = catalog.isPending || configs.isPending || bindings.isPending;
  const loadError = catalog.isError || configs.isError || bindings.isError;
  return <Space className="ak-client-config-panel ak-login-provider-binding-panel" orientation="vertical" size="large">
    {holder}
    <Alert description={t("login_providers.binding.description")} showIcon title={t("login_providers.binding.title")} type="info" />
    {loadError ? <Alert action={<Button onClick={() => { void catalog.refetch(); void configs.refetch(); void bindings.refetch(); }}>{t("common.actions.retry")}</Button>} role="alert" showIcon title={t("login_providers.binding.load_error")} type="error" /> : null}
    {catalogMismatch && !catalog.isPending ? <Alert role="alert" showIcon title={t("login_providers.feedback.catalog_mismatch")} type="error" /> : null}
    {conflict ? <Alert action={<Button loading={conflictRefreshing} onClick={() => { void refreshConflict(); }}>{t("common.actions.refresh")}</Button>} description={t("login_providers.binding.conflict_description")} role="alert" showIcon title={t("login_providers.binding.conflict")} type="warning" /> : null}
    <Form layout="vertical">
      <Space className="ak-login-provider-binding-list" orientation="vertical" size="middle">
        {loginProviderCodes.map((providerCode, index) => {
          const definition = loginProviderDefinitions[providerCode];
          const catalogItem = catalogByProvider.get(providerCode);
          const catalogReady = catalogItem !== undefined && catalogMatchesDefinition(catalogItem);
          const options = configsByProvider.get(providerCode) ?? [];
          const current = currentBindings.get(providerCode);
          const currentConfigId = current?.login_provider_config_id ?? null;
          const currentUnavailable = currentConfigId !== null && (current?.config_status !== "active" || current.preflight_status !== "ready");
          const row = values[index];
          if (!row) return null;
          const selectedId = row.login_provider_config_id;
          const selectedConfig = options.find((item) => item.id === selectedId);
          return <Card
            className="ak-login-provider-binding-card"
            extra={<Controller control={form.control} name={`bindings.${String(index)}.enabled` as `bindings.${number}.enabled`} render={({ field }) => <Switch aria-label={t("login_providers.binding.enable_provider", { provider: t(`login_providers.provider.${providerCode}`) })} checked={field.value} disabled={!canUpdate || loading || !catalogReady || (options.length === 0 && !field.value)} onChange={field.onChange} />} />}
            key={providerCode}
            size="small"
            title={<Space><LoginProviderIcon provider={providerCode} /><span>{t(`login_providers.provider.${providerCode}`)}</span></Space>}
          >
            <Space orientation="vertical" size="small" style={{ width: "100%" }}>
              <Space wrap>{definition.platforms.map((platform) => <Tag key={`platform-${platform}`}>{t(`login_providers.platform.${platform}`)}</Tag>)}{definition.buildVariants.map((variant) => <Tag key={`variant-${variant}`}>{t(`login_providers.build_variant.${variant}`)}</Tag>)}</Space>
              <Controller control={form.control} name={`bindings.${String(index)}.login_provider_config_id` as `bindings.${number}.login_provider_config_id`} render={({ field }) => <Form.Item help={field.value === "" && row.enabled ? t("login_providers.binding.config_required") : undefined} label={t("login_providers.binding.config")} required validateStatus={field.value === "" && row.enabled ? "error" : ""}>
                <Select
                  aria-label={t("login_providers.binding.config_for_provider", { provider: t(`login_providers.provider.${providerCode}`) })}
                  allowClear={!row.enabled}
                  disabled={!canUpdate || loading || !catalogReady}
                  loading={configs.isPending}
                  onBlur={field.onBlur}
                  onChange={(value) => { field.onChange(value); }}
                  options={[
                    ...(currentUnavailable && currentConfigId ? [{ value: currentConfigId, label: `${current?.config_name ?? t(`login_providers.provider.${providerCode}`)} · ${t("login_providers.binding.unavailable")}`, disabled: true }] : []),
                    ...options.map((item) => ({ value: item.id, label: `${item.name} · ${item.external_client_id}` })),
                  ]}
                  placeholder={t("login_providers.binding.select_config")}
                  ref={field.ref}
                  value={field.value || null}
                />
              </Form.Item>} />
              {!catalogReady && !catalog.isPending ? <Alert showIcon title={t("login_providers.binding.provider_unavailable")} type="error" /> : null}
              {catalogReady && options.length === 0 && !configs.isPending ? <Alert action={canOpenConfigurationPage ? <Button href="/system/settings/login-providers" rel="noopener noreferrer" size="small" target="_blank">{t("login_providers.binding.open_configs")}</Button> : undefined} showIcon title={t("login_providers.binding.no_active_config")} type="warning" /> : null}
              {currentUnavailable ? <Alert description={t("login_providers.binding.runtime_fail_closed")} showIcon title={t("login_providers.binding.unavailable")} type="error" /> : null}
              {selectedConfig && row.enabled ? <Alert showIcon title={t("login_providers.binding.ready")} type="success" /> : null}
              {definition.publicFields.some((item) => item.requiresRebuild) ? <Typography.Text type="secondary">{t("login_providers.binding.rebuild_required")}</Typography.Text> : null}
            </Space>
          </Card>;
        })}
      </Space>
    </Form>
    <div className="ak-client-config-actions"><span />{canUpdate ? <Button disabled={!dirty || invalid || loadError || catalogMismatch} loading={mutation.isPending} onClick={() => { void save(); }} type="primary">{t("common.actions.save")}</Button> : null}</div>
  </Space>;
}
