import { DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import { Alert, Button, Form, Input, Space, Switch, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { useAppScannerConfig, useAppScannerConfigMutation } from "../features/apps/hooks";
import { ApiError } from "../shared/api/error";

interface Props {
  appId: string;
  canUpdate: boolean;
  onDirtyChange: (dirty: boolean) => void;
}

interface Snapshot {
  enabled: boolean;
  patterns: string[];
  lockVersion: number;
}

const hostLabel = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

export function normalizeScannerHostPattern(raw: string): string | null {
  let value = raw.trim().toLowerCase().replace(/\.$/, "");
  const wildcard = value.startsWith("*.");
  if (wildcard) value = value.slice(2);
  if (!value || value.length > 253 || value === "localhost" || value.endsWith(".localhost") || value.endsWith(".local") || /^\d+(?:\.\d+){3}$/.test(value)) return null;
  const labels = value.split(".");
  if (labels.length < 2 || labels.some((label) => !hostLabel.test(label))) return null;
  return wildcard ? `*.${value}` : value;
}

export function canonicalScannerHostPatterns(values: string[]): string[] {
  return [...new Set(values.map(normalizeScannerHostPattern).filter((value): value is string => value !== null))].sort();
}

export function scannerHostInputRowForServerIndex(values: string[], canonicalValues: string[], serverIndex: number): number {
  const rejected = canonicalValues[serverIndex];
  if (!rejected) return serverIndex;
  const inputIndex = values.findIndex((value) => normalizeScannerHostPattern(value) === rejected);
  return inputIndex >= 0 ? inputIndex : serverIndex;
}

function samePatterns(left: string[], right: string[]) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

export function AppScannerConfigurationPanel({ appId, canUpdate, onDirtyChange }: Props) {
  const { t } = useTranslation();
  const [messageApi, holder] = message.useMessage();
  const query = useAppScannerConfig(appId);
  const mutation = useAppScannerConfigMutation(appId);
  const [enabled, setEnabled] = useState(false);
  const [patterns, setPatterns] = useState<string[]>([]);
  const [initial, setInitial] = useState<Snapshot>({ enabled: false, patterns: [], lockVersion: 0 });
  const [serverInvalidRow, setServerInvalidRow] = useState<number | null>(null);

  useEffect(() => {
    if (!query.data) return;
    const next = { enabled: query.data.webview_enabled, patterns: [...query.data.allowed_host_patterns], lockVersion: query.data.lock_version };
    setEnabled(next.enabled);
    setPatterns(next.patterns);
    setInitial(next);
    setServerInvalidRow(null);
  }, [appId, query.data?.lock_version]);

  const normalized = useMemo(() => canonicalScannerHostPatterns(patterns), [patterns]);
  const invalidRows = useMemo(() => patterns.map((value) => value.trim() !== "" && normalizeScannerHostPattern(value) === null), [patterns]);
  const dirty = enabled !== initial.enabled || !samePatterns(normalized, initial.patterns);
  const invalid = invalidRows.some(Boolean) || (enabled && normalized.length === 0);

  useEffect(() => { onDirtyChange(dirty); }, [dirty, onDirtyChange]);

  const save = async () => {
    if (!canUpdate || invalid) return;
    setServerInvalidRow(null);
    try {
      const saved = await mutation.mutateAsync({ webview_enabled: enabled, allowed_host_patterns: normalized, lock_version: initial.lockVersion });
      const next = { enabled: saved.webview_enabled, patterns: [...saved.allowed_host_patterns], lockVersion: saved.lock_version };
      setEnabled(next.enabled);
      setPatterns(next.patterns);
      setInitial(next);
      messageApi.success(t("apps.client_config.scanner.saved"));
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        messageApi.error(t("apps.client_config.scanner.conflict"));
        await query.refetch();
        return;
      }
      if (error instanceof ApiError && error.status === 422 && error.details?.["field"] === "allowed_host_patterns" && typeof error.details["index"] === "number") {
        setServerInvalidRow(scannerHostInputRowForServerIndex(patterns, normalized, error.details["index"]));
        messageApi.error(t("apps.client_config.scanner.host_invalid"));
        return;
      }
      messageApi.error(t("apps.client_config.scanner.save_error"));
    }
  };

  if (query.isError) return <Alert showIcon type="error" title={t("apps.client_config.scanner.load_error")} action={<Button onClick={() => void query.refetch()}>{t("common.actions.retry")}</Button>} />;

  return <Space className="ak-client-config-panel" orientation="vertical" size="large">
    {holder}
    <Alert showIcon type="info" title={t("apps.client_config.scanner.runtime_title")} description={t("apps.client_config.scanner.runtime_description")} />
    <Form layout="vertical">
      <Form.Item label={t("apps.client_config.scanner.enabled")} extra={t("apps.client_config.scanner.enabled_help")}>
        <Switch aria-label={t("apps.client_config.scanner.enabled")} checked={enabled} disabled={!canUpdate || query.isLoading} onChange={setEnabled} />
      </Form.Item>
      <Form.Item label={t("apps.client_config.scanner.allowed_hosts")} required={enabled} extra={t("apps.client_config.scanner.allowed_hosts_help")} {...(enabled && normalized.length === 0 ? { validateStatus: "error" as const, help: t("apps.client_config.scanner.host_required") } : {})}>
        <Space className="ak-client-config-hosts" orientation="vertical" size="small">
          {patterns.map((value, index) => {
            const rowInvalid = invalidRows[index] === true ? true : serverInvalidRow === index;
            return <div className="ak-client-config-host-row" key={String(index)}>
            <div>
              <Input aria-label={t("apps.client_config.scanner.host_label", { index: index + 1 })} aria-invalid={rowInvalid} disabled={!canUpdate} placeholder={index === 0 ? "example.com" : "*.example.com"} {...(rowInvalid ? { status: "error" as const } : {})} value={value} onBlur={() => {
                const candidate = normalizeScannerHostPattern(value);
                if (candidate == null) return;
                setPatterns((current) => current.map((entry, currentIndex) => currentIndex === index ? candidate : entry));
              }} onChange={(event) => { setServerInvalidRow(null); setPatterns((current) => current.map((entry, currentIndex) => currentIndex === index ? event.target.value : entry)); }} />
              {rowInvalid ? <Typography.Text role="alert" type="danger">{t("apps.client_config.scanner.host_invalid")}</Typography.Text> : null}
            </div>
            <Button aria-label={t("apps.client_config.scanner.remove_host", { index: index + 1 })} danger disabled={!canUpdate} icon={<DeleteOutlined />} onClick={() => { setPatterns((current) => current.filter((_, currentIndex) => currentIndex !== index)); }} />
          </div>;
          })}
          {patterns.length === 0 ? <Typography.Text type="secondary">{t("apps.client_config.scanner.empty_hosts")}</Typography.Text> : null}
          {canUpdate ? <Button block disabled={patterns.length >= 100} icon={<PlusOutlined />} onClick={() => { setPatterns((current) => [...current, ""]); }}>{t("apps.client_config.scanner.add_host")}</Button> : null}
        </Space>
      </Form.Item>
    </Form>
    <div className="ak-client-config-actions"><span /><Button type="primary" disabled={!canUpdate || !dirty || invalid} loading={mutation.isPending} onClick={() => void save()}>{t("common.actions.save")}</Button></div>
  </Space>;
}
