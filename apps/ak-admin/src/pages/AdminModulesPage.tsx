import {
  Button,
  Card,
  Input,
  Select,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { AdminModule } from "../generated/api/types.gen";
import { useAdminModules } from "../features/settings/hooks";
interface Filters {
  q: string;
  status: string;
}
const read = (): Filters => {
  const p = new URLSearchParams(location.search);
  return { q: p.get("q") ?? "", status: p.get("status") ?? "" };
};
const persist = (f: Filters) => {
  const p = new URLSearchParams();
  if (f.q) p.set("q", f.q);
  if (f.status) p.set("status", f.status);
  history.replaceState(
    history.state,
    "",
    `${location.pathname}${p.size ? `?${p}` : ""}`,
  );
};
export function AdminModulesPage() {
  const { t } = useTranslation();
  const [filters, setState] = useState(read);
  const setFilters = (f: Filters) => {
    setState(f);
    persist(f);
  };
  const query = useAdminModules(filters);
  const columns: TableColumnsType<AdminModule> = [
    {
      title: t("settings.modules.columns.module"),
      render: (_, r) => (
        <div>
          <strong>{r.name}</strong>
          <div className="ak-table-secondary">{r.code}</div>
        </div>
      ),
    },
    { title: t("settings.modules.columns.version"), dataIndex: "version" },
    {
      title: t("settings.modules.columns.capabilities"),
      dataIndex: "capabilities",
      render: (value: AdminModule["capabilities"]) => {
        const keys = Object.keys(value);
        return keys.length
          ? keys.map((key) => <Tag key={key}>{key}</Tag>)
          : "—";
      },
    },
    {
      title: t("settings.modules.columns.description"),
      dataIndex: "description",
    },
    {
      title: t("settings.modules.columns.status"),
      dataIndex: "status",
      render: (v: AdminModule["status"]) => (
        <Tag className={v === "enabled" ? "ak-status-success" : ""}>
          {t(`settings.modules.status.${v}`)}
        </Tag>
      ),
    },
  ];
  return (
    <div className="ak-page-container">
      <header className="ak-page-heading">
        <div>
          <Typography.Title level={1}>
            {t("settings.modules.title")}
          </Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("settings.modules.description")}
          </Typography.Paragraph>
        </div>
        <Tag>{t("settings.common.read_only")}</Tag>
      </header>
      <Card>
        <Typography.Paragraph className="ak-module-safety">
          {t("settings.modules.compile_time_notice")}
        </Typography.Paragraph>
        <div
          className="ak-settings-filters"
          role="search"
          aria-label={t("settings.modules.search_landmark")}
        >
          <Input.Search
            allowClear
            aria-label={t("settings.modules.filters.query")}
            placeholder={t("settings.modules.filters.query")}
            value={filters.q}
            onChange={(e) => {
              setFilters({ ...filters, q: e.target.value });
            }}
          />
          <Select
            allowClear
            aria-label={t("settings.modules.filters.status")}
            placeholder={t("settings.modules.filters.status")}
            value={filters.status || undefined}
            onChange={(value) => {
              setFilters({ ...filters, status: value ?? "" });
            }}
            options={["enabled", "disabled"].map((value) => ({
              value,
              label: t(`settings.modules.status.${value}`),
            }))}
          />
        </div>
        {query.isError ? (
          <div role="alert" className="ak-form-error">
            {t("settings.common.load_error")}{" "}
            <Button onClick={() => void query.refetch()}>
              {t("settings.common.retry")}
            </Button>
          </div>
        ) : null}
        <div
          className="ak-table-scroll"
          tabIndex={0}
          ref={(node) => {
            node
              ?.querySelector(".ant-table-content")
              ?.setAttribute("tabindex", "0");
          }}
        >
          <Table
            rowKey="id"
            columns={columns}
            dataSource={query.data?.items ?? []}
            loading={query.isPending}
            pagination={false}
            scroll={{ x: 850 }}
            locale={{ emptyText: t("settings.modules.empty") }}
          />
        </div>
      </Card>
    </div>
  );
}
