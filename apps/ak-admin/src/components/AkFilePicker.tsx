import { Button, Input, Modal, Table, Tag, Typography } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { AdminFile } from "../generated/api/types.gen";
import { useAuthStore } from "../features/auth/store";
import { useAdminFiles } from "../features/files/hooks";
import { AkFileUploader } from "./AkFileUploader";

interface AkFilePickerProps {
  open: boolean;
  onClose: () => void;
  onSelect: (file: AdminFile) => void;
}

export function AkFilePicker({ open, onClose, onSelect }: AkFilePickerProps) {
  const { t } = useTranslation();
  const [q, setQ] = useState("");
  const [selected, setSelected] = useState<AdminFile | null>(null);
  const canUpload = useAuthStore((state) => state.context?.permissions.includes("storage.file.upload") ?? false);
  const files = useAdminFiles({ q, status: "ready", page: 1, page_size: 50 });
  const selectable = (file: AdminFile) =>
    file.status === "ready" && ["clean", "skipped"].includes(file.scan_status);
  return (
    <Modal
      destroyOnHidden
      open={open}
      title={t("system.files.picker.title")}
      onCancel={onClose}
      okText={t("system.files.actions.select")}
      okButtonProps={{ disabled: !selected }}
      onOk={() => {
        if (selected) onSelect(selected);
      }}
    >
      <Typography.Paragraph type="secondary">
        {t("system.files.picker.description")}
      </Typography.Paragraph>
      <Input.Search
        allowClear
        aria-label={t("system.files.filters.query")}
        value={q}
        onChange={(event) => { setQ(event.target.value); }}
      />
      {canUpload ? <AkFileUploader compact onUploaded={async (file) => { setSelected(file); await files.refetch(); }} /> : null}
      <Table
        className="ak-file-picker-table"
        columns={[
          { title: t("system.files.columns.file"), dataIndex: "original_name" },
          {
            title: t("system.files.columns.scan"),
            dataIndex: "scan_status",
            render: (value: AdminFile["scan_status"]) => (
              <Tag className={["clean", "skipped"].includes(value) ? "ak-status-success" : "ak-status-warning"}>
                {t(`system.files.scan.${value}`)}
              </Tag>
            ),
          },
        ]}
        dataSource={files.data?.items ?? []}
        loading={files.isPending}
        locale={{ emptyText: t("system.files.empty") }}
        pagination={false}
        rowKey="id"
        rowSelection={{
          type: "radio",
          selectedRowKeys: selected ? [selected.id] : [],
          getCheckboxProps: (file) => ({ disabled: !selectable(file) }),
          onSelect: (file) => { setSelected(file); },
        }}
        scroll={{ x: 520, y: 360 }}
      />
      {selected ? (
        <div className="ak-org-feedback" role="status">
          {t("system.files.picker.selected", { name: selected.original_name })}
        </div>
      ) : null}
      <Button className="ak-sr-only" onClick={onClose}>
        {t("common.actions.close")}
      </Button>
    </Modal>
  );
}
