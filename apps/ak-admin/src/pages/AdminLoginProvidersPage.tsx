import {
  CheckCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  ExperimentOutlined,
  KeyOutlined,
  PlusOutlined,
  StopOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Col,
  Drawer,
  Form,
  Grid,
  Input,
  List,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
  type TableColumnsType,
} from "antd";
import { Controller, useForm } from "react-hook-form";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { LoginProviderApplicationGuide } from "../components/LoginProviderApplicationGuide";
import { LoginProviderIcon } from "../components/LoginProviderIcon";
import { useAuthStore } from "../features/auth/store";
import {
  useLoginProviderCatalog,
  useLoginProviderConfigMutations,
  useLoginProviderConfigs,
} from "../features/login-providers/hooks";
import { getLoginProviderConfig } from "../features/login-providers/api";
import {
  buildLoginProviderConfigInput,
  catalogMatchesDefinition,
  emptyLoginProviderConfigForm,
  loginProviderCodes,
  loginProviderConfigToForm,
  loginProviderConfigActions,
  loginProviderDefinitions,
  mergeLoginProviderConflictValues,
  readLoginProviderFilters,
  validateLoginProviderConfigForm,
  validateSecretValues,
  type LoginProviderCode,
  type LoginProviderConfig,
  type LoginProviderConfigFilters,
  type LoginProviderConfigFormValues,
  type LoginProviderConfigStatus,
  type LoginProviderFieldDefinition,
} from "../features/login-providers/model";
import { ApiError } from "../shared/api/error";

interface SecretFormValues {
  values: Record<string, string>;
}

type PublicValuePath = `public_values.${string}`;

function publicFieldPath(name: string): PublicValuePath {
  return `public_values.${name}`;
}

