import {
  Button,
  Card,
  Input,
  InputNumber,
  Select,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Region } from "../generated/api/types.gen";
import { authSession } from "../features/auth/store";
import { useAdminRegions } from "../features/settings/hooks";

interface Filters {
  q: string;
  level: number | undefined;
  status: string;
}
const read = (): Filters => {
  const p = new URLSearchParams(location.search);
  const rawLevel = p.get("level");
  const level = rawLevel === null ? Number.NaN : Number(rawLevel);
  return {
    q: p.get("q") ?? "",
    level: Number.isInteger(level) && level >= 0 ? level : undefined,
    status: p.get("status") ?? "",
  };
};
const persist = (f: Filters) => {
  const p = new URLSearchParams();
  if (f.q) p.set("q", f.q);
  if (f.level !== undefined) p.set("level", String(f.level));
  if (f.status) p.set("status", f.status);
  history.replaceState(
    history.state,
    "",
    `${location.pathname}${p.size ? `?${p}` : ""}`,
  );
};
export function AdminRegionsPage() {
  const { t } = useTranslation();
  const [filters, setFiltersState] = useState(read);
  const [children, setChildren] = useState<Record<string, Region[]>>({});
  const [loading, setLoading] = useState<string[]>([]);
  const setFilters = (next: Filters) => {
    setFiltersState(next);
    persist(next);
  };
  const query = useAdminRegions({
    q: filters.q,
    status: filters.status,
    limit: 100,
    ...(filters.level === undefined ? {} : { level: filters.level }),
  });
  const decorate = (items: Region[]): (Region & { children?: Region[] })[] =>
    items.map((item) => {
      const loaded = children[item.code];
      return {
        ...item,
        ...(loaded
          ? { children: decorate(loaded) }
          : item.has_children
            ? { children: [] }
            : {}),
      };
    });
  const data = useMemo(
    () => decorate(query.data?.items ?? []),
    [query.data, children],
  );
  const load = async (row: Region) => {
    if (!row.has_children || children[row.code]) return;
    setLoading((x) => [...x, row.code]);
    try {
      const result = await authSession.adminRegions({
        parent_code: row.code,
        status: filters.status,
        limit: 100,
      });
      setChildren((x) => ({ ...x, [row.code]: result.items }));
    } finally {
      setLoading((x) => x.filter((code) => code !== row.code));
    }
  };
  const columns: TableColumnsType<Region> = [
    {
      title: t("settings.regions.columns.name"),
      dataIndex: "name",
      render: (_, r) => (
        <div>
          <strong>{r.name}</strong>
          <div className="ak-table-secondary">{r.full_name}</div>
        </div>
      ),
    },
    { title: t("settings.regions.columns.code"), dataIndex: "code" },
    {
      title: t("settings.regions.columns.level"),
      dataIndex: "level",
      width: 90,
    },
    { title: t("settings.regions.columns.postal"), dataIndex: "postal_code" },
    {
      title: t("settings.regions.columns.coordinates"),
      render: (_, r) =>
        r.longitude == null || r.latitude == null
          ? "—"
          : `${String(r.longitude)}, ${String(r.latitude)}`,
    },
    {
      title: t("settings.regions.columns.status"),
      dataIndex: "status",
      render: (v: Region["status"]) => (
        <Tag className={v === "active" ? "ak-status-success" : ""}>
          {t(`settings.common.status.${v}`)}
        </Tag>
      ),
    },
  ];
  return (
    <div className="ak-page-container">
      <header className="ak-page-heading">
        <div>
          <Typography.Title level={1}>
            {t("settings.regions.title")}
          </Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("settings.regions.description")}
          </Typography.Paragraph>
        </div>
        <Tag>{t("settings.common.read_only")}</Tag>
      </header>
      <Card>
        <div
          className="ak-settings-filters"
          role="search"
          aria-label={t("settings.regions.search_landmark")}
        >
          <Input.Search
            allowClear
            aria-label={t("settings.regions.filters.query")}
            placeholder={t("settings.regions.filters.query")}
            value={filters.q}
            onChange={(e) => {
              setChildren({});
              setFilters({ ...filters, q: e.target.value });
            }}
          />
          <InputNumber
            min={0}
            max={10}
            aria-label={t("settings.regions.filters.level")}
            placeholder={t("settings.regions.filters.level")}
            value={filters.level ?? null}
            onChange={(value) => {
              setFilters({
                ...filters,
                level: typeof value === "number" ? value : undefined,
              });
            }}
          />
          <Select
            allowClear
            aria-label={t("settings.regions.filters.status")}
            placeholder={t("settings.regions.filters.status")}
            value={filters.status || undefined}
            onChange={(value) => {
              setFilters({ ...filters, status: value ?? "" });
            }}
            options={["active", "disabled"].map((value) => ({
              value,
              label: t(`settings.common.status.${value}`),
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
          <Table<Region>
            rowKey="code"
            columns={columns}
            dataSource={data}
            loading={query.isPending}
            pagination={false}
            scroll={{ x: 900 }}
            {...(filters.q
              ? {}
              : {
                  expandable: {
                    onExpand: (expanded, row) => {
                      if (expanded) void load(row);
                    },
                    rowExpandable: (row) => row.has_children,
                    expandIcon: (props) =>
                      props.expandable ? (
                        <button
                          type="button"
                          className="ak-tree-expand"
                          aria-label={
                            props.expanded
                              ? t("settings.regions.collapse")
                              : t("settings.regions.expand")
                          }
                          onClick={(e) => {
                            e.stopPropagation();
                            props.onExpand(props.record, e);
                          }}
                        >
                          {loading.includes(props.record.code)
                            ? "…"
                            : props.expanded
                              ? "−"
                              : "+"}
                        </button>
                      ) : null,
                  },
                })}
            locale={{ emptyText: t("settings.regions.empty") }}
          />
        </div>
      </Card>
    </div>
  );
}
