import { CheckCircleOutlined, ExperimentOutlined, KeyOutlined, SettingOutlined, StopOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Col, Drawer, Form, Grid, Input, List, Modal, Row, Select, Space, Statistic, Table, Tag, Typography, message, type TableColumnsType } from "antd";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { AdminPushProviderCatalogItem, AdminPushProviderConfig, AdminPushTestRequest, MobilePushDevice, PushEnvironment } from "../generated/api/types.gen";
import { AppSelectionRequiredState } from "../components/AppSelectionRequiredState";
import { PushProviderApplicationGuide } from "../components/PushProviderApplicationGuide";
import { useAppScope } from "../features/apps/scope";
import { useAuthStore } from "../features/auth/store";
import { usePushChannelMutations, usePushChannels, usePushTestDevices } from "../features/push-channels/hooks";
import { fieldValuesSchema, providerState, testPushSchema, type PushConfigEditor } from "../features/push-channels/model";

type SecretEditor = { config: AdminPushProviderConfig; catalog: AdminPushProviderCatalogItem } | null;
type TestEditor = { config: AdminPushProviderConfig } | null;

const environments: PushEnvironment[] = ["development", "test", "staging", "production"];

export function AdminPushChannelsPage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const scope = useAppScope();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [environment, setEnvironment] = useState<PushEnvironment>("development");
  const [editor, setEditor] = useState<PushConfigEditor | null>(null);
  const [secretEditor, setSecretEditor] = useState<SecretEditor>(null);
  const [testEditor, setTestEditor] = useState<TestEditor>(null);
  const [publicValues, setPublicValues] = useState<Record<string, string>>({});
  const [secretValues, setSecretValues] = useState<Record<string, string>>({});
  const [testValues, setTestValues] = useState<AdminPushTestRequest>({ push_device_id: "", title: "", body: "" });
  const [messageApi, holder] = message.useMessage();
  const queries = usePushChannels(scope.appId, environment);
  const mutations = usePushChannelMutations(scope.appId, environment);
  const testDevices = usePushTestDevices(scope.appId, testEditor?.config.provider ?? null, testEditor !== null);

  useEffect(() => {
    if (!scope.appId) return;
    const params = new URLSearchParams(location.search);
    const current = params.get("environment");
    if (environments.includes(current as PushEnvironment)) setEnvironment(current as PushEnvironment);
  }, [scope.appId]);

  const configs = new Map((queries.configs.data ?? []).map((item) => [item.provider, item]));
  const openConfig = (catalog: AdminPushProviderCatalogItem) => {
    const config = configs.get(catalog.provider) ?? null;
    setPublicValues(Object.fromEntries(catalog.public_fields.map((field) => [field, config?.public_config[field] ?? ""])));
    setEditor({ catalog, config });
  };
  const saveConfig = async () => {
    if (!editor) return;
    const parsed = fieldValuesSchema(editor.catalog.public_fields).safeParse(publicValues);
    if (!parsed.success) { messageApi.error(t("push_channels.feedback.validation")); return; }
    try {
      await mutations.save.mutateAsync({ provider: editor.catalog.provider, input: { environment, config_schema_version: 1, public_config: parsed.data, ...(editor.config ? { lock_version: editor.config.lock_version } : {}) } });
      messageApi.success(t("push_channels.feedback.saved")); setEditor(null);
    } catch { messageApi.error(t("push_channels.feedback.error")); }
  };
  const openSecret = (config: AdminPushProviderConfig, catalog: AdminPushProviderCatalogItem) => { setSecretValues(Object.fromEntries(catalog.secret_fields.map((field) => [field, ""]))); setSecretEditor({ config, catalog }); };
  const saveSecret = async () => {
    if (!secretEditor) return;
    const parsed = fieldValuesSchema(secretEditor.catalog.secret_fields, 65_536).safeParse(secretValues);
    if (!parsed.success) { messageApi.error(t("push_channels.feedback.validation")); return; }
    try { await mutations.rotate.mutateAsync({ id: secretEditor.config.id, input: { lock_version: secretEditor.config.lock_version, values: parsed.data } }); messageApi.success(t("push_channels.feedback.secret_rotated")); setSecretValues({}); setSecretEditor(null); }
    catch { messageApi.error(t("push_channels.feedback.error")); }
  };
  const transition = async (config: AdminPushProviderConfig, action: "preflight" | "activate" | "disable") => {
    try { await mutations.transition.mutateAsync({ id: config.id, action, lockVersion: config.lock_version }); messageApi.success(t(`push_channels.feedback.${action}`)); }
    catch { messageApi.error(t("push_channels.feedback.error")); }
  };
  const openTest = (config: AdminPushProviderConfig) => { setTestValues({ push_device_id: "", title: t("push_channels.test.default_title"), body: t("push_channels.test.default_body") }); setTestEditor({ config }); };
  const sendTest = async () => {
    if (!testEditor) return;
    const parsed = testPushSchema.safeParse(testValues);
    if (!parsed.success) { messageApi.error(t("push_channels.feedback.validation")); return; }
    try { await mutations.test.mutateAsync({ id: testEditor.config.id, input: parsed.data }); messageApi.success(t("push_channels.feedback.test_queued")); setTestEditor(null); }
    catch { messageApi.error(t("push_channels.feedback.error")); }
  };
  const statusTag = (config: AdminPushProviderConfig | null) => {
    const state = providerState(config);
    const className = state === "active" || state === "ready" ? "ak-status-success" : state === "faulted" ? "ak-status-error" : state === "draft" ? "ak-status-warning" : "ak-status-neutral";
    return <Tag className={className}>{t(`push_channels.status.${state}`)}</Tag>;
  };
  const actions = (catalog: AdminPushProviderCatalogItem) => {
    const config = configs.get(catalog.provider) ?? null;
    return <Space wrap size="small">
      {permissions.has("notify.push_provider.manage") ? <Button icon={<SettingOutlined />} size="small" onClick={() => { openConfig(catalog); }}>{t(config ? "common.actions.edit" : "push_channels.actions.configure")}</Button> : null}
      {config && permissions.has("notify.push_provider.rotate_secret") ? <Button icon={<KeyOutlined />} size="small" onClick={() => { openSecret(config, catalog); }}>{t("push_channels.actions.rotate_secret")}</Button> : null}
      {config?.has_secret && permissions.has("notify.push.preflight") ? <Button icon={<ExperimentOutlined />} size="small" onClick={() => void transition(config, "preflight")}>{t("push_channels.actions.preflight")}</Button> : null}
      {config?.last_preflight_status === "ready" && config.status !== "active" && permissions.has("notify.push_provider.manage") ? <Button icon={<CheckCircleOutlined />} size="small" type="primary" onClick={() => void transition(config, "activate")}>{t("push_channels.actions.activate")}</Button> : null}
      {config?.status === "active" && permissions.has("notify.push_provider.manage") ? <Button icon={<StopOutlined />} size="small" onClick={() => void transition(config, "disable")}>{t("push_channels.actions.disable")}</Button> : null}
      {config?.status === "active" && permissions.has("notify.push.test") ? <Button size="small" onClick={() => { openTest(config); }}>{t("push_channels.actions.test")}</Button> : null}
    </Space>;
  };
  const columns: TableColumnsType<AdminPushProviderCatalogItem> = [
    { title: t("push_channels.columns.provider"), render: (_, item) => <strong>{t(`push_channels.provider.${item.provider}`)}</strong> },
    { title: t("push_channels.columns.platform"), render: (_, item) => item.platforms.map((value) => t(`push_channels.platform.${value}`)).join(" / ") },
    { title: t("push_channels.columns.variant"), responsive: ["md"], render: (_, item) => item.build_variants.map((value) => t(`push_channels.variant.${value}`)).join(" / ") },
    { title: t("push_channels.columns.status"), render: (_, item) => statusTag(configs.get(item.provider) ?? null) },
    { title: t("push_channels.columns.credential"), responsive: ["lg"], render: (_, item) => { const config = configs.get(item.provider); return config?.has_secret ? <span>{t("push_channels.values.configured")} · <code>{config.credential_fingerprint}</code></span> : t("push_channels.values.missing"); } },
    { title: t("push_channels.columns.actions"), width: 390, render: (_, item) => actions(item) },
  ];
  const summary = queries.summary.data?.items ?? [];
  const summaryValue = (result: string) => summary.reduce((total, item) => {
    if (result === "opened") return total + item.opened_count;
    if (result === "failed") return total + (["failed", "permanent", "auth_config_error", "unknown_after_write"].includes(item.result) ? item.count : 0);
    return total + (item.result === result ? item.count : 0);
  }, 0);
  const publicField = (field: string) => {
    const options = field === "apns_environment" ? ["sandbox", "production"] : field === "region" ? ["china", "singapore", "europe", "india", "russia"] : null;
    if (options) return <Select aria-label={t(`push_channels.field.${field}`)} value={publicValues[field] === "" ? undefined : publicValues[field]} options={options.map((value) => ({ value, label: t(`push_channels.option.${field}.${value}`) }))} onChange={(value) => { setPublicValues((current) => ({ ...current, [field]: value ?? "" })); }} />;
    return <Input aria-label={t(`push_channels.field.${field}`)} autoComplete="off" value={publicValues[field] ?? ""} onChange={(event) => { setPublicValues((current) => ({ ...current, [field]: event.target.value })); }} />;
  };

  return <div className="ak-page-container">
    {holder}
    <header className="ak-page-heading ak-push-page-heading">
      <div>
        <Typography.Title level={1}>{t("push_channels.title")}</Typography.Title>
        <Typography.Paragraph type="secondary">{t("push_channels.description")}</Typography.Paragraph>
      </div>
      <Space align="center" className="ak-push-page-heading-actions" size="small">
        <Select aria-label={t("push_channels.environment.label")} value={environment} options={environments.map((value) => ({ value, label: t(`push_channels.environment.${value}`) }))} onChange={(value) => { setEnvironment(value); const params = new URLSearchParams(location.search); params.set("environment", value); history.replaceState(null, "", `${location.pathname}?${params}`); }} />
        <PushProviderApplicationGuide />
      </Space>
    </header>
    {!scope.appId ? <AppSelectionRequiredState /> : <Space orientation="vertical" size="large" style={{ width: "100%" }}>
      <Alert showIcon type="info" title={t("push_channels.security.title")} description={t("push_channels.security.description")} />
      {queries.catalog.isError || queries.configs.isError ? <Alert showIcon type="error" title={t("push_channels.feedback.load_error")} action={<Button onClick={() => { void queries.catalog.refetch(); void queries.configs.refetch(); }}>{t("common.actions.retry")}</Button>} /> : null}
      {screens.md ? <div className="ak-table-scroll" role="region" tabIndex={0} aria-label={t("push_channels.title")}><Table rowKey="provider" loading={queries.catalog.isPending || queries.configs.isPending} columns={columns} dataSource={queries.catalog.data ?? []} pagination={false} scroll={{ x: 1180 }} /></div> : <List loading={queries.catalog.isPending || queries.configs.isPending} dataSource={queries.catalog.data ?? []} renderItem={(item) => <List.Item><Card style={{ width: "100%" }} title={t(`push_channels.provider.${item.provider}`)} extra={statusTag(configs.get(item.provider) ?? null)}><Typography.Paragraph type="secondary">{item.platforms.map((value) => t(`push_channels.platform.${value}`)).join(" / ")}</Typography.Paragraph>{actions(item)}</Card></List.Item>} />}
      <Card title={t("push_channels.summary.title")}><Row gutter={[16, 16]}>{["accepted", "failed", "invalid_token", "opened"].map((result) => <Col xs={12} lg={6} key={result}><Statistic title={t(`push_channels.result.${result}`)} value={summaryValue(result)} /></Col>)}</Row><Typography.Paragraph type="secondary" style={{ marginTop: 16 }}>{t("push_channels.summary.note", { days: queries.summary.data?.window_days ?? 30 })}</Typography.Paragraph></Card>
    </Space>}
    <Drawer destroyOnHidden open={editor !== null} size={screens.md ? "large" : "100%"} title={editor ? t("push_channels.editor.title", { provider: t(`push_channels.provider.${editor.catalog.provider}`) }) : ""} onClose={() => { setEditor(null); }} extra={<Button type="primary" loading={mutations.save.isPending} onClick={() => void saveConfig()}>{t("common.actions.save")}</Button>}>
      <Form layout="vertical">{editor?.catalog.public_fields.map((field) => <Form.Item key={field} label={t(`push_channels.field.${field}`)} required>{publicField(field)}</Form.Item>)}</Form>
    </Drawer>
    <Modal destroyOnHidden open={secretEditor !== null} title={t("push_channels.secret.title")} okText={t("push_channels.actions.rotate_secret")} confirmLoading={mutations.rotate.isPending} onCancel={() => { setSecretValues({}); setSecretEditor(null); }} onOk={() => void saveSecret()}>
      <Alert showIcon type="warning" title={t("push_channels.secret.write_only")} />
      <Form layout="vertical" style={{ marginTop: 16 }}>{secretEditor?.catalog.secret_fields.map((field) => <Form.Item key={field} label={t(`push_channels.field.${field}`)} required><Input.Password aria-label={t(`push_channels.field.${field}`)} autoComplete="new-password" value={secretValues[field] ?? ""} onChange={(event) => { setSecretValues((current) => ({ ...current, [field]: event.target.value })); }} /></Form.Item>)}</Form>
    </Modal>
    <Modal destroyOnHidden open={testEditor !== null} title={t("push_channels.test.title")} okText={t("push_channels.actions.test")} confirmLoading={mutations.test.isPending} onCancel={() => { setTestEditor(null); }} onOk={() => void sendTest()}>
      <Form layout="vertical"><Form.Item label={t("push_channels.test.device")} required><Select aria-label={t("push_channels.test.device")} showSearch={{ optionFilterProp: "label" }} loading={testDevices.isPending} value={testValues.push_device_id === "" ? undefined : testValues.push_device_id} options={(testDevices.data ?? []).map((item: MobilePushDevice) => ({ value: item.id, label: `${t(`push_channels.provider.${item.provider}`)} · ${t(`push_channels.platform.${item.platform}`)} · …${item.id.slice(-8)}` }))} onChange={(value) => { setTestValues((current) => ({ ...current, push_device_id: value ?? "" })); }} /></Form.Item><Form.Item label={t("push_channels.test.message_title")} required><Input aria-label={t("push_channels.test.message_title")} value={testValues.title} maxLength={64} showCount onChange={(event) => { setTestValues((current) => ({ ...current, title: event.target.value })); }} /></Form.Item><Form.Item label={t("push_channels.test.body")} required><Input.TextArea aria-label={t("push_channels.test.body")} value={testValues.body} maxLength={180} showCount rows={4} onChange={(event) => { setTestValues((current) => ({ ...current, body: event.target.value })); }} /></Form.Item>{testDevices.isError ? <Alert showIcon type="error" title={t("push_channels.feedback.devices_error")} /> : null}{!testDevices.isPending && (testDevices.data?.length ?? 0) === 0 ? <Alert showIcon type="warning" title={t("push_channels.test.no_device")} /> : null}</Form>
    </Modal>
  </div>;
}