export function AdminLoginProvidersPage() {
  const { t, i18n } = useTranslation();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [messageApi, holder] = message.useMessage();
  const [filters, setFilters] = useState(() => readLoginProviderFilters(location.search));
  const [editing, setEditing] = useState<LoginProviderConfig | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [secretEditor, setSecretEditor] = useState<LoginProviderConfig | null>(null);
  const [conflict, setConflict] = useState(false);
  const [conflictRefreshing, setConflictRefreshing] = useState(false);
  const catalog = useLoginProviderCatalog();
  const configs = useLoginProviderConfigs(filters);
  const mutations = useLoginProviderConfigMutations();
  const form = useForm<LoginProviderConfigFormValues>({ defaultValues: emptyLoginProviderConfigForm() });
  const secretForm = useForm<SecretFormValues>({ defaultValues: { values: {} } });
  const provider = form.watch("provider_code");
  const definition = loginProviderDefinitions[provider];

  const catalogByProvider = useMemo(() => new Map((catalog.data ?? []).map((item) => [item.provider_code, item])), [catalog.data]);
  const supportedProviders = useMemo(() => loginProviderCodes.filter((code) => {
    const item = catalogByProvider.get(code);
    return item !== undefined && catalogMatchesDefinition(item);
  }), [catalogByProvider]);

  useEffect(() => {
    const params = new URLSearchParams();
    if (filters.q) params.set("q", filters.q);
    if (filters.provider_code) params.set("provider", filters.provider_code);
    if (filters.status) params.set("status", filters.status);
    if (filters.page > 1) params.set("page", String(filters.page));
    history.replaceState(null, "", `${location.pathname}${params.size > 0 ? `?${params.toString()}` : ""}`);
  }, [filters]);

  const openEditor = (item: LoginProviderConfig | null) => {
    const initialProvider = item?.provider_code ?? supportedProviders[0] ?? "wechat";
    setEditing(item);
    setConflict(false);
    form.reset(item ? loginProviderConfigToForm(item) : emptyLoginProviderConfigForm(initialProvider));
    setDrawerOpen(true);
  };

  const closeEditor = () => {
    if (!form.formState.isDirty) {
      setDrawerOpen(false);
      return;
    }
    Modal.confirm({
      title: t("login_providers.unsaved.title"),
      content: t("login_providers.unsaved.description"),
      okText: t("login_providers.unsaved.discard"),
      okButtonProps: { danger: true },
      cancelText: t("common.actions.cancel"),
      onOk: () => { setDrawerOpen(false); },
    });
  };

  const refreshConflict = async () => {
    if (!editing) return;
    const editableValues = form.getValues();
    setConflictRefreshing(true);
    try {
      const latest = await getLoginProviderConfig(editing.id);
      setEditing(latest);
      form.reset(
        mergeLoginProviderConflictValues(editableValues, latest),
        { keepDefaultValues: true, keepErrors: true, keepTouched: true },
      );
      setConflict(false);
      messageApi.info(t("login_providers.feedback.conflict_refreshed"));
    } catch {
      messageApi.error(t("login_providers.feedback.conflict_refresh_error"));
    } finally {
      setConflictRefreshing(false);
    }
  };

  const setFormIssues = (values: LoginProviderConfigFormValues): boolean => {
    form.clearErrors();
    const issues = validateLoginProviderConfigForm(values);
    for (const issue of issues) {
      const path = (["name", "description", "external_client_id"].includes(issue.field) ? issue.field : publicFieldPath(issue.field)) as "name" | "description" | "external_client_id" | PublicValuePath;
      form.setError(path, { message: t(issue.key) });
    }
    if (issues[0]) {
      const first = issues[0];
      const path = (["name", "description", "external_client_id"].includes(first.field) ? first.field : publicFieldPath(first.field)) as "name" | "description" | "external_client_id" | PublicValuePath;
      form.setFocus(path);
    }
    return issues.length === 0;
  };

  const save = form.handleSubmit(async (values) => {
    if (!setFormIssues(values)) return;
    const catalogItem = catalogByProvider.get(values.provider_code);
    if (!catalogItem || !catalogMatchesDefinition(catalogItem)) {
      messageApi.error(t("login_providers.feedback.catalog_mismatch"));
      return;
    }
    const input = buildLoginProviderConfigInput(values, catalogItem.config_schema_version, editing?.lock_version);
    if (!input) return;
    try {
      if (editing) await mutations.update.mutateAsync({ id: editing.id, input });
      else await mutations.create.mutateAsync(input);
      messageApi.success(t("login_providers.feedback.saved"));
      setConflict(false);
      setDrawerOpen(false);
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        setConflict(true);
        return;
      }
      messageApi.error(t("login_providers.feedback.error"));
    }
  });

  const changeProvider = (nextProvider: LoginProviderCode) => {
    if (editing) return;
    const current = form.getValues();
    form.reset({ ...emptyLoginProviderConfigForm(nextProvider), name: current.name, description: current.description }, { keepDirty: true });
  };

  const updateProviderFilter = (value?: LoginProviderCode) => {
    setFilters((current) => {
      const next: LoginProviderConfigFilters = { ...current, page: 1 };
      if (value) next.provider_code = value;
      else delete next.provider_code;
      return next;
    });
  };

  const updateStatusFilter = (value?: LoginProviderConfigStatus) => {
    setFilters((current) => {
      const next: LoginProviderConfigFilters = { ...current, page: 1 };
      if (value) next.status = value;
      else delete next.status;
      return next;
    });
  };

  const openSecret = (item: LoginProviderConfig) => {
    const values = Object.fromEntries(loginProviderDefinitions[item.provider_code].secretFields.map((secretField) => [secretField.name, ""]));
    secretForm.reset({ values });
    setSecretEditor(item);
  };

  const closeSecret = () => {
    secretForm.reset({ values: {} });
    setSecretEditor(null);
  };

  const rotateSecret = secretForm.handleSubmit(async (values) => {
    if (!secretEditor) return;
    secretForm.clearErrors();
    const issues = validateSecretValues(secretEditor.provider_code, values.values);
    for (const issue of issues) secretForm.setError(`values.${issue.field}`, { message: t(issue.key) });
    if (issues.length > 0) return;
    try {
      await mutations.rotateSecret.mutateAsync({ id: secretEditor.id, input: { values: values.values, lock_version: secretEditor.lock_version } });
      messageApi.success(t("login_providers.feedback.secret_rotated"));
      closeSecret();
    } catch (error) {
      messageApi.error(t(error instanceof ApiError && error.status === 409 ? "login_providers.feedback.conflict" : "login_providers.feedback.error"));
    }
  });

  const transition = (item: LoginProviderConfig, action: "preflight" | "activate" | "disable") => {
    const execute = async () => {
      try {
        await mutations.transition.mutateAsync({ id: item.id, transition: action, lockVersion: item.lock_version });
        messageApi.success(t(`login_providers.feedback.${action}`));
      } catch (error) {
        messageApi.error(t(error instanceof ApiError && error.status === 409 ? "login_providers.feedback.conflict" : "login_providers.feedback.error"));
      }
    };
    if (action === "preflight") {
      void execute();
      return;
    }
    Modal.confirm({
      title: t(`login_providers.actions.${action}`),
      content: t(`login_providers.confirm.${action}`, { name: item.name, count: item.binding_count }),
      okText: t(`login_providers.actions.${action}`),
      ...(action === "disable" ? { okButtonProps: { danger: true } } : {}),
      cancelText: t("common.actions.cancel"),
      onOk: execute,
    });
  };

  const remove = (item: LoginProviderConfig) => {
    Modal.confirm({
      title: t("login_providers.actions.delete"),
      content: t("login_providers.confirm.delete", { name: item.name }),
      okText: t("common.actions.delete"),
      okButtonProps: { danger: true },
      cancelText: t("common.actions.cancel"),
      onOk: async () => {
        try {
          await mutations.remove.mutateAsync({ id: item.id, lockVersion: item.lock_version });
          messageApi.success(t("login_providers.feedback.deleted"));
        } catch (error) {
          messageApi.error(t(error instanceof ApiError && error.status === 409 ? "login_providers.feedback.conflict" : "login_providers.feedback.error"));
        }
      },
    });
  };

  const configStatus = (item: LoginProviderConfig) => <Tag className={item.status === "active" ? "ak-status-success" : item.status === "draft" ? "ak-status-warning" : "ak-status-neutral"}>{t(`login_providers.status.${item.status}`)}</Tag>;
  const preflightStatus = (item: LoginProviderConfig) => <Tag className={item.last_preflight_status === "ready" ? "ak-status-success" : item.last_preflight_status === "failed" ? "ak-status-error" : "ak-status-neutral"}>{t(`login_providers.preflight.${item.last_preflight_status ?? "not_run"}`)}</Tag>;

  const actions = (item: LoginProviderConfig) => {
    const itemDefinition = loginProviderDefinitions[item.provider_code];
    const availableActions = loginProviderConfigActions(item, permissions);
    return <Space wrap size="small">
      {availableActions.has("edit") ? <Button icon={<EditOutlined />} size="small" onClick={() => { openEditor(item); }}>{t("common.actions.edit")}</Button> : null}
      {availableActions.has("rotate_secret") ? <Button icon={<KeyOutlined />} size="small" onClick={() => { openSecret(item); }}>{t("login_providers.actions.rotate_secret")}</Button> : null}
      {availableActions.has("preflight") ? <Button disabled={itemDefinition.secretFields.length > 0 && !item.has_secret} icon={<ExperimentOutlined />} loading={mutations.transition.isPending && mutations.transition.variables.id === item.id && mutations.transition.variables.transition === "preflight"} size="small" onClick={() => { transition(item, "preflight"); }}>{t("login_providers.actions.preflight")}</Button> : null}
      {availableActions.has("activate") ? <Button icon={<CheckCircleOutlined />} size="small" type="primary" onClick={() => { transition(item, "activate"); }}>{t("login_providers.actions.activate")}</Button> : null}
      {availableActions.has("disable") ? <Button icon={<StopOutlined />} size="small" onClick={() => { transition(item, "disable"); }}>{t("login_providers.actions.disable")}</Button> : null}
      {availableActions.has("delete") ? <Button danger disabled={item.binding_count > 0} icon={<DeleteOutlined />} size="small" onClick={() => { remove(item); }}>{t("common.actions.delete")}</Button> : null}
    </Space>;
  };

  const columns: TableColumnsType<LoginProviderConfig> = [
    { title: t("login_providers.columns.name"), dataIndex: "name", render: (value: string, item) => <Space><LoginProviderIcon provider={item.provider_code} /><strong>{value}</strong></Space> },
    { title: t("login_providers.columns.provider"), render: (_, item) => t(`login_providers.provider.${item.provider_code}`) },
    { title: t("login_providers.columns.client_id"), dataIndex: "external_client_id", responsive: ["lg"], render: (value: string) => <code className="ak-login-provider-client-id">{value}</code> },
    { title: t("login_providers.columns.status"), render: (_, item) => configStatus(item) },
    { title: t("login_providers.columns.preflight"), responsive: ["md"], render: (_, item) => preflightStatus(item) },
    { title: t("login_providers.columns.bindings"), dataIndex: "binding_count", responsive: ["lg"] },
    { title: t("login_providers.columns.updated_at"), responsive: ["xl"], render: (_, item) => new Intl.DateTimeFormat(i18n.resolvedLanguage === "en-US" ? "en-US" : "zh-CN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(item.updated_at)) },
    { title: t("login_providers.columns.actions"), width: 450, render: (_, item) => actions(item) },
  ];

  const renderPublicField = (fieldDefinition: LoginProviderFieldDefinition) => <Controller
    control={form.control}
    key={fieldDefinition.name}
    name={publicFieldPath(fieldDefinition.name)}
    render={({ field: controlledField, fieldState }) => <Form.Item
      help={fieldState.error?.message ?? t(fieldDefinition.helpKey)}
      label={t(fieldDefinition.labelKey)}
      required={fieldDefinition.required}
      validateStatus={fieldState.error ? "error" : ""}
    >
      {fieldDefinition.valueType === "string_array" ? <Input.TextArea
        aria-label={t(fieldDefinition.labelKey)}
        autoComplete="off"
        onChange={(event) => { controlledField.onChange(event.target.value.split(/\r?\n/).map((value) => value.trim()).filter(Boolean)); }}
        placeholder={t("login_providers.placeholders.one_per_line")}
        rows={4}
        value={Array.isArray(controlledField.value) ? controlledField.value.join("\n") : ""}
      /> : <Input
        aria-label={t(fieldDefinition.labelKey)}
        autoComplete="off"
        maxLength={fieldDefinition.maxLength}
        onBlur={controlledField.onBlur}
        onChange={controlledField.onChange}
        ref={controlledField.ref}
        value={typeof controlledField.value === "string" ? controlledField.value : ""}
      />}
    </Form.Item>}
  />;

  const items = configs.data?.items ?? [];
  const loadError = catalog.isError || configs.isError;
  const catalogMismatch = (catalog.data ?? []).some((item) => !catalogMatchesDefinition(item)) || (catalog.data !== undefined && supportedProviders.length !== loginProviderCodes.length);
  return <div className="ak-page-container ak-login-providers-page">
    {holder}
    <header className="ak-page-heading ak-login-providers-heading">
      <div><Typography.Title level={1}>{t("login_providers.title")}</Typography.Title><Typography.Paragraph type="secondary">{t("login_providers.description")}</Typography.Paragraph></div>
      <Space align="center" size="small">
        <LoginProviderApplicationGuide />
        {permissions.has("sys.login_provider_config.create") ? <Button disabled={catalogMismatch || supportedProviders.length === 0} icon={<PlusOutlined />} onClick={() => { openEditor(null); }} type="primary">{t("login_providers.actions.create")}</Button> : null}
      </Space>
    </header>
    <Space orientation="vertical" size="large" style={{ width: "100%" }}>
      <Alert description={t("login_providers.security.description")} showIcon title={t("login_providers.security.title")} type="info" />
      {catalogMismatch ? <Alert description={t("login_providers.feedback.catalog_mismatch_description")} role="alert" showIcon title={t("login_providers.feedback.catalog_mismatch")} type="error" /> : null}
      {loadError ? <Alert action={<Button onClick={() => { void catalog.refetch(); void configs.refetch(); }}>{t("common.actions.retry")}</Button>} role="alert" showIcon title={t("login_providers.feedback.load_error")} type="error" /> : null}
      <Card>
        <Row gutter={[12, 12]}>
          <Col xs={24} md={12}><Input.Search allowClear aria-label={t("login_providers.filters.query")} onChange={(event) => { setFilters((current) => ({ ...current, q: event.target.value, page: 1 })); }} placeholder={t("login_providers.filters.query")} value={filters.q} /></Col>
          <Col xs={24} md={6}><Select<LoginProviderCode> allowClear aria-label={t("login_providers.filters.provider")} onChange={updateProviderFilter} options={loginProviderCodes.map((code) => ({ value: code, label: t(`login_providers.provider.${code}`) }))} placeholder={t("login_providers.filters.provider")} style={{ width: "100%" }} value={filters.provider_code ?? null} /></Col>
          <Col xs={24} md={6}><Select<LoginProviderConfigStatus> allowClear aria-label={t("login_providers.filters.status")} onChange={updateStatusFilter} options={(["draft", "active", "disabled"] as const).map((status) => ({ value: status, label: t(`login_providers.status.${status}`) }))} placeholder={t("login_providers.filters.status")} style={{ width: "100%" }} value={filters.status ?? null} /></Col>
        </Row>
      </Card>
      {screens.md ? <div className="ak-table-scroll" role="region" tabIndex={0} aria-label={t("login_providers.title")}><Table columns={columns} dataSource={items} loading={catalog.isPending || configs.isPending} locale={{ emptyText: t("login_providers.empty") }} pagination={{ current: filters.page, pageSize: filters.page_size, total: configs.data?.total ?? 0, onChange: (page) => { setFilters((current) => ({ ...current, page })); } }} rowKey="id" scroll={{ x: 1220 }} /></div> : <List dataSource={items} loading={catalog.isPending || configs.isPending} locale={{ emptyText: t("login_providers.empty") }} renderItem={(item) => <List.Item><Card className="ak-login-provider-card" extra={configStatus(item)} title={<Space><LoginProviderIcon provider={item.provider_code} />{item.name}</Space>}><Space orientation="vertical" size="small" style={{ width: "100%" }}><Typography.Text type="secondary">{t(`login_providers.provider.${item.provider_code}`)}</Typography.Text><code className="ak-login-provider-client-id">{item.external_client_id}</code><Space wrap>{preflightStatus(item)}<Typography.Text type="secondary">{t("login_providers.binding_count", { count: item.binding_count })}</Typography.Text></Space>{actions(item)}</Space></Card></List.Item>} />}
    </Space>
    <Drawer destroyOnHidden extra={permissions.has(editing ? "sys.login_provider_config.update" : "sys.login_provider_config.create") ? <Button loading={mutations.create.isPending || mutations.update.isPending} onClick={() => { void save(); }} type="primary">{t("common.actions.save")}</Button> : null} onClose={closeEditor} open={drawerOpen} size={screens.md ? "large" : "100%"} title={t(editing ? "login_providers.editor.edit_title" : "login_providers.editor.create_title")}>
      {conflict ? <Alert action={<Button loading={conflictRefreshing} onClick={() => { void refreshConflict(); }}>{t("common.actions.refresh")}</Button>} className="ak-login-provider-drawer-alert" description={t("login_providers.feedback.conflict_description")} role="alert" showIcon title={t("login_providers.feedback.conflict")} type="warning" /> : null}
      <Form layout="vertical">
        <Card size="small" title={t("login_providers.editor.basic")}>
          <Controller control={form.control} name="name" render={({ field, fieldState }) => <Form.Item help={fieldState.error?.message} label={t("login_providers.field.name")} required validateStatus={fieldState.error ? "error" : ""}><Input {...field} aria-label={t("login_providers.field.name")} maxLength={160} /></Form.Item>} />
          <Controller control={form.control} name="description" render={({ field, fieldState }) => <Form.Item help={fieldState.error?.message} label={t("login_providers.field.description")} validateStatus={fieldState.error ? "error" : ""}><Input.TextArea {...field} aria-label={t("login_providers.field.description")} maxLength={2000} rows={3} showCount /></Form.Item>} />
          <Controller control={form.control} name="provider_code" render={({ field }) => <Form.Item label={t("login_providers.field.provider")} required><Select {...field} aria-label={t("login_providers.field.provider")} disabled={editing !== null} onChange={changeProvider} options={supportedProviders.map((code) => ({ value: code, label: t(`login_providers.provider.${code}`) }))} /></Form.Item>} />
          <Controller control={form.control} name="external_client_id" render={({ field, fieldState }) => <Form.Item help={fieldState.error?.message ?? t(definition.externalClientHelpKey)} label={t(definition.externalClientLabelKey)} required validateStatus={fieldState.error ? "error" : ""}><Input {...field} aria-label={t(definition.externalClientLabelKey)} autoComplete="off" maxLength={255} /></Form.Item>} />
          {editing?.callback_uri ? <Form.Item label={t("login_providers.field.callback_uri")}><Input aria-label={t("login_providers.field.callback_uri")} readOnly value={editing.callback_uri} /></Form.Item> : null}
        </Card>
        <Card className="ak-login-provider-fields-card" size="small" title={t("login_providers.editor.provider_fields")}>
          {definition.publicFields.map(renderPublicField)}
        </Card>
        {editing ? <Alert className="ak-login-provider-drawer-alert" description={t("login_providers.editor.secret_description")} showIcon title={t("login_providers.editor.secret_title")} type={definition.secretFields.length === 0 || editing.has_secret ? "success" : "warning"} /> : null}
      </Form>
    </Drawer>
    <Modal confirmLoading={mutations.rotateSecret.isPending} destroyOnHidden okText={t("login_providers.actions.rotate_secret")} onCancel={closeSecret} onOk={() => { void rotateSecret(); }} open={secretEditor !== null} title={secretEditor ? t("login_providers.secret.title", { provider: t(`login_providers.provider.${secretEditor.provider_code}`) }) : ""}>
      <Alert description={t("login_providers.secret.description")} showIcon title={t("login_providers.secret.write_only")} type="warning" />
      <Form layout="vertical" style={{ marginTop: 16 }}>{secretEditor ? loginProviderDefinitions[secretEditor.provider_code].secretFields.map((secretDefinition) => <Controller control={secretForm.control} key={secretDefinition.name} name={`values.${secretDefinition.name}`} render={({ field, fieldState }) => <Form.Item help={fieldState.error?.message ?? t(secretDefinition.helpKey)} label={t(secretDefinition.labelKey)} required validateStatus={fieldState.error ? "error" : ""}>{secretDefinition.valueType === "pem" ? <Input.TextArea {...field} aria-label={t(secretDefinition.labelKey)} autoComplete="new-password" rows={8} value={field.value} /> : <Input.Password {...field} aria-label={t(secretDefinition.labelKey)} autoComplete="new-password" value={field.value} />}</Form.Item>} />) : null}</Form>
    </Modal>
  </div>;
}
