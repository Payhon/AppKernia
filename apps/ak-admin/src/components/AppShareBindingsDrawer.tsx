import {
  Alert,
  Button,
  Checkbox,
  Descriptions,
  Drawer,
  Form,
  Grid,
  Input,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
  message,
} from "antd";
import { Controller, useForm } from "react-hook-form";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import type {
  AdminShareBindingInput,
  AdminSharePreflight,
} from "../generated/api/types.gen";
import {
  useAppShareBindingMutations,
  useAppShareBindings,
  useShareConfigs,
} from "../features/share-configs/hooks";

interface Props {
  appId: string | null;
  appName?: string | undefined;
  onClose: () => void;
}

const bindingDefaults = (): AdminShareBindingInput => ({
  share_config_id: "",
  enabled: true,
  scenes: ["session", "timeline", "favorite"],
  share_origin: "",
  fallback_mode: "system",
});

export function AppShareBindingsDrawer({ appId, appName, onClose }: Props) {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const [messageApi, holder] = message.useMessage();
  const [preflight, setPreflight] = useState<AdminSharePreflight | null>(null);
  const bindings = useAppShareBindings(appId);
  const configs = useShareConfigs({ provider_code: "wechat", status: "active", page: 1, page_size: 100 });
  const mutations = useAppShareBindingMutations(appId);
  const form = useForm<AdminShareBindingInput>({ defaultValues: bindingDefaults() });
  const current = bindings.data?.[0];
  const issueText = (issue: string) => {
    const keyByIssue: Record<string, string> = {
      "share.config.not_active": "share_configs.binding.issue_config_not_active",
      "share.config.invalid": "share_configs.binding.issue_config_invalid",
      "share.config.platform_incomplete": "share_configs.binding.issue_platform_incomplete",
      "share.origin.invalid": "share_configs.binding.issue_origin_invalid",
    };
    return t(keyByIssue[issue] ?? "share_configs.binding.issue_unknown");
  };

  useEffect(() => {
    if (!appId) return;
    setPreflight(null);
    form.reset(
      current
        ? {
            share_config_id: current.share_config_id,
            enabled: current.enabled,
            scenes: current.scenes,
            share_origin: current.share_origin,
            fallback_mode: "system",
            lock_version: current.lock_version,
          }
        : bindingDefaults(),
    );
  }, [appId, current?.id, current?.lock_version]);

  const options = useMemo(
    () =>
      (configs.data?.items ?? []).map((item) => ({
        value: item.id,
        label: `${item.name} · ${item.external_app_id}`,
      })),
    [configs.data?.items],
  );

  const executePreflight = form.handleSubmit(async (input) => {
    try {
      const result = await mutations.preflight.mutateAsync(input);
      setPreflight(result);
      messageApi[result.ready ? "success" : "warning"](
        t(
          result.ready
            ? "share_configs.binding.preflight_ready"
            : "share_configs.binding.preflight_failed",
        ),
      );
    } catch {
      messageApi.error(t("share_configs.feedback.error"));
    }
  });

  const save = form.handleSubmit(async (input) => {
    try {
      const result = await mutations.preflight.mutateAsync(input);
      setPreflight(result);
      if (!result.ready) {
        messageApi.warning(t("share_configs.binding.preflight_failed"));
        return;
      }
      await mutations.save.mutateAsync(input);
      messageApi.success(t("share_configs.feedback.bound"));
    } catch {
      messageApi.error(t("share_configs.feedback.error"));
    }
  });

  const remove = async () => {
    if (!current) return;
    try {
      await mutations.remove.mutateAsync(current.lock_version);
      messageApi.success(t("share_configs.feedback.unbound"));
      setPreflight(null);
    } catch {
      messageApi.error(t("share_configs.feedback.error"));
    }
  };

  return (
    <Drawer
      destroyOnHidden
      open={Boolean(appId)}
      onClose={onClose}
      size={screens.md ? 640 : "100%"}
      title={`${t("share_configs.binding.title")} · ${appName ?? ""}`}
      extra={
        <Space wrap>
          <Button loading={mutations.preflight.isPending} onClick={() => void executePreflight()}>
            {t("share_configs.actions.preflight")}
          </Button>
          <Button type="primary" loading={mutations.save.isPending} onClick={() => void save()}>
            {t("common.actions.save")}
          </Button>
        </Space>
      }
    >
      {holder}
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        <Alert showIcon type="info" title={t("share_configs.help.rebuild")} description={t("share_configs.binding.description")} />
        {options.length === 0 && !configs.isLoading ? <Alert showIcon type="warning" title={t("share_configs.binding.no_active_config")} /> : null}
        <Form layout="vertical">
          <Form.Item label={t("share_configs.binding.config")} required>
            <Controller control={form.control} name="share_config_id" rules={{ required: true }} render={({ field }) => <Select {...field} aria-label={t("share_configs.binding.config")} loading={configs.isLoading} options={options} />} />
          </Form.Item>
          <Form.Item label={t("share_configs.binding.enabled")}>
            <Controller control={form.control} name="enabled" render={({ field }) => <Switch aria-label={t("share_configs.binding.enabled")} checked={field.value} onChange={field.onChange} />} />
          </Form.Item>
          <Form.Item label={t("share_configs.binding.scenes")} required>
            <Controller control={form.control} name="scenes" rules={{ required: true }} render={({ field }) => <Checkbox.Group {...field} aria-label={t("share_configs.binding.scenes")} options={(["session", "timeline", "favorite"] as const).map((scene) => ({ value: scene, label: t(`share_configs.binding.scene_${scene}`) }))} />} />
          </Form.Item>
          <Form.Item label={t("share_configs.binding.share_origin")} required>
            <Controller control={form.control} name="share_origin" rules={{ required: true, pattern: /^https:\/\/[A-Za-z0-9.-]+(?::[0-9]+)?$/ }} render={({ field, fieldState }) => <><Input {...field} aria-label={t("share_configs.binding.share_origin")} placeholder="https://share.example.com" {...(fieldState.error ? { status: "error" as const } : {})} />{fieldState.error ? <Typography.Text type="danger">{t("validation.invalid")}</Typography.Text> : null}</>} />
          </Form.Item>
          <Form.Item label={t("share_configs.binding.fallback")}>
            <Controller control={form.control} name="fallback_mode" render={({ field }) => <Select {...field} aria-label={t("share_configs.binding.fallback")} options={[{ value: "system", label: t("share_configs.binding.fallback_system") }]} />} />
          </Form.Item>
        </Form>
        {preflight ? (
          <Alert
            showIcon
            type={preflight.ready ? "success" : "error"}
            title={t(preflight.ready ? "share_configs.binding.preflight_ready" : "share_configs.binding.preflight_failed")}
            description={preflight.issues.length ? preflight.issues.map(issueText).join(", ") : t("share_configs.binding.build_pending")}
          />
        ) : null}
        {current ? (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label={t("share_configs.columns.status")}><Tag className={current.config_status === "active" ? "ak-status-success" : current.config_status === "disabled" ? "ak-status-neutral" : "ak-status-warning"}>{t(`share_configs.status.${current.config_status}`)}</Tag></Descriptions.Item>
            <Descriptions.Item label={t("share_configs.binding.fallback")}>{t("share_configs.binding.system_fallback")}</Descriptions.Item>
          </Descriptions>
        ) : null}
        {current ? <Button danger loading={mutations.remove.isPending} onClick={() => void remove()}>{t("share_configs.actions.remove_binding")}</Button> : null}
      </Space>
    </Drawer>
  );
}
