import { Link } from "@tanstack/react-router";
import { Alert, Button, Card, Descriptions, Table, Tag, Typography } from "antd";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import { useAdminFile, useAdminFileUsages } from "../features/files/hooks";

const sizeLabel = (bytes: number) =>
  bytes < 1024
    ? `${String(bytes)} B`
    : bytes < 1024 * 1024
      ? `${(bytes / 1024).toFixed(1)} KiB`
      : `${(bytes / 1024 / 1024).toFixed(1)} MiB`;

export function AdminFileDetailPage({ fileId }: { fileId: string }) {
  const { t, i18n } = useTranslation();
  const file = useAdminFile(fileId);
  const usages = useAdminFileUsages(fileId);
  const formatter = useMemo(
    () => new Intl.DateTimeFormat(i18n.language, { dateStyle: "long", timeStyle: "short" }),
    [i18n.language],
  );

  if (file.isPending) {
    return <div className="ak-centered-state" aria-live="polite"><span className="ak-loading-indicator" /><span>{t("common.states.loading")}</span></div>;
  }
  if (file.isError) {
    return <div className="ak-page-container"><div className="ak-form-error" role="alert">{t("system.files.load_error")} <Button onClick={() => { void file.refetch(); }}>{t("common.actions.retry")}</Button></div></div>;
  }

  const current = file.data;
  return (
    <div className="ak-page-container">
      <Link className="ak-back-link" to="/system/storage/files">{t("common.actions.back")}</Link>
      <header className="ak-page-heading">
        <div>
          <Typography.Title level={1}>{current.original_name}</Typography.Title>
          <Typography.Paragraph type="secondary">{current.media_type}</Typography.Paragraph>
        </div>
        <Tag className={current.status === "ready" ? "ak-status-success" : current.status === "quarantined" ? "ak-status-error" : "ak-status-warning"}>{t(`system.files.status.${current.status}`)}</Tag>
      </header>
      <Alert showIcon type="info" title={t("system.files.detail.scan_gate")} />
      <Card>
        <Descriptions column={{ xs: 1, sm: 2 }} bordered items={[
          { key: "type", label: t("system.files.filters.media"), children: current.media_type },
          { key: "size", label: t("system.files.columns.size"), children: sizeLabel(current.size_bytes) },
          { key: "status", label: t("system.files.columns.status"), children: t(`system.files.status.${current.status}`) },
          { key: "scan", label: t("system.files.columns.scan"), children: t(`system.files.scan.${current.scan_status}`) },
          { key: "created", label: t("system.files.columns.created"), children: formatter.format(new Date(current.created_at)) },
          { key: "usages", label: t("system.files.columns.usages"), children: current.usage_count },
        ]} />
      </Card>
      <Typography.Title level={2}>{t("system.files.detail.usages")}</Typography.Title>
      <Card>
        <Table
          columns={[{ title: t("system.files.detail.usages"), key: "usage", render: (_, usage) => <span>{usage.module_code} / {usage.entity_type} / {usage.field_name}</span> }]}
          dataSource={usages.data?.items ?? []}
          loading={usages.isPending}
          locale={{ emptyText: t("system.files.detail.no_usages") }}
          pagination={false}
          rowKey="id"
        />
      </Card>
    </div>
  );
}
