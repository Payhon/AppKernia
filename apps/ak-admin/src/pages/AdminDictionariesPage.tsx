import {
  Alert,
  Button,
  Card,
  Drawer,
  Form,
  Grid,
  Input,
  InputNumber,
  List,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";

import type {
  AdminDictionaryItem,
  AdminDictionaryItemWriteRequest,
  AdminDictionaryType,
  AdminDictionaryTypeWriteRequest,
} from "../generated/api/types.gen";
import { useAuthStore } from "../features/auth/store";
import {
  useAdminDictionaryItems,
  useAdminDictionaryMutations,
  useAdminDictionaryTypes,
} from "../features/settings/hooks";
import { ApiError } from "../shared/api/error";

interface DictionarySearch {
  q: string;
  status: "active" | "disabled" | "";
  type_id: string;
  item_q: string;
  locale: "" | "neutral" | "zh-CN" | "en-US";
  item_status: "active" | "disabled" | "";
  page: number;
  page_size: number;
  item_page: number;
  item_page_size: number;
}
interface TypeValues {
  code: string;
  name: string;
  description: string;
  status: "active" | "disabled";
}
interface ItemValues {
  item_value: string;
  label: string;
  locale: "" | "zh-CN" | "en-US";
  color: string;
  css_class: string;
  sort_order: number;
  is_default: boolean;
  extra: string;
  status: "active" | "disabled";
}
interface Feedback {
  key: string;
  error: boolean;
}
const typeSchema = z.object({
  code: z
    .string()
    .trim()
    .min(1)
    .max(160)
    .regex(/^[a-zA-Z0-9][a-zA-Z0-9._-]*$/),
  name: z.string().trim().min(1).max(160),
  description: z.string().max(500),
  status: z.enum(["active", "disabled"]),
});
const itemSchema = z.object({
  item_value: z.string().trim().min(1).max(255),
  label: z.string().trim().min(1).max(255),
  locale: z.enum(["", "zh-CN", "en-US"]),
  color: z.string().max(64),
  css_class: z.string().max(128),
  sort_order: z.number().int(),
  is_default: z.boolean(),
  extra: z.string(),
  status: z.enum(["active", "disabled"]),
});
const typeDefaults: TypeValues = {
  code: "",
  name: "",
  description: "",
  status: "active",
};
const itemDefaults: ItemValues = {
  item_value: "",
  label: "",
  locale: "",
  color: "",
  css_class: "",
  sort_order: 0,
  is_default: false,
  extra: "{}",
  status: "active",
};
function positive(raw: string | null, fallback: number) {
  const value = Number(raw);
  return Number.isInteger(value) && value > 0 ? value : fallback;
}
function readSearch(): DictionarySearch {
  const p = new URLSearchParams(location.search);
  const status = p.get("status");
  const itemStatus = p.get("item_status");
  const locale = p.get("locale");
  return {
    q: p.get("q") ?? "",
    status: status === "active" || status === "disabled" ? status : "",
    type_id: p.get("type_id") ?? "",
    item_q: p.get("item_q") ?? "",
    locale:
      locale === "neutral" || locale === "zh-CN" || locale === "en-US"
        ? locale
        : "",
    item_status:
      itemStatus === "active" || itemStatus === "disabled" ? itemStatus : "",
    page: positive(p.get("page"), 1),
    page_size: [10, 20, 50].includes(positive(p.get("page_size"), 20))
      ? positive(p.get("page_size"), 20)
      : 20,
    item_page: positive(p.get("item_page"), 1),
    item_page_size: [10, 20, 50, 100].includes(
      positive(p.get("item_page_size"), 20),
    )
      ? positive(p.get("item_page_size"), 20)
      : 20,
  };
}
function persistSearch(search: DictionarySearch) {
  const p = new URLSearchParams();
  for (const [key, value] of Object.entries(search))
    if (
      value !== "" &&
      !(["page", "item_page"].includes(key) && value === 1) &&
      !(["page_size", "item_page_size"].includes(key) && value === 20)
    )
      p.set(key, String(value));
  history.replaceState(
    history.state,
    "",
    `${location.pathname}${p.size ? `?${p}` : ""}`,
  );
}

export function AdminDictionariesPage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(
    useAuthStore((state) => state.context?.permissions ?? []),
  );
  const initial = useMemo(readSearch, []);
  const [search, setSearch] = useState(initial);
  const [typeEditor, setTypeEditor] = useState<
    AdminDictionaryType | null | undefined
  >(undefined);
  const [itemEditor, setItemEditor] = useState<
    AdminDictionaryItem | null | undefined
  >(undefined);
  const [feedback, setFeedback] = useState<Feedback | null>(null);
  const tableRegion = useRef<HTMLDivElement>(null);
  const types = useAdminDictionaryTypes({
    q: search.q,
    status: search.status,
    sort: "key",
    page: search.page,
    page_size: search.page_size,
  });
  const selectedType =
    types.data?.items.find((item) => item.id === search.type_id) ?? null;
  const items = useAdminDictionaryItems(search.type_id, {
    q: search.item_q,
    locale: search.locale,
    status: search.item_status,
    sort: "sort_order",
    page: search.item_page,
    page_size: search.item_page_size,
  });
  const mutations = useAdminDictionaryMutations();
  const typeForm = useForm<TypeValues>({ defaultValues: typeDefaults });
  const itemForm = useForm<ItemValues>({ defaultValues: itemDefaults });
  useEffect(() => {
    persistSearch(search);
  }, [search]);
  useEffect(() => {
    const first = types.data?.items[0];
    if (!search.type_id && first)
      setSearch((current) => ({ ...current, type_id: first.id }));
  }, [search.type_id, types.data]);
  useEffect(() => {
    const scrollable =
      tableRegion.current?.querySelector<HTMLElement>(".ant-table-content");
    if (scrollable) scrollable.tabIndex = 0;
  }, [items.data, screens.md, search.type_id]);
  const patchSearch = (patch: Partial<DictionarySearch>) => {
    setSearch((current) => ({ ...current, ...patch }));
  };
  const errorKey = (error: unknown) =>
    error instanceof ApiError ? error.messageKey : "errors.common.unknown";
  const openTypeCreate = () => {
    typeForm.reset(typeDefaults);
    setTypeEditor(null);
    setFeedback(null);
  };
  const openTypeEdit = (type: AdminDictionaryType) => {
    typeForm.reset({
      code: type.code,
      name: type.name,
      description: type.description,
      status: type.status,
    });
    setTypeEditor(type);
    setFeedback(null);
  };
  const submitType = typeForm.handleSubmit(async (raw) => {
    const parsed = typeSchema.safeParse(raw);
    if (!parsed.success) {
      for (const issue of parsed.error.issues)
        typeForm.setError(issue.path[0] as keyof TypeValues, {
          message: t("validation.invalid", {
            name: t(`settings.dictionaries.fields.${String(issue.path[0])}`),
          }),
        });
      return;
    }
    const input = parsed.data satisfies AdminDictionaryTypeWriteRequest;
    try {
      const saved = typeEditor
        ? await mutations.updateType.mutateAsync({ id: typeEditor.id, input })
        : await mutations.createType.mutateAsync(input);
      setTypeEditor(undefined);
      patchSearch({ type_id: saved.id, item_page: 1 });
      setFeedback({
        key: typeEditor
          ? "settings.dictionaries.feedback.type_updated"
          : "settings.dictionaries.feedback.type_created",
        error: false,
      });
    } catch (error) {
      setFeedback({ key: errorKey(error), error: true });
    }
  });
  const openItemCreate = () => {
    itemForm.reset(itemDefaults);
    setItemEditor(null);
    setFeedback(null);
  };
  const openItemEdit = (item: AdminDictionaryItem) => {
    itemForm.reset({
      item_value: item.item_value,
      label: item.label,
      locale: item.locale ?? "",
      color: item.color,
      css_class: item.css_class,
      sort_order: item.sort_order,
      is_default: item.is_default,
      extra: JSON.stringify(item.extra, null, 2),
      status: item.status,
    });
    setItemEditor(item);
    setFeedback(null);
  };
  const submitItem = itemForm.handleSubmit(async (raw) => {
    const parsed = itemSchema.safeParse(raw);
    if (!parsed.success || !search.type_id) {
      if (!parsed.success)
        for (const issue of parsed.error.issues)
          itemForm.setError(issue.path[0] as keyof ItemValues, {
            message: t("validation.invalid", {
              name: t(`settings.dictionaries.fields.${String(issue.path[0])}`),
            }),
          });
      return;
    }
    let extra: Record<string, unknown>;
    try {
      extra = JSON.parse(parsed.data.extra || "{}") as Record<string, unknown>;
    } catch {
      itemForm.setError("extra", {
        message: t("settings.dictionaries.validation.extra"),
      });
      return;
    }
    const input: AdminDictionaryItemWriteRequest = {
      item_value: parsed.data.item_value.trim(),
      label: parsed.data.label.trim(),
      locale: parsed.data.locale || null,
      color: parsed.data.color.trim(),
      css_class: parsed.data.css_class.trim(),
      sort_order: parsed.data.sort_order,
      is_default: parsed.data.is_default,
      extra,
      status: parsed.data.status,
    };
    try {
      if (itemEditor)
        await mutations.updateItem.mutateAsync({ id: itemEditor.id, input });
      else
        await mutations.createItem.mutateAsync({
          typeId: search.type_id,
          input,
        });
      setItemEditor(undefined);
      setFeedback({
        key: itemEditor
          ? "settings.dictionaries.feedback.item_updated"
          : "settings.dictionaries.feedback.item_created",
        error: false,
      });
    } catch (error) {
      setFeedback({ key: errorKey(error), error: true });
    }
  });
  const confirmDelete = (item: AdminDictionaryItem) => {
    Modal.confirm({
      title: t("settings.dictionaries.delete.title"),
      content: t("settings.dictionaries.delete.description", {
        label: item.label,
      }),
      okText: t("common.actions.delete"),
      cancelText: t("common.actions.cancel"),
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await mutations.deleteItem.mutateAsync(item.id);
          setFeedback({
            key: "settings.dictionaries.feedback.item_deleted",
            error: false,
          });
        } catch (error) {
          setFeedback({ key: errorKey(error), error: true });
        }
      },
    });
  };
  const columns: TableColumnsType<AdminDictionaryItem> = [
    {
      title: t("settings.dictionaries.fields.label"),
      key: "label",
      render: (_, item) => (
        <div>
          <strong>{item.label}</strong>
          <div className="ak-settings-code">{item.item_value}</div>
        </div>
      ),
    },
    {
      title: t("settings.dictionaries.fields.locale"),
      dataIndex: "locale",
      render: (value: AdminDictionaryItem["locale"]) => (
        <Tag>
          {t(
            value
              ? `settings.common.locale.${value}`
              : "settings.common.locale.neutral",
          )}
        </Tag>
      ),
    },
    {
      title: t("settings.dictionaries.fields.appearance"),
      key: "appearance",
      responsive: ["lg"],
      render: (_, item) => (
        <Space>
          {item.color ? (
            <span
              className="ak-dictionary-swatch"
              style={{ backgroundColor: item.color }}
              aria-hidden="true"
            />
          ) : null}
          <span>{item.color || item.css_class || "—"}</span>
        </Space>
      ),
    },
    {
      title: t("settings.dictionaries.fields.sort_order"),
      dataIndex: "sort_order",
      responsive: ["md"],
    },
    {
      title: t("settings.dictionaries.fields.status"),
      dataIndex: "status",
      responsive: ["md"],
      render: (value: "active" | "disabled") => (
        <Tag
          className={
            value === "active" ? "ak-status-success" : "ak-status-neutral"
          }
        >
          {t(`settings.common.status.${value}`)}
        </Tag>
      ),
    },
    {
      title: t("common.actions.edit"),
      key: "actions",
      width: screens.md ? 150 : 100,
      render: (_, item) =>
        item.is_locked ? (
          "—"
        ) : (
          <Space>
            {permissions.has("sys.dictionary.update") ? (
              <Button
                type="link"
                onClick={() => {
                  openItemEdit(item);
                }}
              >
                {t("common.actions.edit")}
              </Button>
            ) : null}
            {permissions.has("sys.dictionary.delete") ? (
              <Button
                danger
                type="link"
                onClick={() => {
                  confirmDelete(item);
                }}
              >
                {t("common.actions.delete")}
              </Button>
            ) : null}
          </Space>
        ),
    },
  ];
  return (
    <div className="ak-page-container">
      <header className="ak-page-heading ak-users-heading">
        <div>
          <Typography.Title level={1}>
            {t("routes.system.settings.dictionaries.title")}
          </Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("settings.dictionaries.description")}
          </Typography.Paragraph>
        </div>
        {permissions.has("sys.dictionary.create") ? (
          <Button type="primary" onClick={openTypeCreate}>
            {t("settings.dictionaries.actions.create_type")}
          </Button>
        ) : null}
      </header>
      {feedback ? (
        <div
          className={feedback.error ? "ak-form-error" : "ak-org-feedback"}
          role={feedback.error ? "alert" : "status"}
        >
          {t(feedback.key)}
        </div>
      ) : null}
      <div className="ak-dictionary-layout">
        <Card
          className="ak-dictionary-master"
          title={t("settings.dictionaries.types_title")}
        >
          <div
            className="ak-dictionary-type-filters"
            role="search"
            aria-label={t("settings.dictionaries.search.types")}
          >
            <Input.Search
              allowClear
              aria-label={t("settings.dictionaries.filters.type_query")}
              placeholder={t("settings.dictionaries.filters.type_query")}
              value={search.q}
              onChange={(event) => {
                patchSearch({ q: event.target.value, page: 1, type_id: "" });
              }}
            />
            <Select
              allowClear
              aria-label={t("settings.dictionaries.filters.status")}
              placeholder={t("settings.dictionaries.filters.status")}
              value={search.status || undefined}
              onChange={(value) => {
                patchSearch({ status: value ?? "", page: 1, type_id: "" });
              }}
              options={["active", "disabled"].map((value) => ({
                value,
                label: t(`settings.common.status.${value}`),
              }))}
            />
          </div>
          {types.isError ? (
            <Alert
              showIcon
              type="error"
              title={t("settings.dictionaries.load_error")}
            />
          ) : (
            <List
              loading={types.isPending}
              dataSource={types.data?.items ?? []}
              locale={{ emptyText: t("settings.dictionaries.empty_types") }}
              pagination={{
                current: search.page,
                pageSize: search.page_size,
                total: types.data?.total ?? 0,
                size: "small",
                hideOnSinglePage: true,
                onChange: (page) => {
                  patchSearch({ page, type_id: "" });
                },
              }}
              renderItem={(type) => (
                <List.Item
                  className={
                    type.id === search.type_id
                      ? "ak-dictionary-type-selected"
                      : ""
                  }
                  {...(!type.is_locked &&
                  permissions.has("sys.dictionary.update")
                    ? {
                        actions: [
                          <Button
                            key="edit"
                            type="link"
                            onClick={(event) => {
                              event.stopPropagation();
                              openTypeEdit(type);
                            }}
                          >
                            {t("common.actions.edit")}
                          </Button>,
                        ],
                      }
                    : {})}
                >
                  <button
                    className="ak-dictionary-type-button"
                    type="button"
                    aria-pressed={type.id === search.type_id}
                    onClick={() => {
                      patchSearch({ type_id: type.id, item_page: 1 });
                    }}
                  >
                    <span className="ak-dictionary-type-title">
                      <Space>
                        <span>{type.name}</span>
                        {type.is_locked ? (
                          <Tag>{t("settings.common.system_locked")}</Tag>
                        ) : null}
                      </Space>
                    </span>
                    <span className="ak-settings-code">{type.code}</span>
                  </button>
                </List.Item>
              )}
            />
          )}
        </Card>
        <Card
          className="ak-dictionary-detail"
          title={selectedType?.name ?? t("settings.dictionaries.select_type")}
          extra={
            selectedType &&
            !selectedType.is_locked &&
            permissions.has("sys.dictionary.create") ? (
              <Button onClick={openItemCreate}>
                {t("settings.dictionaries.actions.create_item")}
              </Button>
            ) : null
          }
        >
          {!selectedType ? (
            <Typography.Paragraph type="secondary">
              {t("settings.dictionaries.select_hint")}
            </Typography.Paragraph>
          ) : (
            <>
              {selectedType.is_locked ? (
                <Alert
                  className="ak-settings-lock-alert"
                  showIcon
                  type="info"
                  title={t("settings.dictionaries.locked.title")}
                  description={t("settings.dictionaries.locked.description")}
                />
              ) : null}
              <div
                className="ak-dictionary-item-filters"
                role="search"
                aria-label={t("settings.dictionaries.search.items")}
              >
                <Input.Search
                  allowClear
                  aria-label={t("settings.dictionaries.filters.item_query")}
                  placeholder={t("settings.dictionaries.filters.item_query")}
                  value={search.item_q}
                  onChange={(event) => {
                    patchSearch({ item_q: event.target.value, item_page: 1 });
                  }}
                />
                <Select
                  allowClear
                  aria-label={t("settings.dictionaries.filters.locale")}
                  placeholder={t("settings.dictionaries.filters.locale")}
                  value={search.locale || undefined}
                  onChange={(value) => {
                    patchSearch({ locale: value ?? "", item_page: 1 });
                  }}
                  options={["neutral", "zh-CN", "en-US"].map((value) => ({
                    value,
                    label: t(`settings.common.locale.${value}`),
                  }))}
                />
                <Select
                  allowClear
                  aria-label={t("settings.dictionaries.filters.status")}
                  placeholder={t("settings.dictionaries.filters.status")}
                  value={search.item_status || undefined}
                  onChange={(value) => {
                    patchSearch({ item_status: value ?? "", item_page: 1 });
                  }}
                  options={["active", "disabled"].map((value) => ({
                    value,
                    label: t(`settings.common.status.${value}`),
                  }))}
                />
              </div>
              {items.isError ? (
                <Alert
                  showIcon
                  type="error"
                  title={t("settings.dictionaries.load_items_error")}
                />
              ) : (
                <div ref={tableRegion}>
                  <Table
                    columns={columns}
                    dataSource={items.data?.items ?? []}
                    loading={items.isPending}
                    locale={{
                      emptyText: t("settings.dictionaries.empty_items"),
                    }}
                    pagination={{
                      current: search.item_page,
                      pageSize: search.item_page_size,
                      total: items.data?.total ?? 0,
                      showSizeChanger: true,
                      hideOnSinglePage: true,
                      onChange: (item_page, item_page_size) => {
                        patchSearch({ item_page, item_page_size });
                      },
                    }}
                    rowKey="id"
                    scroll={{ x: screens.md ? 760 : 480 }}
                  />
                </div>
              )}
            </>
          )}
        </Card>
      </div>
      <Drawer
        destroyOnHidden
        open={typeEditor !== undefined}
        onClose={() => {
          setTypeEditor(undefined);
        }}
        title={t(
          typeEditor
            ? "settings.dictionaries.editor.edit_type"
            : "settings.dictionaries.editor.create_type",
        )}
      >
        <Form
          layout="vertical"
          onFinish={() => void submitType()}
          requiredMark={false}
        >
          {(["code", "name", "description"] as const).map((name) => (
            <Controller
              key={name}
              control={typeForm.control}
              name={name}
              render={({ field, fieldState }) => (
                <Form.Item
                  label={t(`settings.dictionaries.fields.${name}`)}
                  {...(fieldState.error
                    ? {
                        help: fieldState.error.message,
                        validateStatus: "error" as const,
                      }
                    : {})}
                >
                  {name === "description" ? (
                    <Input.TextArea
                      {...field}
                      aria-label={t(`settings.dictionaries.fields.${name}`)}
                      rows={3}
                    />
                  ) : (
                    <Input
                      {...field}
                      aria-label={t(`settings.dictionaries.fields.${name}`)}
                    />
                  )}
                </Form.Item>
              )}
            />
          ))}
          <Controller
            control={typeForm.control}
            name="status"
            render={({ field }) => (
              <Form.Item label={t("settings.dictionaries.fields.status")}>
                <Select
                  {...field}
                  aria-label={t("settings.dictionaries.fields.status")}
                  options={["active", "disabled"].map((value) => ({
                    value,
                    label: t(`settings.common.status.${value}`),
                  }))}
                />
              </Form.Item>
            )}
          />
          <div className="ak-drawer-actions">
            <Button
              onClick={() => {
                setTypeEditor(undefined);
              }}
            >
              {t("common.actions.cancel")}
            </Button>
            <Button
              htmlType="submit"
              loading={
                mutations.createType.isPending || mutations.updateType.isPending
              }
              type="primary"
            >
              {t(typeEditor ? "common.actions.save" : "common.actions.create")}
            </Button>
          </div>
        </Form>
      </Drawer>
      <Drawer
        destroyOnHidden
        open={itemEditor !== undefined}
        onClose={() => {
          setItemEditor(undefined);
        }}
        size="large"
        title={t(
          itemEditor
            ? "settings.dictionaries.editor.edit_item"
            : "settings.dictionaries.editor.create_item",
        )}
      >
        <Form
          layout="vertical"
          onFinish={() => void submitItem()}
          requiredMark={false}
        >
          <div className="ak-settings-form-grid">
            {(["item_value", "label", "color", "css_class"] as const).map(
              (name) => (
                <Controller
                  key={name}
                  control={itemForm.control}
                  name={name}
                  render={({ field, fieldState }) => (
                    <Form.Item
                      label={t(`settings.dictionaries.fields.${name}`)}
                      {...(fieldState.error
                        ? {
                            help: fieldState.error.message,
                            validateStatus: "error" as const,
                          }
                        : {})}
                    >
                      <Input
                        {...field}
                        aria-label={t(`settings.dictionaries.fields.${name}`)}
                      />
                    </Form.Item>
                  )}
                />
              ),
            )}
            <Controller
              control={itemForm.control}
              name="locale"
              render={({ field }) => (
                <Form.Item label={t("settings.dictionaries.fields.locale")}>
                  <Select
                    {...field}
                    aria-label={t("settings.dictionaries.fields.locale")}
                    options={["", "zh-CN", "en-US"].map((value) => ({
                      value,
                      label: t(
                        value
                          ? `settings.common.locale.${value}`
                          : "settings.common.locale.neutral",
                      ),
                    }))}
                  />
                </Form.Item>
              )}
            />
            <Controller
              control={itemForm.control}
              name="status"
              render={({ field }) => (
                <Form.Item label={t("settings.dictionaries.fields.status")}>
                  <Select
                    {...field}
                    aria-label={t("settings.dictionaries.fields.status")}
                    options={["active", "disabled"].map((value) => ({
                      value,
                      label: t(`settings.common.status.${value}`),
                    }))}
                  />
                </Form.Item>
              )}
            />
            <Controller
              control={itemForm.control}
              name="sort_order"
              render={({ field }) => (
                <Form.Item label={t("settings.dictionaries.fields.sort_order")}>
                  <InputNumber
                    {...field}
                    aria-label={t("settings.dictionaries.fields.sort_order")}
                    className="ak-full-width"
                  />
                </Form.Item>
              )}
            />
            <Controller
              control={itemForm.control}
              name="is_default"
              render={({ field }) => (
                <Form.Item label={t("settings.dictionaries.fields.is_default")}>
                  <Switch
                    aria-label={t("settings.dictionaries.fields.is_default")}
                    checked={field.value}
                    onChange={field.onChange}
                  />
                </Form.Item>
              )}
            />
          </div>
          <Controller
            control={itemForm.control}
            name="extra"
            render={({ field, fieldState }) => (
              <Form.Item
                label={t("settings.dictionaries.fields.extra")}
                {...(fieldState.error
                  ? {
                      help: fieldState.error.message,
                      validateStatus: "error" as const,
                    }
                  : {})}
              >
                <Input.TextArea
                  {...field}
                  aria-label={t("settings.dictionaries.fields.extra")}
                  autoSize={{ minRows: 4, maxRows: 10 }}
                />
              </Form.Item>
            )}
          />
          <div className="ak-drawer-actions">
            <Button
              onClick={() => {
                setItemEditor(undefined);
              }}
            >
              {t("common.actions.cancel")}
            </Button>
            <Button
              htmlType="submit"
              loading={
                mutations.createItem.isPending || mutations.updateItem.isPending
              }
              type="primary"
            >
              {t(itemEditor ? "common.actions.save" : "common.actions.create")}
            </Button>
          </div>
        </Form>
      </Drawer>
    </div>
  );
}
