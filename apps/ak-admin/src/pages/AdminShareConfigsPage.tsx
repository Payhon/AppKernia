import {
  Alert,
  Button,
  Card,
  Checkbox,
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
import {
  CheckCircleOutlined,
  EditOutlined,
  PlusOutlined,
  StopOutlined,
} from "@ant-design/icons";
import { Controller, useForm } from "react-hook-form";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { z } from "zod";

import type {
  AdminShareConfig,
  AdminShareConfigInput,
} from "../generated/api/types.gen";
import {
  useShareConfigMutations,
  useShareConfigs,
} from "../features/share-configs/hooks";
import { emptyWechatShareConfig } from "../features/share-configs/model";
import type { ShareConfigFilters } from "../features/share-configs/model";
import { WechatShareApplicationGuide } from "../components/WechatShareApplicationGuide";

interface FormValues {
  android_enabled: boolean;
  android_package: string;
  android_signature: string;
  description: string;
  external_app_id: string;
  harmony_bundle_name: string;
  harmony_enabled: boolean;
  ios_bundle_id: string;
  ios_enabled: boolean;
  ios_universal_link: string;
  name: string;
}

const schema = z
  .object({
    name: z.string().trim().min(1).max(160),
    description: z.string().max(1000),
    external_app_id: z.string().regex(/^wx[A-Za-z0-9]{16}$/),
    android_enabled: z.boolean(),
    android_package: z.string().max(255),
    android_signature: z.string().max(256),
    ios_enabled: z.boolean(),
    ios_bundle_id: z.string().max(255),
    ios_universal_link: z.string().max(2048),
    harmony_enabled: z.boolean(),
    harmony_bundle_name: z.string().max(255),
  })
  .superRefine((value, ctx) => {
    if (value.android_enabled && (!value.android_package || !value.android_signature))
      ctx.addIssue({ code: "custom", path: ["android_package"] });
    if (value.ios_enabled && (!value.ios_bundle_id || !value.ios_universal_link.startsWith("https://")))
      ctx.addIssue({ code: "custom", path: ["ios_universal_link"] });
    if (value.harmony_enabled && !value.harmony_bundle_name)
      ctx.addIssue({ code: "custom", path: ["harmony_bundle_name"] });
  });

const defaults = (): FormValues => ({
  name: "",
  description: "",
  external_app_id: "",
  android_enabled: false,
  android_package: "",
  android_signature: "",
  ios_enabled: false,
  ios_bundle_id: "",
  ios_universal_link: "",
  harmony_enabled: false,
  harmony_bundle_name: "",
});

function fromConfig(item: AdminShareConfig): FormValues {
  return {
    name: item.name,
    description: item.description,
    external_app_id: item.external_app_id,
    android_enabled: item.public_config.android.enabled,
    android_package: item.public_config.android.package_name ?? "",
    android_signature: item.public_config.android.signature ?? "",
    ios_enabled: item.public_config.ios.enabled,
    ios_bundle_id: item.public_config.ios.bundle_id ?? "",
    ios_universal_link: item.public_config.ios.universal_link ?? "",
    harmony_enabled: item.public_config.harmony.enabled,
    harmony_bundle_name: item.public_config.harmony.bundle_name ?? "",
  };
}

function toInput(value: FormValues, item: AdminShareConfig | null): AdminShareConfigInput {
  const input = emptyWechatShareConfig();
  return {
    ...input,
    name: value.name.trim(),
    description: value.description.trim(),
    external_app_id: value.external_app_id.trim(),
    ...(item ? { lock_version: item.lock_version } : {}),
    public_config: {
      android: {
        enabled: value.android_enabled,
        ...(value.android_package.trim() ? { package_name: value.android_package.trim() } : {}),
        ...(value.android_signature.trim() ? { signature: value.android_signature.trim() } : {}),
      },
      ios: {
        enabled: value.ios_enabled,
        ...(value.ios_bundle_id.trim() ? { bundle_id: value.ios_bundle_id.trim() } : {}),
        ...(value.ios_universal_link.trim() ? { universal_link: value.ios_universal_link.trim() } : {}),
      },
      harmony: {
        enabled: value.harmony_enabled,
        ...(value.harmony_bundle_name.trim() ? { bundle_name: value.harmony_bundle_name.trim() } : {}),
      },
    },
  };
}

function readFilters(): ShareConfigFilters {
  const params = new URLSearchParams(location.search);
  const status = params.get("status");
  const parsedStatus =
    status === "draft" || status === "active" || status === "disabled"
      ? status
      : null;
  return {
    q: params.get("q") ?? "",
    provider_code: "wechat" as const,
    ...(parsedStatus ? { status: parsedStatus } : {}),
    page: Math.max(1, Number(params.get("page")) || 1),
    page_size: 20,
  };
}

export function AdminShareConfigsPage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const [messageApi, holder] = message.useMessage();
  const [filters, setFilters] = useState(readFilters);
  const [editing, setEditing] = useState<AdminShareConfig | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const query = useShareConfigs(filters);
  const mutations = useShareConfigMutations();
  const { control, handleSubmit, reset, setError, watch, formState } =
    useForm<FormValues>({ defaultValues: defaults() });
  const changeStatusFilter = (status: "draft" | "active" | "disabled" | undefined) => {
    setFilters((old) => {
      const rest = { ...old };
      delete rest.status;
      return status ? { ...rest, status, page: 1 } : { ...rest, page: 1 };
    });
  };

  useEffect(() => {
    const params = new URLSearchParams();
    if (filters.q) params.set("q", filters.q);
    if (filters.status) params.set("status", filters.status);
    if (filters.page > 1) params.set("page", String(filters.page));
    history.replaceState(null, "", `${location.pathname}${params.size ? `?${params}` : ""}`);
  }, [filters]);

  const open = (item: AdminShareConfig | null) => {
    setEditing(item);
    reset(item ? fromConfig(item) : defaults());
    setDrawerOpen(true);
  };

  const fail = () => messageApi.error(t("share_configs.feedback.error"));
  const submit = handleSubmit(async (values) => {
    const parsed = schema.safeParse(values);
    if (!parsed.success) {
      for (const issue of parsed.error.issues)
        setError(issue.path[0] as keyof FormValues, { message: t("validation.invalid", { ns: "validation" }) });
      return;
    }
    try {
      if (editing)
        await mutations.update.mutateAsync({ id: editing.id, input: toInput(parsed.data, editing) });
      else await mutations.create.mutateAsync(toInput(parsed.data, null));
      messageApi.success(t("share_configs.feedback.saved"));
      setDrawerOpen(false);
    } catch {
      fail();
    }
  });

  const transition = (item: AdminShareConfig, action: "activate" | "disable") => {
    Modal.confirm({
      title: t(`share_configs.actions.${action}`),
      content: t(`share_configs.confirm.${action}`),
      onOk: async () => {
        try {
          await mutations.transition.mutateAsync({ id: item.id, transition: action, lockVersion: item.lock_version });
          messageApi.success(t(`share_configs.feedback.${action === "activate" ? "activated" : "disabled"}`));
        } catch {
          fail();
        }
      },
    });
  };

  const remove = (item: AdminShareConfig) => {
    Modal.confirm({
      title: t("share_configs.actions.delete"),
      content: t("share_configs.confirm.delete"),
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await mutations.remove.mutateAsync({ id: item.id, lockVersion: item.lock_version });
          messageApi.success(t("share_configs.feedback.deleted"));
        } catch {
          fail();
        }
      },
    });
  };

  const columns: TableColumnsType<AdminShareConfig> = useMemo(
    () => [
      { title: t("share_configs.columns.name"), dataIndex: "name" },
      { title: t("share_configs.columns.provider"), render: () => t("share_configs.provider.wechat") },
      { title: t("share_configs.columns.app_id"), dataIndex: "external_app_id" },
      { title: t("share_configs.columns.status"), render: (_: unknown, item: AdminShareConfig) => <Tag className={item.status === "active" ? "ak-status-success" : item.status === "disabled" ? "ak-status-neutral" : "ak-status-warning"}>{t(`share_configs.status.${item.status}`)}</Tag> },
      { title: t("share_configs.columns.bindings"), dataIndex: "binding_count" },
      { title: t("share_configs.columns.updated_at"), render: (_: unknown, item: AdminShareConfig) => new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(item.updated_at)) },
      {
        title: t("share_configs.columns.actions"),
        render: (_: unknown, item: AdminShareConfig) => (
          <Space wrap>
            <Button icon={<EditOutlined />} onClick={() => { open(item); }}>{t("share_configs.actions.edit")}</Button>
            {item.status !== "active" ? (
              <Button icon={<CheckCircleOutlined />} onClick={() => { transition(item, "activate"); }}>{t("share_configs.actions.activate")}</Button>
            ) : (
              <Button icon={<StopOutlined />} onClick={() => { transition(item, "disable"); }}>{t("share_configs.actions.disable")}</Button>
            )}
            <Button danger disabled={item.binding_count > 0} onClick={() => { remove(item); }}>{t("share_configs.actions.delete")}</Button>
          </Space>
        ),
      },
    ],
    [t],
  );

  const renderField = (name: keyof FormValues, label: string, password = false) => (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <Form.Item label={label} {...(fieldState.error ? { validateStatus: "error" as const, help: fieldState.error.message } : {})}>
          <Input {...field} aria-label={label} value={String(field.value)} type={password ? "password" : "text"} />
        </Form.Item>
      )}
    />
  );

  const platformSection = (platform: "android" | "ios" | "harmony") => {
    const enabledName = `${platform}_enabled` as keyof FormValues;
    const enabled = Boolean(watch(enabledName));
    return (
      <Card size="small" title={t(`share_configs.editor.${platform}`)}>
        <Controller control={control} name={enabledName} render={({ field }) => <Checkbox checked={Boolean(field.value)} onChange={(event) => { field.onChange(event.target.checked); }}>{t("share_configs.fields.platform_enabled")}</Checkbox>} />
        {enabled && (
          <div style={{ marginTop: 16 }}>
            {platform === "android" && <>{renderField("android_package", t("share_configs.fields.android_package"))}{renderField("android_signature", t("share_configs.fields.android_signature"), true)}</>}
            {platform === "ios" && <>{renderField("ios_bundle_id", t("share_configs.fields.ios_bundle_id"))}{renderField("ios_universal_link", t("share_configs.fields.ios_universal_link"))}</>}
            {platform === "harmony" && renderField("harmony_bundle_name", t("share_configs.fields.harmony_bundle_name"))}
          </div>
        )}
      </Card>
    );
  };

  const items = query.data?.items ?? [];
  return (
    <div className="ak-page-container">
      {holder}
      <header className="ak-page-heading ak-org-heading">
        <div><Typography.Title level={1}>{t("share_configs.title")}</Typography.Title><Typography.Paragraph type="secondary">{t("share_configs.description")}</Typography.Paragraph></div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { open(null); }}>{t("share_configs.actions.create")}</Button>
      </header>
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        <Alert showIcon type="info" title={t("share_configs.help.rebuild")} description={t("share_configs.help.app_secret")} />
        <Card>
          <Row gutter={[12, 12]}>
            <Col xs={24} md={12}><Input.Search allowClear aria-label={t("share_configs.filters.query")} placeholder={t("share_configs.filters.query")} value={filters.q} onChange={(event) => { setFilters((old) => ({ ...old, q: event.target.value, page: 1 })); }} /></Col>
            <Col xs={24} md={6}><Select<"draft" | "active" | "disabled"> allowClear aria-label={t("share_configs.filters.status")} style={{ width: "100%" }} placeholder={t("share_configs.filters.status")} value={filters.status ?? null} options={(["draft", "active", "disabled"] as const).map((value) => ({ value, label: t(`share_configs.status.${value}`) }))} onChange={changeStatusFilter} /></Col>
          </Row>
        </Card>
        {screens.md ? (
          <Table rowKey="id" loading={query.isLoading} columns={columns} dataSource={items} pagination={{ current: filters.page, pageSize: filters.page_size, total: query.data?.total ?? 0, onChange: (page) => { setFilters((old) => ({ ...old, page })); } }} locale={{ emptyText: t("share_configs.empty") }} scroll={{ x: 980 }} />
        ) : (
          <List loading={query.isLoading} dataSource={items} locale={{ emptyText: t("share_configs.empty") }} renderItem={(item) => <List.Item><Card style={{ width: "100%" }} title={item.name} extra={<Tag className={item.status === "active" ? "ak-status-success" : item.status === "disabled" ? "ak-status-neutral" : "ak-status-warning"}>{t(`share_configs.status.${item.status}`)}</Tag>}><Space orientation="vertical"><Typography.Text>{item.external_app_id}</Typography.Text><Typography.Text type="secondary">{t("share_configs.columns.bindings")}: {item.binding_count}</Typography.Text><Space wrap><Button onClick={() => { open(item); }}>{t("share_configs.actions.edit")}</Button>{item.status !== "active" ? <Button onClick={() => { transition(item, "activate"); }}>{t("share_configs.actions.activate")}</Button> : <Button onClick={() => { transition(item, "disable"); }}>{t("share_configs.actions.disable")}</Button>}</Space></Space></Card></List.Item>} />
        )}
      </Space>
      <Drawer open={drawerOpen} size={screens.md ? 720 : "100%"} destroyOnHidden title={<Space size="small"><span>{t(editing ? "share_configs.editor.edit_title" : "share_configs.editor.create_title")}</span><WechatShareApplicationGuide /></Space>} onClose={() => { setDrawerOpen(false); }} extra={<Button type="primary" loading={formState.isSubmitting} onClick={() => { void submit(); }}>{t("share_configs.actions.save_draft")}</Button>}>
        <Form layout="vertical">
          <Card size="small" title={t("share_configs.editor.basic")}>
            {renderField("name", t("share_configs.fields.name"))}
            <Controller control={control} name="description" render={({ field, fieldState }) => <Form.Item label={t("share_configs.fields.description")} {...(fieldState.error ? { validateStatus: "error" as const, help: fieldState.error.message } : {})}><Input.TextArea {...field} aria-label={t("share_configs.fields.description")} rows={3} /></Form.Item>} />
            {renderField("external_app_id", t("share_configs.fields.external_app_id"))}
          </Card>
          <Space orientation="vertical" size="middle" style={{ width: "100%", marginTop: 16 }}>{platformSection("android")}{platformSection("ios")}{platformSection("harmony")}</Space>
        </Form>
      </Drawer>
    </div>
  );
}
