import {
  Alert,
  Button,
  Card,
  Collapse,
  Drawer,
  Empty,
  Form,
  Grid,
  Input,
  InputNumber,
  List,
  Modal,
  Pagination,
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

import "../shared/i18n/settings-catalog";

import type {
  AdminDictionaryItem,
  AdminDictionaryItemWriteRequest,
  AdminDictionaryType,
  AdminDictionaryTypeWriteRequest,
} from "../generated/api/types.gen";
import { AkCreatableSelect } from "../components/AkCreatableSelect";
import { useAuthStore } from "../features/auth/store";
import {
  useAdminDictionaryItems,
  useAdminDictionaryMutations,
  useAdminDictionaryTypes,
} from "../features/settings/hooks";
import { ApiError } from "../shared/api/error";
import {
  findTenantOverride,
  groupDictionaryTypes,
} from "./dictionaryPresentation";
import {
  DICTIONARY_COLOR_PRESETS,
  DICTIONARY_STYLE_PRESETS,
  isPreviewableDictionaryColor,
  previewableDictionaryStyleClass,
} from "./dictionaryAppearance";

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
    page_size: [20, 50, 100].includes(positive(p.get("page_size"), 100))
      ? positive(p.get("page_size"), 100)
      : 100,
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
      !(key === "page_size" && value === 100) &&
      !(key === "item_page_size" && value === 20)
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
  const colorOptions = DICTIONARY_COLOR_PRESETS.map((preset) => {
    const name = t(`settings.dictionaries.appearance.colors.${preset.key}`);
    return {
      value: preset.value,
      searchText: `${name} ${preset.value}`,
      label: (
        <span className="ak-dictionary-option">
          <span
            aria-hidden="true"
            className="ak-dictionary-swatch"
            style={{ backgroundColor: preset.value }}
          />
          <span className="ak-dictionary-option-copy">
            <strong>{name}</strong>
            <code>{preset.value}</code>
          </span>
        </span>
      ),
      selectedLabel: (
        <span className="ak-dictionary-selected-option">
          <span
            aria-hidden="true"
            className="ak-dictionary-swatch"
            style={{ backgroundColor: preset.value }}
          />
          <span>{name}</span>
        </span>
      ),
    };
  });
  const styleOptions = DICTIONARY_STYLE_PRESETS.map((preset) => {
    const name = t(`settings.dictionaries.appearance.styles.${preset.key}`);
    return {
      value: preset.value,
      searchText: `${name} ${preset.value}`,
      label: (
        <span className="ak-dictionary-option">
          <span className={`ak-dictionary-style-preview ${preset.value}`}>
            {t("settings.dictionaries.appearance.preview")}
          </span>
          <span className="ak-dictionary-option-copy">
            <strong>{name}</strong>
            <code>{preset.value}</code>
          </span>
        </span>
      ),
      selectedLabel: (
        <span className="ak-dictionary-selected-option">
          <span className={`ak-dictionary-style-dot ${preset.value}`} aria-hidden="true" />
          <span>{name}</span>
        </span>
      ),
    };
  });
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
  const categoryLabels: Record<string, string> = {
    custom: t("settings.dictionaries.categories.custom"),
    notification: t("settings.dictionaries.categories.notification"),
    sms: t("settings.dictionaries.categories.sms"),
    storage: t("settings.dictionaries.categories.storage"),
    system: t("settings.dictionaries.categories.system"),
  };
  const dictionaryTypeName = (type: AdminDictionaryType) =>
    type.name_key
      ? t(type.name_key, { defaultValue: type.name })
      : type.name;
  const dictionaryTypeDescription = (type: AdminDictionaryType) =>
    type.description_key
      ? t(type.description_key, { defaultValue: type.description })
      : type.description;
  const groupedTypes = groupDictionaryTypes(
    types.data?.items ?? [],
    (code) => categoryLabels[code] ?? code,
  );
  const selectedType =
    types.data?.items.find((item) => item.id === search.type_id) ?? null;
  const canExtendSelected =
    selectedType?.extension_policy === "open" ||
    selectedType?.extension_policy === "registered" ||
    selectedType?.extension_policy === "s3_compatible";
  const items = useAdminDictionaryItems(search.type_id, {
    q: search.item_q,
    locale: search.locale,
    status: search.item_status,
    sort: "sort_order",
    page: search.item_page,
    page_size: search.item_page_size,
  });
  const creatingOverride = Boolean(itemEditor?.is_locked);
  const builtinOverrideSource = itemEditor
    ? itemEditor.is_locked
      ? itemEditor
      : items.data?.items.find(
          (candidate) =>
            candidate.is_locked &&
            candidate.item_value === itemEditor.item_value &&
            candidate.locale === itemEditor.locale,
        )
    : undefined;
  const isOverrideEditor = Boolean(builtinOverrideSource);
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
    if (types.isPending || !types.data) return;
    const lastPage = Math.max(1, Math.ceil(types.data.total / search.page_size));
    if (search.page > lastPage)
      setSearch((current) => ({ ...current, page: lastPage, type_id: "" }));
  }, [search.page, search.page_size, types.data, types.isPending]);
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
    const editable =
      findTenantOverride(item, items.data?.items ?? []) ?? item;
    itemForm.reset({
      item_value: editable.item_value,
      label: editable.label,
      locale: editable.locale ?? "",
      color: editable.color,
      css_class: editable.css_class,
      sort_order: editable.sort_order,
      is_default: editable.is_default,
      extra: JSON.stringify(editable.extra, null, 2),
      status: editable.status,
    });
    setItemEditor(editable);
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
    if (isOverrideEditor && builtinOverrideSource) {
      extra = builtinOverrideSource.extra;
    } else if (selectedType?.extension_policy === "s3_compatible") {
      extra = {
        adapter: "s3_compatible",
        provider: parsed.data.item_value.trim(),
      };
    } else {
      try {
        extra = JSON.parse(parsed.data.extra || "{}") as Record<string, unknown>;
      } catch {
        itemForm.setError("extra", {
          message: t("settings.dictionaries.validation.extra"),
        });
        return;
      }
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
      if (itemEditor && !creatingOverride)
        await mutations.updateItem.mutateAsync({ id: itemEditor.id, input });
      else
        await mutations.createItem.mutateAsync({
          typeId: search.type_id,
          input,
        });
      setItemEditor(undefined);
      setFeedback({
        key: creatingOverride
          ? "settings.dictionaries.feedback.override_created"
          : itemEditor
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
      dataIndex: "label",
      ...(screens.md ? { fixed: "left" as const } : {}),
      width: 220,
      render: (_, item) => (
        <Space wrap>
          <strong>{item.label}</strong>
          <Tag>
            {t(
              item.tenant_id
                ? "settings.dictionaries.badges.tenant_override"
                : "settings.dictionaries.badges.builtin",
            )}
          </Tag>
          {item.status === "disabled" ? (
            <Tag className="ak-status-warning">
              {t("settings.dictionaries.badges.unavailable")}
            </Tag>
          ) : null}
        </Space>
      ),
    },
    {
      title: t("settings.dictionaries.fields.item_value"),
      dataIndex: "item_value",
      ...(screens.md ? { fixed: "left" as const } : {}),
      width: 140,
      render: (value: string) => <code className="ak-settings-code">{value}</code>,
    },
    {
      title: t("settings.dictionaries.fields.locale"),
      dataIndex: "locale",
      width: 110,
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
      width: 220,
      render: (_, item) => {
        const styleClass = previewableDictionaryStyleClass(item.css_class);
        if (!item.color && !item.css_class) return "—";
        return (
          <div className="ak-dictionary-appearance-cell">
            {item.color ? (
              <span className="ak-dictionary-appearance-value">
                {isPreviewableDictionaryColor(item.color) ? (
                  <span
                    className="ak-dictionary-swatch"
                    style={{ backgroundColor: item.color }}
                    aria-hidden="true"
                  />
                ) : null}
                <code>{item.color}</code>
              </span>
            ) : null}
            {item.css_class ? (
              <span className="ak-dictionary-appearance-value">
                {styleClass ? (
                  <span className={`ak-dictionary-style-dot ${styleClass}`} aria-hidden="true" />
                ) : null}
                <code>{item.css_class}</code>
              </span>
            ) : null}
          </div>
        );
      },
    },
    {
      title: t("settings.dictionaries.fields.sort_order"),
      dataIndex: "sort_order",
      responsive: ["md"],
      width: 90,
    },
    {
      title: t("settings.dictionaries.fields.status"),
      dataIndex: "status",
      responsive: ["md"],
      width: 90,
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
      title: t("settings.dictionaries.fields.actions"),
      key: "actions",
      ...(screens.md ? { fixed: "right" as const } : {}),
      width: screens.md ? 130 : 100,
      render: (_, item) => {
        const canEdit = item.is_locked
          ? canExtendSelected && permissions.has("sys.dictionary.create")
          : permissions.has("sys.dictionary.update");
        return canEdit ||
          (!item.is_locked && permissions.has("sys.dictionary.delete")) ? (
          <Space>
            {canEdit ? (
              <Button
                type="link"
                onClick={() => {
                  openItemEdit(item);
                }}
              >
                {t("common.actions.edit")}
              </Button>
            ) : null}
            {!item.is_locked && permissions.has("sys.dictionary.delete") ? (
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
        ) : (
          "—"
        );
      },
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
          ) : types.isPending ? (
            <List loading dataSource={[]} />
          ) : groupedTypes.length === 0 ? (
            <Empty description={t("settings.dictionaries.empty_types")} />
          ) : (
            <>
              <Collapse
                className="ak-dictionary-categories"
                defaultActiveKey={groupedTypes.map((category) => category.code)}
                items={groupedTypes.map((category) => ({
                  key: category.code,
                  label: (
                    <span className="ak-dictionary-category-heading">
                      <span>
                        <strong>{category.label}</strong>
                        <code>{category.code}</code>
                      </span>
                      <Tag>
                        {t("settings.dictionaries.category_count", {
                          count: category.types.length,
                        })}
                      </Tag>
                    </span>
                  ),
                  children: (
                    <List
                      dataSource={category.types}
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
                              patchSearch({
                                type_id: type.id,
                                item_page: 1,
                              });
                            }}
                          >
                            <span className="ak-dictionary-type-title">
                              <Space wrap size={[4, 4]}>
                                <span>{dictionaryTypeName(type)}</span>
                                {type.is_locked ? (
                                  <Tag>
                                    {t("settings.dictionaries.badges.builtin")}
                                  </Tag>
                                ) : null}
                                <Tag>
                                  {t(
                                    `settings.dictionaries.policy.${type.extension_policy}`,
                                  )}
                                </Tag>
                                {type.visibility === "public" ? (
                                  <Tag color="cyan">
                                    {t("settings.dictionaries.badges.public")}
                                  </Tag>
                                ) : null}
                                {type.status === "disabled" ? (
                                  <Tag className="ak-status-warning">
                                    {t(
                                      "settings.dictionaries.badges.unavailable",
                                    )}
                                  </Tag>
                                ) : null}
                              </Space>
                            </span>
                            <span className="ak-settings-code">
                              {type.code}
                            </span>
                          </button>
                        </List.Item>
                      )}
                    />
                  ),
                }))}
              />
              <Pagination
                className="ak-dictionary-type-pagination"
                current={search.page}
                pageSize={search.page_size}
                total={types.data.total}
                size="small"
                hideOnSinglePage
                onChange={(page) => {
                  patchSearch({ page, type_id: "" });
                }}
              />
            </>
          )}
        </Card>
        <Card
          className="ak-dictionary-detail"
          title={selectedType ? dictionaryTypeName(selectedType) : t("settings.dictionaries.select_type")}
          extra={
            selectedType &&
            canExtendSelected &&
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
              <Typography.Paragraph type="secondary">
                {dictionaryTypeDescription(selectedType)}
              </Typography.Paragraph>
              {selectedType.is_locked && !canExtendSelected ? (
                <Alert
                  className="ak-settings-lock-alert"
                  showIcon
                  type="info"
                  title={t("settings.dictionaries.locked.title")}
                  description={t("settings.dictionaries.locked.description")}
                />
              ) : null}
              {canExtendSelected ? (
                <Alert
                  className="ak-settings-lock-alert"
                  showIcon
                  type="info"
                  title={t("settings.dictionaries.extensible.title")}
                  description={t("settings.dictionaries.extensible.description")}
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
                    scroll={{ x: screens.md ? 900 : 680 }}
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
          isOverrideEditor
            ? "settings.dictionaries.editor.override_item"
            : itemEditor
            ? "settings.dictionaries.editor.edit_item"
            : "settings.dictionaries.editor.create_item",
        )}
      >
        <Form
          layout="vertical"
          onFinish={() => void submitItem()}
          requiredMark={false}
        >
          {isOverrideEditor ? (
            <Alert
              className="ak-settings-lock-alert"
              showIcon
              type="info"
              title={t("settings.dictionaries.override.title")}
              description={t("settings.dictionaries.override.description")}
            />
          ) : null}
          <div className="ak-settings-form-grid">
            {(["item_value", "label"] as const).map(
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
                        disabled={isOverrideEditor && name === "item_value"}
                        aria-label={t(`settings.dictionaries.fields.${name}`)}
                      />
                    </Form.Item>
                  )}
                />
              ),
            )}
            <Controller
              control={itemForm.control}
              name="color"
              render={({ field, fieldState }) => (
                <Form.Item
                  label={t("settings.dictionaries.fields.color")}
                  help={
                    fieldState.error?.message ??
                    t("settings.dictionaries.appearance.color_hint")
                  }
                  {...(fieldState.error
                    ? { validateStatus: "error" as const }
                    : {})}
                >
                  <AkCreatableSelect
                    aria-label={t("settings.dictionaries.fields.color")}
                    defaultLabel={t("settings.dictionaries.appearance.default")}
                    onBlur={field.onBlur}
                    onChange={field.onChange}
                    options={colorOptions}
                    value={field.value}
                  />
                </Form.Item>
              )}
            />
            <Controller
              control={itemForm.control}
              name="css_class"
              render={({ field, fieldState }) => (
                <Form.Item
                  label={t("settings.dictionaries.fields.css_class")}
                  help={
                    fieldState.error?.message ??
                    t("settings.dictionaries.appearance.css_class_hint")
                  }
                  {...(fieldState.error
                    ? { validateStatus: "error" as const }
                    : {})}
                >
                  <AkCreatableSelect
                    aria-label={t("settings.dictionaries.fields.css_class")}
                    defaultLabel={t("settings.dictionaries.appearance.default")}
                    onBlur={field.onBlur}
                    onChange={field.onChange}
                    options={styleOptions}
                    value={field.value}
                  />
                </Form.Item>
              )}
            />
            <Controller
              control={itemForm.control}
              name="locale"
              render={({ field }) => (
                <Form.Item label={t("settings.dictionaries.fields.locale")}>
                  <Select
                    {...field}
                    disabled={isOverrideEditor}
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
          {selectedType?.extension_policy === "s3_compatible" ? (
            <Alert
              showIcon
              type="info"
              title={t("settings.dictionaries.s3.title")}
              description={t("settings.dictionaries.s3.description")}
            />
          ) : isOverrideEditor ? null : (
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
          )}
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
