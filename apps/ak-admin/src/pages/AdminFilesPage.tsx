import { Alert, Button, Card, Descriptions, Drawer, Grid, Input, Modal, Select, Space, Table, Tag, Typography, type TableColumnsType } from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";

import { AkFileUploader } from "../components/AkFileUploader";
import type { AdminFile } from "../generated/api/types.gen";
import { useAuthStore, authSession } from "../features/auth/store";
import { useAdminFileMutations, useAdminFiles, useAdminFileUsages } from "../features/files/hooks";
import { useAdminDictionary } from "../features/settings/hooks";

interface Filters { q: string; status: string; scan_status: string; media_type: string; provider: string; page: number; page_size: number }
const readFilters = (): Filters => {
  const params = new URLSearchParams(location.search);
  return { q: params.get("q") ?? "", status: params.get("status") ?? "", scan_status: params.get("scan_status") ?? "", media_type: params.get("media_type") ?? "", provider: params.get("provider") ?? "", page: Number(params.get("page") ?? 1), page_size: Number(params.get("page_size") ?? 20) };
};
const saveFilters = (filters: Filters) => {
  const params = new URLSearchParams();
  Object.entries(filters).forEach(([key, value]) => { if (value !== "" && !(key === "page" && value === 1) && !(key === "page_size" && value === 20)) params.set(key, String(value)); });
  history.replaceState(history.state, "", `${location.pathname}${params.size ? `?${params}` : ""}`);
};
const safe = (file: AdminFile) => file.status === "ready" && ["clean", "skipped"].includes(file.scan_status);
const sizeLabel = (bytes: number) => bytes < 1024 ? `${String(bytes)} B` : bytes < 1024 * 1024 ? `${(bytes / 1024).toFixed(1)} KiB` : `${(bytes / 1024 / 1024).toFixed(1)} MiB`;

