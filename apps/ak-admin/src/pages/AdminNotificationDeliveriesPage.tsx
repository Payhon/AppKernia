import {
  Alert,
  Button,
  Card,
  Descriptions,
  Drawer,
  Grid,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { AdminNotificationDelivery } from "../generated/api/types.gen";
import { useAuthStore } from "../features/auth/store";
import {
  useNotificationDeliveries,
  useNotificationDelivery,
  useNotificationDeliveryMutations,
} from "../features/notifications/hooks";

interface Filters {
  q: string;
  status: string;
  channel: string;
  page: number;
  page_size: number;
}
const read = (): Filters => {
  const p = new URLSearchParams(location.search);
  return {
    q: p.get("q") ?? "",
    status: p.get("status") ?? "",
    channel: p.get("channel") ?? "",
    page: Number(p.get("page") ?? 1),
    page_size: Number(p.get("page_size") ?? 20),
  };
};
const persist = (f: Filters) => {
  const p = new URLSearchParams();
  Object.entries(f).forEach(([k, v]) => {
    if (
      v !== "" &&
      !(k === "page" && v === 1) &&
      !(k === "page_size" && v === 20)
    )
      p.set(k, String(v));
  });
  history.replaceState(
    history.state,
    "",
    `${location.pathname}${p.size ? `?${p}` : ""}`,
  );
};

export function AdminNotificationDeliveriesPage() {
  const { t, i18n } = useTranslation();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(
    useAuthStore((state) => state.context?.permissions ?? []),
  );
  const [filters, setState] = useState(read);
  const setFilters = (next: Filters) => {
    setState(next);
    persist(next);
  };
  const query = useNotificationDeliveries(filters);
  const [selected, setSelected] = useState<string | null>(null);
  const detail = useNotificationDelivery(selected);
  const mutations = useNotificationDeliveryMutations();
  const [feedback, setFeedback] = useState<{
    key: string;
    error?: boolean;
  } | null>(null);
  const formatter = useMemo(
    () =>
      new Intl.DateTimeFormat(i18n.language, {
        dateStyle: "medium",
        timeStyle: "short",
      }),
    [i18n.language],
  );
  const retry = (delivery: AdminNotificationDelivery) => {
    Modal.confirm({
      title: t("notifications.deliveries.retry.title"),
      content: t("notifications.deliveries.retry.confirm", {
        target: delivery.target_hint,
        attempt: delivery.attempt_count + 1,
        max: delivery.max_attempts,
      }),
      okText: t("notifications.deliveries.retry.action"),
      cancelText: t("common.actions.cancel"),
      onOk: async () => {
        try {
          await mutations.retry.mutateAsync(delivery.id);
          setFeedback({ key: "notifications.deliveries.retry.success" });
        } catch {
          setFeedback({
            key: "notifications.deliveries.retry.error",
            error: true,
          });
        }
      },
    });
  };
  const columns: TableColumnsType<AdminNotificationDelivery> = [
    {
      title: t("notifications.deliveries.columns.target"),
      dataIndex: "target_hint",
      render: (v: string) => (
        <strong>{v || t("notifications.deliveries.no_target")}</strong>
      ),
    },
    {
      title: t("notifications.deliveries.columns.channel"),
      dataIndex: "channel",
      render: (v: string) => t(`notifications.channel.${v}`),
    },
    {
      title: t("notifications.deliveries.columns.provider"),
      dataIndex: "provider",
      render: (v: string) => v || "—",
      responsive: ["md"],
    },
    {
      title: t("notifications.columns.status"),
      dataIndex: "status",
      render: (v: string) => (
        <Tag
          className={
            v === "sent"
              ? "ak-status-success"
              : v === "failed"
                ? "ak-status-error"
                : "ak-status-warning"
          }
        >
          {t(`notifications.delivery_status.${v}`)}
        </Tag>
      ),
    },
    {
      title: t("notifications.deliveries.columns.attempts"),
      key: "attempts",
      render: (_, x) =>
        `${String(x.attempt_count)} / ${String(x.max_attempts)}`,
      responsive: ["md"],
    },
    {
      title: t("notifications.deliveries.columns.scheduled"),
      dataIndex: "scheduled_at",
      render: (v: string) => formatter.format(new Date(v)),
      responsive: ["lg"],
    },
    {
      title: t("notifications.columns.actions"),
      key: "actions",
      width: screens.md ? 220 : 130,
      render: (_, x) => (
        <Space wrap>
          <Button
            size="small"
            onClick={() => {
              setSelected(x.id);
            }}
          >
            {t("common.actions.view")}
          </Button>
          {x.status === "failed" &&
          x.attempt_count < x.max_attempts &&
          permissions.has("notify.delivery.retry") ? (
            <Button
              type="primary"
              size="small"
              loading={mutations.retry.isPending}
              onClick={() => {
                retry(x);
              }}
            >
              {t("notifications.deliveries.retry.action")}
            </Button>
          ) : null}
        </Space>
      ),
    },
  ];
  const current = detail.data;
  return (
    <div className="ak-page-container">
      <header className="ak-page-heading">
        <div>
          <Typography.Title level={1}>
            {t("notifications.deliveries.title")}
          </Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("notifications.deliveries.description")}
          </Typography.Paragraph>
        </div>
      </header>
      {feedback ? (
        <div
          className={feedback.error ? "ak-form-error" : "ak-org-feedback"}
          role={feedback.error ? "alert" : "status"}
        >
          {t(feedback.key)}
        </div>
      ) : null}
      <Card>
        <div
          className="ak-settings-filters"
          role="search"
          aria-label={t("notifications.filters.landmark")}
        >
          <Input.Search
            allowClear
            aria-label={t("notifications.filters.query")}
            value={filters.q}
            onChange={(e) => {
              setFilters({ ...filters, q: e.target.value, page: 1 });
            }}
          />
          <Select
            allowClear
            aria-label={t("notifications.filters.status")}
            placeholder={t("notifications.filters.status")}
            value={filters.status || undefined}
            onChange={(v) => {
              setFilters({ ...filters, status: v ?? "", page: 1 });
            }}
            options={[
              "pending",
              "processing",
              "sent",
              "failed",
              "cancelled",
            ].map((v) => ({
              value: v,
              label: t(`notifications.delivery_status.${v}`),
            }))}
          />
          <Select
            allowClear
            aria-label={t("notifications.deliveries.filters.channel")}
            placeholder={t("notifications.deliveries.filters.channel")}
            value={filters.channel || undefined}
            onChange={(v) => {
              setFilters({ ...filters, channel: v ?? "", page: 1 });
            }}
            options={["email", "sms", "push", "webhook"].map((v) => ({
              value: v,
              label: t(`notifications.channel.${v}`),
            }))}
          />
        </div>
        {query.isError ? (
          <Alert
            type="error"
            showIcon
            title={t("notifications.feedback.load_error")}
            action={
              <Button onClick={() => void query.refetch()}>
                {t("common.actions.retry")}
              </Button>
            }
          />
        ) : null}
        <div className="ak-table-scroll">
          <Table
            rowKey="id"
            columns={columns}
            dataSource={query.data?.items ?? []}
            loading={query.isPending}
            locale={{ emptyText: t("notifications.deliveries.empty") }}
            {...((query.data?.items.length ?? 0) > 0
              ? { scroll: { x: 900 } }
              : {})}
            pagination={{
              current: filters.page,
              pageSize: filters.page_size,
              total: query.data?.total ?? 0,
              showSizeChanger: true,
              onChange: (page, page_size) => {
                setFilters({ ...filters, page, page_size });
              },
            }}
          />
        </div>
      </Card>
      <Drawer
        rootClassName="ak-notification-drawer"
        destroyOnHidden
        size={screens.md ? "large" : "100%"}
        open={selected !== null}
        title={t("notifications.deliveries.detail.title")}
        onClose={() => {
          setSelected(null);
        }}
        loading={detail.isPending}
      >
        {detail.isError ? (
          <Alert
            type="error"
            showIcon
            title={t("notifications.feedback.load_error")}
          />
        ) : null}
        {current ? (
          <>
            <Alert
              type="info"
              showIcon
              title={t("notifications.deliveries.detail.privacy")}
            />
            <Descriptions
              bordered
              column={1}
              items={[
                {
                  key: "target",
                  label: t("notifications.deliveries.columns.target"),
                  children:
                    current.target_hint ||
                    t("notifications.deliveries.no_target"),
                },
                {
                  key: "status",
                  label: t("notifications.columns.status"),
                  children: t(
                    `notifications.delivery_status.${current.status}`,
                  ),
                },
                {
                  key: "channel",
                  label: t("notifications.deliveries.columns.channel"),
                  children: t(`notifications.channel.${current.channel}`),
                },
                {
                  key: "provider",
                  label: t("notifications.deliveries.columns.provider"),
                  children: current.provider || "—",
                },
                {
                  key: "attempts",
                  label: t("notifications.deliveries.columns.attempts"),
                  children: `${String(current.attempt_count)} / ${String(current.max_attempts)}`,
                },
                {
                  key: "error_code",
                  label: t("notifications.deliveries.detail.error_code"),
                  children: current.error_code || "—",
                },
                {
                  key: "error",
                  label: t("notifications.deliveries.detail.error_summary"),
                  children: (
                    <span role={current.error_summary ? "alert" : undefined}>
                      {current.error_summary || "—"}
                    </span>
                  ),
                },
              ]}
            />
            {current.status === "failed" &&
            current.attempt_count < current.max_attempts &&
            permissions.has("notify.delivery.retry") ? (
              <Button
                type="primary"
                loading={mutations.retry.isPending}
                onClick={() => {
                  retry(current);
                }}
              >
                {t("notifications.deliveries.retry.action")}
              </Button>
            ) : null}
          </>
        ) : null}
      </Drawer>
    </div>
  );
}
