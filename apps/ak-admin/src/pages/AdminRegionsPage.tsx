import {
  Button,
  Card,
  Drawer,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { useMemo, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import type {
  AdminRegionCreateRequest,
  AdminRegionUpdateRequest,
  Region,
} from "../generated/api/types.gen";
import { authSession, useAuthStore } from "../features/auth/store";
import {
  useAdminRegionMutations,
  useAdminRegions,
} from "../features/settings/hooks";
import { ApiError } from "../shared/api/error";

interface Filters {
  q: string;
  level: number | undefined;
  status: string;
}

interface RegionFormValues {
  code: string;
  parent_code: string;
  level: number;
  name: string;
  full_name: string;
  postal_code: string;
  longitude: number | null;
  latitude: number | null;
  status: "active" | "disabled";
}

type Editor =
  | { mode: "create"; parent: Region }
  | { mode: "edit"; region: Region }
  | null;

export const regionFormSchema = z.object({
  code: z.string().trim().regex(/^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$/),
  parent_code: z.string().trim().min(1).max(32),
  level: z.number().int().min(1).max(2),
  name: z.string().trim().min(1).max(160),
  full_name: z.string().trim().min(1).max(500),
  postal_code: z.string().trim().max(24),
  longitude: z.number().min(-180).max(180).nullable(),
  latitude: z.number().min(-90).max(90).nullable(),
  status: z.enum(["active", "disabled"]),
});

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

const emptyForm: RegionFormValues = {
  code: "",
  parent_code: "",
  level: 1,
  name: "",
  full_name: "",
  postal_code: "",
  longitude: null,
  latitude: null,
  status: "active",
};

export function AdminRegionsPage() {
  const { t } = useTranslation();
  const permissions = new Set(
    useAuthStore((state) => state.context?.permissions ?? []),
  );
  const writable = [
    "sys.region.create",
    "sys.region.update",
    "sys.region.delete",
  ].some((permission) => permissions.has(permission));
  const [filters, setFiltersState] = useState(read);
  const [children, setChildren] = useState<Record<string, Region[]>>({});
  const [loading, setLoading] = useState<string[]>([]);
  const [editor, setEditor] = useState<Editor>(null);
  const [fullNameEdited, setFullNameEdited] = useState(false);
  const [feedback, setFeedback] = useState<{
    key: string;
    error?: boolean;
  } | null>(null);
  const form = useForm<RegionFormValues>({ defaultValues: emptyForm });
  const mutations = useAdminRegionMutations();

  const setFilters = (next: Filters) => {
    setChildren({});
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

  const fetchChildren = async (parentCode: string) => {
    const result = await authSession.adminRegions({
      parent_code: parentCode,
      status: filters.status,
      limit: 100,
    });
    setChildren((current) => {
      const next = { ...current, [parentCode]: result.items };
      for (const [key, items] of Object.entries(next)) {
        next[key] = items.map((item) =>
          item.code === parentCode
            ? { ...item, has_children: result.items.length > 0 }
            : item,
        );
      }
      return next;
    });
  };
  const load = async (row: Region) => {
    if (!row.has_children || children[row.code]) return;
    setLoading((current) => [...current, row.code]);
    try {
      await fetchChildren(row.code);
    } finally {
      setLoading((current) => current.filter((code) => code !== row.code));
    }
  };
  const refreshBranch = async (parentCode: string | null) => {
    if (parentCode) await fetchChildren(parentCode);
    else await query.refetch();
  };

  const openCreate = (parent: Region) => {
    setEditor({ mode: "create", parent });
    setFullNameEdited(false);
    setFeedback(null);
    form.clearErrors();
    form.reset({
      ...emptyForm,
      parent_code: parent.code,
      level: parent.level + 1,
      full_name: `${parent.full_name} / `,
    });
  };
  const openEdit = (region: Region) => {
    setEditor({ mode: "edit", region });
    setFullNameEdited(true);
    setFeedback(null);
    form.clearErrors();
    form.reset({
      code: region.code,
      parent_code: region.parent_code ?? "",
      level: region.level,
      name: region.name,
      full_name: region.full_name,
      postal_code: region.postal_code,
      longitude: region.longitude,
      latitude: region.latitude,
      status: region.status,
    });
  };

  const submit = async () => {
    const parsed = regionFormSchema.safeParse(form.getValues());
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0] as keyof RegionFormValues;
        form.setError(field, {
          message: t("settings.regions.validation.invalid"),
        });
      }
      setFeedback({ key: "settings.regions.feedback.validation", error: true });
      return;
    }
    if (!editor) return;
    try {
      if (editor.mode === "create") {
        const input: AdminRegionCreateRequest = {
          code: parsed.data.code,
          parent_code: editor.parent.code,
          name: parsed.data.name,
          full_name: parsed.data.full_name,
          postal_code: parsed.data.postal_code,
          longitude: parsed.data.longitude,
          latitude: parsed.data.latitude,
          status: parsed.data.status,
        };
        await mutations.create.mutateAsync(input);
        await refreshBranch(editor.parent.code);
        setFeedback({ key: "settings.regions.feedback.created" });
      } else {
        const input: AdminRegionUpdateRequest = {
          name: parsed.data.name,
          full_name: parsed.data.full_name,
          postal_code: parsed.data.postal_code,
          longitude: parsed.data.longitude,
          latitude: parsed.data.latitude,
          status: parsed.data.status,
          version: editor.region.version,
        };
        await mutations.update.mutateAsync({
          code: editor.region.code,
          input,
        });
        await refreshBranch(editor.region.parent_code);
        setFeedback({ key: "settings.regions.feedback.updated" });
      }
      setEditor(null);
    } catch (error) {
      setFeedback({
        key:
          error instanceof ApiError && error.status === 409
            ? "settings.regions.feedback.conflict"
            : "settings.regions.feedback.save_error",
        error: true,
      });
    }
  };

  const remove = async (region: Region) => {
    if (region.has_children) {
      setFeedback({
        key: "settings.regions.feedback.delete_has_children",
        error: true,
      });
      return;
    }
    try {
      await mutations.remove.mutateAsync(region.code);
      await refreshBranch(region.parent_code);
      setFeedback({ key: "settings.regions.feedback.deleted" });
    } catch (error) {
      setFeedback({
        key:
          error instanceof ApiError && error.status === 409
            ? "settings.regions.feedback.delete_has_children"
            : "settings.regions.feedback.delete_error",
        error: true,
      });
    }
  };

  const columns: TableColumnsType<Region> = [
    {
      title: t("settings.regions.columns.name"),
      dataIndex: "name",
      render: (_, region) => (
        <div>
          <strong>{region.name}</strong>
          <div className="ak-table-secondary">{region.full_name}</div>
        </div>
      ),
    },
    { title: t("settings.regions.columns.code"), dataIndex: "code" },
    {
      title: t("settings.regions.columns.level"),
      dataIndex: "level",
      width: 90,
    },
    {
      title: t("settings.regions.columns.postal"),
      dataIndex: "postal_code",
      responsive: ["md"],
    },
    {
      title: t("settings.regions.columns.coordinates"),
      responsive: ["lg"],
      render: (_, region) =>
        region.longitude == null || region.latitude == null
          ? "—"
          : `${String(region.longitude)}, ${String(region.latitude)}`,
    },
    {
      title: t("settings.regions.columns.status"),
      dataIndex: "status",
      render: (value: Region["status"]) => (
        <Tag className={value === "active" ? "ak-status-success" : ""}>
          {t(`settings.common.status.${value}`)}
        </Tag>
      ),
    },
    ...(writable
      ? [
          {
            title: t("settings.regions.columns.actions"),
            key: "actions",
            width: 250,
            render: (_: unknown, region: Region) => (
              <Space wrap size={4}>
                {permissions.has("sys.region.create") && region.level < 2 ? (
                  <Button size="small" onClick={() => { openCreate(region); }}>
                    {t("settings.regions.actions.create_child")}
                  </Button>
                ) : null}
                {permissions.has("sys.region.update") ? (
                  <Button size="small" onClick={() => { openEdit(region); }}>
                    {t("common.actions.edit")}
                  </Button>
                ) : null}
                {permissions.has("sys.region.delete") && region.has_children ? (
                  <Button danger size="small" onClick={() => void remove(region)}>
                    {t("common.actions.delete")}
                  </Button>
                ) : permissions.has("sys.region.delete") ? (
                  <Popconfirm
                    title={t("settings.regions.delete.title")}
                    description={t("settings.regions.delete.description", {
                      name: region.name,
                    })}
                    okText={t("common.actions.delete")}
                    cancelText={t("common.actions.cancel")}
                    onConfirm={() => void remove(region)}
                  >
                    <Button danger size="small">
                      {t("common.actions.delete")}
                    </Button>
                  </Popconfirm>
                ) : null}
              </Space>
            ),
          },
        ]
      : []),
  ];

  const fieldStatus = (name: keyof RegionFormValues) => {
    const error = form.formState.errors[name];
    return error
      ? { help: error.message, validateStatus: "error" as const }
      : {};
  };
  const saving = mutations.create.isPending || mutations.update.isPending;

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
        <Tag>
          {t(
            writable
              ? "settings.regions.editable"
              : "settings.common.read_only",
          )}
        </Tag>
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
          aria-label={t("settings.regions.search_landmark")}
        >
          <Input.Search
            allowClear
            aria-label={t("settings.regions.filters.query")}
            placeholder={t("settings.regions.filters.query")}
            value={filters.q}
            onChange={(event) => { setFilters({ ...filters, q: event.target.value }); }}
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
            onChange={(value) => { setFilters({ ...filters, status: value ?? "" }); }}
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
          ref={(node) =>
            node
              ?.querySelector(".ant-table-content")
              ?.setAttribute("tabindex", "0")
          }
        >
          <Table<Region>
            rowKey="code"
            columns={columns}
            dataSource={data}
            loading={query.isPending}
            pagination={false}
            scroll={{ x: writable ? 1120 : 900 }}
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
                          onClick={(event) => {
                            event.stopPropagation();
                            props.onExpand(props.record, event);
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

      <Drawer
        destroyOnHidden
        size="large"
        open={editor !== null}
        title={t(
          editor?.mode === "create"
            ? "settings.regions.editor.create_title"
            : "settings.regions.editor.edit_title",
        )}
        onClose={() => { setEditor(null); }}
      >
        <Form layout="vertical" requiredMark={false} onFinish={() => void submit()}>
          <div className="ak-access-form-row">
            <Controller
              name="code"
              control={form.control}
              render={({ field }) => (
                <Form.Item
                  label={t("settings.regions.fields.code")}
                  extra={t("settings.regions.fields.code_help")}
                  {...fieldStatus("code")}
                >
                  <Input
                    {...field}
                    autoFocus={editor?.mode === "create"}
                    disabled={editor?.mode === "edit"}
                    aria-label={t("settings.regions.fields.code")}
                  />
                </Form.Item>
              )}
            />
            <Controller
              name="parent_code"
              control={form.control}
              render={({ field }) => (
                <Form.Item label={t("settings.regions.fields.parent")}>
                  <Input
                    {...field}
                    disabled
                    aria-label={t("settings.regions.fields.parent")}
                  />
                </Form.Item>
              )}
            />
          </div>
          <div className="ak-access-form-row">
            <Controller
              name="level"
              control={form.control}
              render={({ field }) => (
                <Form.Item label={t("settings.regions.fields.level")}>
                  <InputNumber
                    {...field}
                    disabled
                    aria-label={t("settings.regions.fields.level")}
                  />
                </Form.Item>
              )}
            />
            <Controller
              name="status"
              control={form.control}
              render={({ field }) => (
                <Form.Item
                  label={t("settings.regions.fields.status")}
                  {...fieldStatus("status")}
                >
                  <Select
                    {...field}
                    aria-label={t("settings.regions.fields.status")}
                    options={["active", "disabled"].map((value) => ({
                      value,
                      label: t(`settings.common.status.${value}`),
                    }))}
                  />
                </Form.Item>
              )}
            />
          </div>
          <Controller
            name="name"
            control={form.control}
            render={({ field }) => (
              <Form.Item
                label={t("settings.regions.fields.name")}
                {...fieldStatus("name")}
              >
                <Input
                  {...field}
                  aria-label={t("settings.regions.fields.name")}
                  onChange={(event) => {
                    field.onChange(event);
                    if (editor?.mode === "create" && !fullNameEdited) {
                      form.setValue(
                        "full_name",
                        `${editor.parent.full_name} / ${event.target.value}`,
                      );
                    }
                  }}
                />
              </Form.Item>
            )}
          />
          <Controller
            name="full_name"
            control={form.control}
            render={({ field }) => (
              <Form.Item
                label={t("settings.regions.fields.full_name")}
                {...fieldStatus("full_name")}
              >
                <Input
                  {...field}
                  aria-label={t("settings.regions.fields.full_name")}
                  onChange={(event) => {
                    setFullNameEdited(true);
                    field.onChange(event);
                  }}
                />
              </Form.Item>
            )}
          />
          <div className="ak-access-form-row">
            <Controller
              name="postal_code"
              control={form.control}
              render={({ field }) => (
                <Form.Item
                  label={t("settings.regions.fields.postal_code")}
                  {...fieldStatus("postal_code")}
                >
                  <Input
                    {...field}
                    inputMode="numeric"
                    aria-label={t("settings.regions.fields.postal_code")}
                  />
                </Form.Item>
              )}
            />
            <Controller
              name="longitude"
              control={form.control}
              render={({ field }) => (
                <Form.Item
                  label={t("settings.regions.fields.longitude")}
                  {...fieldStatus("longitude")}
                >
                  <InputNumber
                    min={-180}
                    max={180}
                    precision={7}
                    value={field.value}
                    onBlur={field.onBlur}
                    onChange={field.onChange}
                    aria-label={t("settings.regions.fields.longitude")}
                  />
                </Form.Item>
              )}
            />
            <Controller
              name="latitude"
              control={form.control}
              render={({ field }) => (
                <Form.Item
                  label={t("settings.regions.fields.latitude")}
                  {...fieldStatus("latitude")}
                >
                  <InputNumber
                    min={-90}
                    max={90}
                    precision={7}
                    value={field.value}
                    onBlur={field.onBlur}
                    onChange={field.onChange}
                    aria-label={t("settings.regions.fields.latitude")}
                  />
                </Form.Item>
              )}
            />
          </div>
          <div className="ak-drawer-actions">
            <Button onClick={() => { setEditor(null); }}>
              {t("common.actions.cancel")}
            </Button>
            <Button type="primary" htmlType="submit" loading={saving}>
              {t("common.actions.save")}
            </Button>
          </div>
        </Form>
      </Drawer>
    </div>
  );
}