export function AdminFilesPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [filters, setState] = useState(readFilters);
  const setFilters = (next: Filters) => { setState(next); saveFilters(next); };
  const files = useAdminFiles(filters);
  const storageDrivers = useAdminDictionary("storage.driver");
  const storageDriverLabels = useMemo(
    () => new Map((storageDrivers.data?.items ?? []).map((item) => [item.value, item.label])),
    [storageDrivers.data?.items],
  );
  const mutations = useAdminFileMutations();
  const [detail, setDetail] = useState<AdminFile | null>(null);
  const usages = useAdminFileUsages(detail?.id ?? null);
  const [feedback, setFeedback] = useState<{ key: string; error?: boolean } | null>(null);
  const formatter = useMemo(() => new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" }), [i18n.language]);

  const download = async (file: AdminFile) => {
    try {
      const result = await authSession.downloadAdminFile(file.id);
      const url = URL.createObjectURL(result.blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = result.file.original_name;
      anchor.click();
      URL.revokeObjectURL(url);
      setFeedback({ key: "system.files.feedback.downloaded" });
    } catch { setFeedback({ key: "system.files.load_error", error: true }); }
  };
  const remove = async (file: AdminFile) => {
    try {
      const currentUsages = await authSession.adminFileUsages(file.id);
      if (currentUsages.items.length > 0) {
        Modal.warning({ title: t("system.files.delete.title"), content: t("system.files.delete.in_use", { count: currentUsages.items.length }), okButtonProps: { type: "default" } });
        return;
      }
      Modal.confirm({ title: t("system.files.delete.title"), content: t("system.files.delete.impact"), okButtonProps: { danger: true }, okText: t("common.actions.delete"), cancelText: t("common.actions.cancel"), onOk: async () => { await mutations.remove.mutateAsync(file.id); setFeedback({ key: "system.files.feedback.deleted" }); } });
    } catch { setFeedback({ key: "system.files.load_error", error: true }); }
  };

  const columns: TableColumnsType<AdminFile> = [
    { title: t("system.files.columns.file"), key: "file", render: (_, file) => <div><strong>{file.original_name}</strong><div className="ak-table-secondary">{file.media_type}</div></div> },
    { title: t("system.files.columns.provider"), dataIndex: "provider", render: (value: AdminFile["provider"]) => <Tag>{storageDriverLabels.get(value) ?? value}</Tag>, responsive: ["md"] },
    { title: t("system.files.columns.size"), dataIndex: "size_bytes", render: (value: number) => sizeLabel(value), responsive: ["sm"] },
    { title: t("system.files.columns.status"), dataIndex: "status", render: (value: AdminFile["status"]) => <Tag className={value === "ready" ? "ak-status-success" : value === "quarantined" ? "ak-status-error" : "ak-status-warning"}>{t(`system.files.status.${value}`)}</Tag> },
    { title: t("system.files.columns.scan"), dataIndex: "scan_status", render: (value: AdminFile["scan_status"]) => <Tag className={["clean", "skipped"].includes(value) ? "ak-status-success" : value === "infected" ? "ak-status-error" : "ak-status-warning"}>{t(`system.files.scan.${value}`)}</Tag> },
    { title: t("system.files.columns.usages"), dataIndex: "usage_count", responsive: ["lg"] },
    { title: t("system.files.columns.created"), dataIndex: "created_at", render: (value: string) => formatter.format(new Date(value)), responsive: ["lg"] },
    { title: t("system.files.columns.actions"), key: "actions", width: screens.md ? 300 : 140, render: (_, file) => <Space wrap><Button size="small" onClick={() => { void navigate({ to: "/system/storage/files/$fileId", params: { fileId: file.id } }); }}>{t("system.files.actions.details")}</Button>{permissions.has("storage.file.download") && safe(file) ? <Button size="small" onClick={() => void download(file)}>{t("system.files.actions.download")}</Button> : null}{permissions.has("storage.file.delete") ? <Button danger size="small" onClick={() => void remove(file)}>{t("common.actions.delete")}</Button> : null}</Space> },
  ];

  return <div className="ak-page-container">
    <header className="ak-page-heading"><div><Typography.Title level={1}>{t("system.files.title")}</Typography.Title><Typography.Paragraph type="secondary">{t("system.files.description")}</Typography.Paragraph></div></header>
    {permissions.has("storage.file.upload") ? <AkFileUploader onUploaded={async () => { await files.refetch(); }} /> : null}
    {feedback ? <div className={feedback.error ? "ak-form-error" : "ak-org-feedback"} role={feedback.error ? "alert" : "status"}>{t(feedback.key)}</div> : null}
    <Card>
      <div className="ak-settings-filters" role="search">
        <Input.Search allowClear aria-label={t("system.files.filters.query")} value={filters.q} onChange={(event) => { setFilters({ ...filters, q: event.target.value, page: 1 }); }} />
        <Select allowClear aria-label={t("system.files.filters.status")} placeholder={t("system.files.filters.status")} value={filters.status || undefined} onChange={(value) => { setFilters({ ...filters, status: value ?? "", page: 1 }); }} options={["pending", "ready", "quarantined"].map((value) => ({ value, label: t(`system.files.status.${value}`) }))} />
        <Select allowClear aria-label={t("system.files.filters.scan")} placeholder={t("system.files.filters.scan")} value={filters.scan_status || undefined} onChange={(value) => { setFilters({ ...filters, scan_status: value ?? "", page: 1 }); }} options={["pending", "clean", "infected", "failed", "skipped"].map((value) => ({ value, label: t(`system.files.scan.${value}`) }))} />
        <Select allowClear aria-label={t("system.files.filters.provider")} disabled={storageDrivers.isPending || storageDrivers.isError} loading={storageDrivers.isPending} placeholder={t("system.files.filters.provider")} value={filters.provider || undefined} onChange={(value) => { setFilters({ ...filters, provider: value ?? "", page: 1 }); }} options={(storageDrivers.data?.items ?? []).map((item) => ({ value: item.value, label: item.label }))} />
        <Input aria-label={t("system.files.filters.media")} placeholder={t("system.files.filters.media")} value={filters.media_type} onChange={(event) => { setFilters({ ...filters, media_type: event.target.value, page: 1 }); }} />
      </div>
      {files.isError ? <Alert type="error" showIcon title={t("system.files.load_error")} action={<Button onClick={() => void files.refetch()}>{t("common.actions.retry")}</Button>} /> : null}
      <div className="ak-table-scroll"><Table columns={columns} dataSource={files.data?.items ?? []} loading={files.isPending} locale={{ emptyText: t("system.files.empty") }} pagination={{ current: filters.page, pageSize: filters.page_size, total: files.data?.total ?? 0, showSizeChanger: true, onChange: (page, page_size) => { setFilters({ ...filters, page, page_size }); } }} rowKey="id" {...((files.data?.items.length ?? 0) > 0 ? { scroll: { x: screens.md ? 1120 : 680 } } : {})} /></div>
    </Card>
    <Drawer destroyOnHidden open={Boolean(detail)} onClose={() => { setDetail(null); }} size="large" title={t("system.files.detail.title")}>
      <Alert showIcon type="info" title={t("system.files.detail.scan_gate")} />
      {detail ? <Descriptions column={1} bordered items={[{ key: "name", label: t("system.files.columns.file"), children: detail.original_name }, { key: "provider", label: t("system.files.columns.provider"), children: storageDriverLabels.get(detail.provider) ?? detail.provider }, { key: "type", label: t("system.files.filters.media"), children: detail.media_type }, { key: "size", label: t("system.files.columns.size"), children: sizeLabel(detail.size_bytes) }, { key: "status", label: t("system.files.columns.status"), children: t(`system.files.status.${detail.status}`) }, { key: "scan", label: t("system.files.columns.scan"), children: t(`system.files.scan.${detail.scan_status}`) }]} /> : null}
      <Typography.Title level={2}>{t("system.files.detail.usages")}</Typography.Title>
      <Table columns={[{ title: t("system.files.detail.usages"), key: "usage", render: (_, usage) => <span>{usage.module_code} / {usage.entity_type} / {usage.field_name}</span> }]} dataSource={usages.data?.items ?? []} loading={usages.isPending} locale={{ emptyText: t("system.files.detail.no_usages") }} pagination={false} rowKey="id" />
    </Drawer>
  </div>;
}
