import {
  Button,
  Card,
  Checkbox,
  Drawer,
  Form,
  Grid,
  Input,
  InputNumber,
  Modal,
  Segmented,
  Select,
  Skeleton,
  Space,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { FormOutlined, SaveOutlined, TableOutlined } from "@ant-design/icons";
import { useEffect, useMemo, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";

import type {
  AdminConfigItem,
  AdminConfigWriteRequestWritable,
} from "../generated/api/types.gen";
import { useAuthStore } from "../features/auth/store";
import {
  type AdminConfigSaveOperation,
  useAdminConfigMutations,
  useAdminConfigs,
} from "../features/settings/hooks";
import { ApiError } from "../shared/api/error";
import { AkFileUploader } from "../components/AkFileUploader";
import {
  AkConfigValueField,
  type AkConfigDraftValue,
  configDraftValue,
  parseConfigDraftValue,
  validateConfigDraftValue,
} from "../components/AkConfigValueField";

type ValueType = AdminConfigItem["value_type"];
type ConfigMode = "form" | "table";
interface SearchState {
  mode: ConfigMode;
  q: string;
  module_code: string;
  config_group: string;
  value_type: ValueType | "";
  status: "active" | "disabled" | "";
  scope: "" | "public" | "secret";
  sort: "sort_order" | "updated_desc" | "key";
  page: number;
  page_size: number;
}
interface ConfigValues {
  module_code: string;
  config_group: string;
  config_key: string;
  display_name: string;
  value_type: ValueType;
  value_text: string;
  default_value_text: string;
  secret_value: string;
  is_secret: boolean;
  is_public: boolean;
  validation_schema: string;
  description: string;
  sort_order: number;
  status: "active" | "disabled";
}
interface SecretValues {
  secret_value: string;
}
interface Feedback {
  key: string;
  error: boolean;
  values?: Record<string, number | string>;
}
type DirectDraft = Record<string, AkConfigDraftValue>;
type PendingNavigation =
  | { mode: ConfigMode; type: "mode" }
  | { config_group: string; module_code: string; type: "category" };

const configCategories = [
  ["core", "basic", "basic"],
  ["notifications", "email", "email"],
  ["notifications", "sms", "sms"],
  ["iam", "registration", "registration"],
  ["iam", "security", "security"],
  ["billing", "withdrawal", "withdrawal"],
  ["storage", "cloud", "cloud"],
  ["integrations", "location", "location"],
  ["billing", "payment", "payment"],
  ["integrations", "wechat", "wechat"],
] as const;

const configSchema = z
  .object({
    module_code: z.string().trim().min(1).max(64),
    config_group: z.string().trim().min(1).max(96),
    config_key: z
      .string()
      .trim()
      .min(1)
      .max(160)
      .regex(/^[a-zA-Z0-9][a-zA-Z0-9._-]*$/),
    display_name: z.string().trim().min(1).max(160),
    value_type: z.enum([
      "string",
      "integer",
      "decimal",
      "boolean",
      "json",
      "datetime",
    ]),
    value_text: z.string(),
    default_value_text: z.string(),
    secret_value: z.string().max(16384),
    is_secret: z.boolean(),
    is_public: z.boolean(),
    validation_schema: z.string(),
    description: z.string().max(1000),
    sort_order: z.number().int(),
    status: z.enum(["active", "disabled"]),
  })
  .refine((value) => !(value.is_secret && value.is_public), {
    path: ["is_public"],
  });
const secretSchema = z.object({
  secret_value: z.string().trim().min(1).max(16384),
});
const defaults: ConfigValues = {
  module_code: "core",
  config_group: "default",
  config_key: "",
  display_name: "",
  value_type: "string",
  value_text: "",
  default_value_text: "",
  secret_value: "",
  is_secret: false,
  is_public: false,
  validation_schema: "{}",
  description: "",
  sort_order: 0,
  status: "active",
};

function positive(value: string | null, fallback: number) {
  const parsed = Number(value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback;
}
function readSearch(): SearchState {
  const p = new URLSearchParams(location.search);
  const type = p.get("value_type");
  const status = p.get("status");
  const scope = p.get("scope");
  const sort = p.get("sort");
  return {
    mode: p.get("mode") === "table" ? "table" : "form",
    q: p.get("q") ?? "",
    module_code: p.get("module_code") ?? "core",
    config_group: p.get("config_group") ?? "basic",
    value_type: [
      "string",
      "integer",
      "decimal",
      "boolean",
      "json",
      "datetime",
    ].includes(type ?? "")
      ? (type as ValueType)
      : "",
    status: status === "active" || status === "disabled" ? status : "",
    scope: scope === "public" || scope === "secret" ? scope : "",
    sort: sort === "updated_desc" || sort === "key" ? sort : "sort_order",
    page: positive(p.get("page"), 1),
    page_size: [10, 20, 50, 100].includes(positive(p.get("page_size"), 20))
      ? positive(p.get("page_size"), 20)
      : 20,
  };
}
function persistSearch(search: SearchState) {
  const p = new URLSearchParams();
  for (const [key, value] of Object.entries(search))
    if (
      value !== "" &&
      !(key === "page" && value === 1) &&
      !(key === "page_size" && value === 20) &&
      !(key === "sort" && value === "sort_order") &&
      !(key === "mode" && value === "form")
    )
      p.set(key, String(value));
  history.replaceState(
    history.state,
    "",
    `${location.pathname}${p.size ? `?${p}` : ""}`,
  );
}
function textValue(value: unknown, type: ValueType): string {
  if (value === null || value === undefined) return "";
  if ((type === "string" || type === "datetime") && typeof value === "string")
    return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  return typeof value === "object" ? JSON.stringify(value, null, 2) : "";
}
function parseValue(text: string, type: ValueType): unknown {
  if (!text.trim()) return undefined;
  if (type === "string" || type === "datetime") return text;
  if (type === "boolean") {
    if (text === "true") return true;
    if (text === "false") return false;
    throw new Error("boolean");
  }
  if (type === "integer") {
    const value = Number(text);
    if (!Number.isInteger(value)) throw new Error("integer");
    return value;
  }
  if (type === "decimal") {
    const value = Number(text);
    if (!Number.isFinite(value)) throw new Error("decimal");
    return value;
  }
  return JSON.parse(text) as unknown;
}
function editorValues(item: AdminConfigItem): ConfigValues {
  return {
    module_code: item.module_code,
    config_group: item.config_group,
    config_key: item.config_key,
    display_name: item.display_name,
    value_type: item.value_type,
    value_text: textValue(item.value, item.value_type),
    default_value_text: textValue(item.default_value, item.value_type),
    secret_value: "",
    is_secret: item.is_secret,
    is_public: item.is_public,
    validation_schema: JSON.stringify(item.validation_schema, null, 2),
    description: item.description,
    sort_order: item.sort_order,
    status: item.status,
  };
}
function catalogItem(item: AdminConfigItem | null | undefined) {
  if (!item) return false;
  return (item.validation_schema as Record<string, unknown>)["x-appkernia-catalog"] === true;
}

function directDraft(items: AdminConfigItem[]): DirectDraft {
  return Object.fromEntries(
    items.map((item) => [item.id, configDraftValue(item)]),
  );
}

function draftEqual(left: AkConfigDraftValue, right: AkConfigDraftValue) {
  return left === right;
}

function draftFor(item: AdminConfigItem, values: DirectDraft) {
  const value = values[item.id];
  return value === undefined ? configDraftValue(item) : value;
}

function configWriteInput(
  item: AdminConfigItem,
  value: AkConfigDraftValue,
): AdminConfigWriteRequestWritable {
  return {
    module_code: item.module_code,
    config_group: item.config_group,
    config_key: item.config_key,
    display_name: item.display_name,
    value_type: item.value_type,
    value: parseConfigDraftValue(item, value),
    default_value: item.default_value,
    is_secret: false,
    is_public: item.is_public,
    validation_schema: item.validation_schema,
    description: item.description,
    sort_order: item.sort_order,
    status: item.status,
    version: item.version,
  };
}

export function AdminConfigsPage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(
    useAuthStore((state) => state.context?.permissions ?? []),
  );
  const initial = useMemo(readSearch, []);
  const [search, setSearch] = useState(initial);
  const [editing, setEditing] = useState<AdminConfigItem | null | undefined>(
    undefined,
  );
  const [rotating, setRotating] = useState<AdminConfigItem | null>(null);
  const [feedback, setFeedback] = useState<Feedback | null>(null);
  const [directValues, setDirectValues] = useState<DirectDraft>({});
  const [directInitial, setDirectInitial] = useState<DirectDraft>({});
  const [directErrors, setDirectErrors] = useState<Record<string, string>>({});
  const [pendingNavigation, setPendingNavigation] =
    useState<PendingNavigation | null>(null);
  const tableRegion = useRef<HTMLDivElement>(null);
  const directSaving = useRef(false);
  const query = useAdminConfigs(search.mode === "form" ? {
    module_code: search.module_code,
    config_group: search.config_group,
    sort: "sort_order",
    page: 1,
    page_size: 100,
  } : {
    q: search.q,
    module_code: search.module_code,
    config_group: search.config_group,
    value_type: search.value_type,
    status: search.status,
    ...(search.scope === "public" ? { is_public: true } : {}),
    ...(search.scope === "secret" ? { is_secret: true } : {}),
    sort: search.sort,
    page: search.page,
    page_size: search.page_size,
  });
  const mutations = useAdminConfigMutations();
  const form = useForm<ConfigValues>({ defaultValues: defaults });
  const secretForm = useForm<SecretValues>({
    defaultValues: { secret_value: "" },
  });
  const isSecret = form.watch("is_secret");
  const catalogEditing = catalogItem(editing);
  const directItems = useMemo(
    () => (search.mode === "form" ? (query.data?.items ?? []) : []),
    [query.data?.items, search.mode],
  );
  const visibleDirectItems = useMemo(() => {
    const valuesByKey = new Map(
      directItems.map((item) => [item.config_key, draftFor(item, directValues)]),
    );
    return directItems.filter((item) => {
      const condition = (item.validation_schema as Record<string, unknown>)[
        "x-visible-when"
      ];
      if (!condition || typeof condition !== "object") return true;
      const rule = condition as { equals?: unknown; key?: unknown };
      return typeof rule.key === "string" && valuesByKey.get(rule.key) === rule.equals;
    });
  }, [directItems, directValues]);
  const directDirtyIds = useMemo(
    () =>
      directItems
        .filter((item) => {
          const value = directValues[item.id];
          if (value === undefined) return false;
          return !draftEqual(
            value,
            draftFor(item, directInitial),
          );
        })
        .map((item) => item.id),
    [directInitial, directItems, directValues],
  );
  const hasDirectChanges = directDirtyIds.length > 0;
  useEffect(() => {
    persistSearch(search);
  }, [search]);
  useEffect(() => {
    const scrollable =
      tableRegion.current?.querySelector<HTMLElement>(".ant-table-content");
    if (scrollable) scrollable.tabIndex = 0;
  }, [query.data, screens.md, search.mode]);
  useEffect(() => {
    if (
      search.mode !== "form" ||
      query.isPlaceholderData ||
      directSaving.current ||
      hasDirectChanges
    )
      return;
    const next = directDraft(query.data?.items ?? []);
    setDirectValues(next);
    setDirectInitial(next);
    setDirectErrors({});
  }, [
    hasDirectChanges,
    query.data?.items,
    query.isPlaceholderData,
    search.config_group,
    search.mode,
    search.module_code,
  ]);
  useEffect(() => {
    if (!hasDirectChanges) return undefined;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
    };
    window.addEventListener("beforeunload", warn);
    return () => {
      window.removeEventListener("beforeunload", warn);
    };
  }, [hasDirectChanges]);
  const patchSearch = (patch: Partial<SearchState>) => {
    setSearch((current) => ({ ...current, ...patch }));
  };
  const resetDirectForm = (items = directItems) => {
    const next = directDraft(items);
    setDirectValues(next);
    setDirectInitial(next);
    setDirectErrors({});
  };
  const applyCategory = (module_code: string, config_group: string) => {
    resetDirectForm([]);
    setFeedback(null);
    patchSearch({ module_code, config_group, page: 1 });
  };
  const requestCategory = (module_code: string, config_group: string) => {
    if (hasDirectChanges)
      setPendingNavigation({ type: "category", module_code, config_group });
    else applyCategory(module_code, config_group);
  };
  const applyMode = (mode: ConfigMode) => {
    resetDirectForm([]);
    setFeedback(null);
    patchSearch({ mode, page: 1 });
  };
  const requestMode = (mode: ConfigMode) => {
    if (mode === search.mode) return;
    if (hasDirectChanges) setPendingNavigation({ type: "mode", mode });
    else applyMode(mode);
  };
  const errorKey = (error: unknown) =>
    error instanceof ApiError ? error.messageKey : "errors.common.unknown";
  const itemLabel = (item: AdminConfigItem) => t(`settings.configItems.${item.config_key}`, { defaultValue: item.display_name });
  const openCreate = () => {
    form.reset(defaults);
    setEditing(null);
    setFeedback(null);
  };
  const openEdit = (item: AdminConfigItem) => {
    form.reset(editorValues(item));
    setEditing(item);
    setFeedback(null);
  };
  const closeEditor = () => {
    setEditing(undefined);
    form.reset(defaults);
  };
  const submit = form.handleSubmit(async (raw) => {
    const parsed = configSchema.safeParse(raw);
    if (!parsed.success) {
      for (const issue of parsed.error.issues)
        form.setError(issue.path[0] as keyof ConfigValues, {
          message: t("validation.invalid", {
            name: t(`settings.configs.fields.${String(issue.path[0])}`),
          }),
        });
      return;
    }
    if (!editing && parsed.data.is_secret && !parsed.data.secret_value.trim()) {
      form.setError("secret_value", {
        message: t("validation.required", {
          name: t("settings.configs.fields.secret_value"),
        }),
      });
      return;
    }
    let schema: Record<string, unknown>, value: unknown, defaultValue: unknown;
    try {
      schema = JSON.parse(parsed.data.validation_schema || "{}") as Record<
        string,
        unknown
      >;
      value = parsed.data.is_secret
        ? undefined
        : parseValue(parsed.data.value_text, parsed.data.value_type);
      defaultValue = parsed.data.is_secret
        ? undefined
        : parseValue(parsed.data.default_value_text, parsed.data.value_type);
    } catch {
      form.setError("value_text", {
        message: t("settings.configs.validation.value"),
      });
      return;
    }
    const input: AdminConfigWriteRequestWritable = {
      module_code: parsed.data.module_code.trim(),
      config_group: parsed.data.config_group.trim(),
      config_key: parsed.data.config_key.trim(),
      display_name: parsed.data.display_name.trim(),
      value_type: parsed.data.value_type,
      ...(value !== undefined ? { value } : {}),
      ...(defaultValue !== undefined ? { default_value: defaultValue } : {}),
      ...(!editing && parsed.data.is_secret
        ? { secret_value: parsed.data.secret_value.trim() }
        : {}),
      is_secret: parsed.data.is_secret,
      is_public: parsed.data.is_public,
      validation_schema: schema,
      description: parsed.data.description.trim(),
      sort_order: parsed.data.sort_order,
      status: parsed.data.status,
      version: editing?.version ?? 1,
    };
    try {
      if (editing)
        await mutations.update.mutateAsync({ id: editing.id, input });
      else await mutations.create.mutateAsync(input);
      closeEditor();
      setFeedback({
        key: editing
          ? "settings.configs.feedback.updated"
          : "settings.configs.feedback.created",
        error: false,
      });
    } catch (error) {
      setFeedback({ key: errorKey(error), error: true });
    }
  });
  const rotate = secretForm.handleSubmit(async (raw) => {
    const parsed = secretSchema.safeParse(raw);
    if (!parsed.success || !rotating) {
      secretForm.setError("secret_value", {
        message: t("validation.required", {
          name: t("settings.configs.fields.secret_value"),
        }),
      });
      return;
    }
    try {
      await mutations.rotate.mutateAsync({
        id: rotating.id,
        input: {
          secret_value: parsed.data.secret_value,
          version: rotating.version,
        },
      });
      setRotating(null);
      secretForm.reset();
      setFeedback({ key: "settings.configs.feedback.rotated", error: false });
    } catch (error) {
      setFeedback({ key: errorKey(error), error: true });
    }
  });
  const saveDirectValues = async () => {
    const dirtySet = new Set(directDirtyIds);
    const errors: Record<string, string> = {};
    const operations: AdminConfigSaveOperation[] = [];
    const submitted = { ...directValues };
    for (const item of directItems) {
      if (!dirtySet.has(item.id)) continue;
      const value = draftFor(item, directValues);
      const error = validateConfigDraftValue(item, value, t);
      if (error) {
        errors[item.id] = error;
        continue;
      }
      if (item.is_secret) {
        const secret = parseConfigDraftValue(item, value);
        if (typeof secret !== "string" || !secret) continue;
        operations.push({
          id: item.id,
          kind: "secret",
          input: { secret_value: secret, version: item.version },
        });
      } else {
        operations.push({
          id: item.id,
          kind: "update",
          input: configWriteInput(item, value),
        });
      }
    }
    setDirectErrors(errors);
    if (Object.keys(errors).length > 0) {
      setFeedback({
        key: "settings.configs.form.feedback.validation_failed",
        error: true,
        values: { count: Object.keys(errors).length },
      });
      return;
    }
    if (operations.length === 0) {
      resetDirectForm();
      setFeedback({
        key: "settings.configs.form.feedback.no_changes",
        error: false,
      });
      return;
    }
    directSaving.current = true;
    setFeedback(null);
    try {
      const result = await mutations.saveMany.mutateAsync(operations);
      const failedIds = new Set(result.failures.map(({ id }) => id));
      const refreshed = await query.refetch();
      const items = refreshed.data?.items ?? directItems;
      const initialValues = directDraft(items);
      const nextValues = { ...initialValues };
      for (const id of failedIds) {
        const value = submitted[id];
        if (value !== undefined) nextValues[id] = value;
      }
      setDirectInitial(initialValues);
      setDirectValues(nextValues);
      if (result.failures.length === 0) {
        setFeedback({
          key: "settings.configs.form.feedback.saved",
          error: false,
          values: { count: result.successIds.length },
        });
      } else {
        setFeedback({
          key: "settings.configs.form.feedback.partial",
          error: true,
          values: {
            failed: result.failures.length,
            saved: result.successIds.length,
          },
        });
      }
    } catch (error) {
      setFeedback({ key: errorKey(error), error: true });
    } finally {
      directSaving.current = false;
    }
  };
  const columns: TableColumnsType<AdminConfigItem> = [
    {
      title: t("settings.configs.fields.key"),
      key: "key",
      render: (_, item) => (
        <div>
          <strong>{itemLabel(item)}</strong>
          <div className="ak-settings-code">
            {item.module_code}.{item.config_group}.{item.config_key}
          </div>
        </div>
      ),
    },
    {
      title: t("settings.configs.fields.value"),
      key: "value",
      render: (_, item) =>
        item.is_secret ? (
          <Tag
            className={
              item.secret_configured ? "ak-status-success" : "ak-status-warning"
            }
          >
            {t(
              item.secret_configured
                ? "settings.configs.secret.configured"
                : "settings.configs.secret.missing",
            )}
          </Tag>
        ) : (
          <span className="ak-settings-value">
            {textValue(item.value, item.value_type) || "—"}
          </span>
        ),
    },
    {
      title: t("settings.configs.fields.type"),
      dataIndex: "value_type",
      responsive: ["md"],
      render: (value: ValueType) => t(`settings.configs.types.${value}`),
    },
    {
      title: t("settings.configs.fields.scope"),
      key: "scope",
      responsive: ["lg"],
      render: (_, item) => (
        <Space wrap>
          {item.is_locked ? (
            <Tag>{t("settings.common.system_locked")}</Tag>
          ) : (
            <Tag color="blue">{t("settings.common.tenant")}</Tag>
          )}
          {item.is_public ? (
            <Tag className="ak-config-public-tag" color="cyan">
              {t("settings.configs.scope.public")}
            </Tag>
          ) : null}
        </Space>
      ),
    },
    {
      title: t("settings.configs.fields.status"),
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
      width: screens.md ? 170 : 110,
      render: (_, item) =>
        item.is_locked ? (
          "—"
        ) : (
          <Space>
            {permissions.has("sys.config.update") && !item.is_secret ? (
              <Button
                type="link"
                onClick={() => {
                  openEdit(item);
                }}
              >
                {t("common.actions.edit")}
              </Button>
            ) : null}
            {item.is_secret && permissions.has("sys.config.rotate_secret") ? (
              <Button
                type="link"
                onClick={() => {
                  secretForm.reset();
                  setRotating(item);
                }}
              >
                {t("settings.configs.actions.rotate")}
              </Button>
            ) : null}
          </Space>
        ),
    },
  ];
  const categoryKey =
    configCategories.find(
      ([moduleCode, group]) =>
        moduleCode === search.module_code && group === search.config_group,
    )?.[2] ?? "basic";
  return (
    <div className="ak-page-container">
      <header className="ak-page-heading ak-users-heading">
        <div>
          <Typography.Title level={1}>
            {t("routes.system.settings.configs.title")}
          </Typography.Title>
          <Typography.Paragraph type="secondary">
            {t("settings.configs.description")}
          </Typography.Paragraph>
        </div>
        <div className="ak-config-heading-actions">
          <Segmented<ConfigMode>
            aria-label={t("settings.configs.mode.label")}
            options={[
              {
                label: (
                  <Space size={6}>
                    <FormOutlined aria-hidden="true" />
                    {t("settings.configs.mode.form")}
                  </Space>
                ),
                value: "form",
              },
              {
                label: (
                  <Space size={6}>
                    <TableOutlined aria-hidden="true" />
                    {t("settings.configs.mode.table")}
                  </Space>
                ),
                value: "table",
              },
            ]}
            value={search.mode}
            onChange={requestMode}
          />
          {search.mode === "table" && permissions.has("sys.config.create") ? (
            <Button type="primary" onClick={openCreate}>
              {t("settings.configs.actions.create")}
            </Button>
          ) : null}
        </div>
      </header>
      {feedback ? (
        <div
          className={feedback.error ? "ak-form-error" : "ak-org-feedback"}
          role={feedback.error ? "alert" : "status"}
        >
          {feedback.values
            ? t(feedback.key, feedback.values)
            : t(feedback.key)}
        </div>
      ) : null}
      <div className="ak-config-layout">
        <aside className="ak-config-categories" aria-label={t("settings.configs.categories.label")}>
          {screens.md ? <Card size="small">
            <nav className="ak-config-category-list">
              {configCategories.map(([moduleCode, group, key]) => {
                const selected = search.module_code === moduleCode && search.config_group === group;
                return <button type="button" key={`${moduleCode}.${group}`} className={selected ? "ak-config-category is-active" : "ak-config-category"} aria-current={selected ? "page" : undefined} onClick={() => { requestCategory(moduleCode, group); }}>
                  <strong>{t(`settings.configCategories.${key}.name`)}</strong>
                  <span>{t(`settings.configCategories.${key}.description`)}</span>
                </button>;
              })}
            </nav>
          </Card> : <Select className="ak-config-category-select" aria-label={t("settings.configs.categories.label")} value={`${search.module_code}.${search.config_group}`} onChange={(value) => { const [moduleCode = "core", group = "basic"] = value.split("."); requestCategory(moduleCode, group); }} options={configCategories.map(([moduleCode, group, key]) => ({ value: `${moduleCode}.${group}`, label: t(`settings.configCategories.${key}.name`) }))} />}
        </aside>
        <div className="ak-config-content">
          <Card title={t(`settings.configCategories.${categoryKey}.name`)}>
            {search.mode === "form" ? (
              <>
                <div className="ak-config-form-toolbar">
                  <div aria-live="polite">
                    <Typography.Paragraph type="secondary">
                      {t("settings.configs.form.description")}
                    </Typography.Paragraph>
                    {hasDirectChanges ? (
                      <Typography.Text strong>
                        {t("settings.configs.form.changed", {
                          count: directDirtyIds.length,
                        })}
                      </Typography.Text>
                    ) : null}
                  </div>
                  <Space wrap>
                    <Button
                      disabled={!hasDirectChanges || mutations.saveMany.isPending}
                      onClick={() => {
                        resetDirectForm();
                      }}
                    >
                      {t("settings.configs.form.reset")}
                    </Button>
                    <Button
                      disabled={!hasDirectChanges}
                      icon={<SaveOutlined aria-hidden="true" />}
                      loading={mutations.saveMany.isPending}
                      type="primary"
                      onClick={() => void saveDirectValues()}
                    >
                      {t("settings.configs.form.save")}
                    </Button>
                  </Space>
                </div>
                {query.isError ? (
                  <div className="ak-form-error" role="alert">
                    {t("settings.configs.load_error")} {" "}
                    <Button onClick={() => void query.refetch()}>
                      {t("common.actions.retry")}
                    </Button>
                  </div>
                ) : null}
                {query.isPending || query.isPlaceholderData ? (
                  <Skeleton active paragraph={{ rows: 8 }} />
                ) : (
                  <Form layout="vertical" requiredMark={false}>
                    <div className="ak-config-direct-grid">
                      {visibleDirectItems.map((item) => (
                        <AkConfigValueField
                          disabled={
                            item.is_locked ||
                            item.status === "disabled" ||
                            (item.is_secret
                              ? !permissions.has("sys.config.rotate_secret")
                              : !permissions.has("sys.config.update"))
                          }
                          error={directErrors[item.id]}
                          item={item}
                          key={item.id}
                          label={itemLabel(item)}
                          t={t}
                          value={draftFor(item, directValues)}
                          onChange={(value) => {
                            setDirectValues((current) => ({
                              ...current,
                              [item.id]: value,
                            }));
                            setDirectErrors((current) => {
                              if (!current[item.id]) return current;
                              return Object.fromEntries(
                                Object.entries(current).filter(
                                  ([id]) => id !== item.id,
                                ),
                              );
                            });
                            setFeedback(null);
                          }}
                        />
                      ))}
                    </div>
                  </Form>
                )}
                {search.module_code === "storage" &&
                search.config_group === "cloud" &&
                permissions.has("storage.file.upload") ? (
                  <div className="ak-config-storage-test">
                    <Typography.Paragraph type="secondary">
                      {t("settings.configs.cloud.test_upload_description")}
                    </Typography.Paragraph>
                    <AkFileUploader compact />
                  </div>
                ) : null}
              </>
            ) : (
              <>
        <div className="ak-settings-filters" role="search">
          <Input.Search
            allowClear
            aria-label={t("settings.configs.filters.query")}
            placeholder={t("settings.configs.filters.query")}
            value={search.q}
            onChange={(event) => {
              patchSearch({ q: event.target.value, page: 1 });
            }}
          />
          <Select
            allowClear
            aria-label={t("settings.configs.filters.type")}
            placeholder={t("settings.configs.filters.type")}
            value={search.value_type || undefined}
            onChange={(value) => {
              patchSearch({ value_type: value ?? "", page: 1 });
            }}
            options={(
              [
                "string",
                "integer",
                "decimal",
                "boolean",
                "json",
                "datetime",
              ] as const
            ).map((value) => ({
              value,
              label: t(`settings.configs.types.${value}`),
            }))}
          />
          <Select
            allowClear
            aria-label={t("settings.configs.filters.scope")}
            placeholder={t("settings.configs.filters.scope")}
            value={search.scope || undefined}
            onChange={(value) => {
              patchSearch({ scope: value ?? "", page: 1 });
            }}
            options={["public", "secret"].map((value) => ({
              value,
              label: t(`settings.configs.scope.${value}`),
            }))}
          />
          <Select
            allowClear
            aria-label={t("settings.configs.filters.status")}
            placeholder={t("settings.configs.filters.status")}
            value={search.status || undefined}
            onChange={(value) => {
              patchSearch({ status: value ?? "", page: 1 });
            }}
            options={["active", "disabled"].map((value) => ({
              value,
              label: t(`settings.common.status.${value}`),
            }))}
          />
        </div>
        {query.isError ? (
          <div className="ak-form-error" role="alert">
            {t("settings.configs.load_error")}{" "}
            <Button onClick={() => void query.refetch()}>
              {t("common.actions.retry")}
            </Button>
          </div>
        ) : null}
        <div ref={tableRegion}>
          <Table
            columns={columns}
            dataSource={query.data?.items ?? []}
            loading={query.isPending}
            locale={{ emptyText: t("settings.configs.empty") }}
            pagination={{
              current: search.page,
              pageSize: search.page_size,
              total: query.data?.total ?? 0,
              showSizeChanger: true,
              onChange: (page, page_size) => {
                patchSearch({ page, page_size });
              },
            }}
            rowKey="id"
            scroll={{ x: screens.md ? 980 : 520 }}
          />
        </div>
              </>
            )}
          </Card>
        </div>
      </div>
      <Modal
        cancelText={t("common.actions.cancel")}
        okText={t("settings.configs.form.discard")}
        open={Boolean(pendingNavigation)}
        title={t("settings.configs.form.unsaved_title")}
        onCancel={() => {
          setPendingNavigation(null);
        }}
        onOk={() => {
          const pending = pendingNavigation;
          setPendingNavigation(null);
          if (!pending) return;
          if (pending.type === "mode") applyMode(pending.mode);
          else applyCategory(pending.module_code, pending.config_group);
        }}
      >
        <Typography.Paragraph>
          {t("settings.configs.form.unsaved_description")}
        </Typography.Paragraph>
      </Modal>
      <Drawer
        destroyOnHidden
        onClose={closeEditor}
        open={editing !== undefined}
        size="large"
        title={t(
          editing
            ? "settings.configs.editor.edit_title"
            : "settings.configs.editor.create_title",
        )}
      >
        <Form
          layout="vertical"
          onFinish={() => void submit()}
          requiredMark={false}
        >
          <div className="ak-settings-form-grid">
            {(
              [
                "module_code",
                "config_group",
                "config_key",
                "display_name",
              ] as const
            ).map((name) => (
              <Controller
                key={name}
                control={form.control}
                name={name}
                render={({ field, fieldState }) => (
                  <Form.Item
                    label={t(`settings.configs.fields.${name}`)}
                    {...(fieldState.error
                      ? {
                          help: fieldState.error.message,
                          validateStatus: "error" as const,
                        }
                      : {})}
                  >
                    <Input
                      {...field}
                      aria-label={t(`settings.configs.fields.${name}`)}
                      disabled={Boolean(
                        catalogEditing || (editing?.is_secret && name === "config_key"),
                      )}
                    />
                  </Form.Item>
                )}
              />
            ))}
            <Controller
              control={form.control}
              name="value_type"
              render={({ field }) => (
                <Form.Item label={t("settings.configs.fields.value_type")}>
                  <Select
                    {...field}
                    aria-label={t("settings.configs.fields.value_type")}
                    disabled={Boolean(editing?.is_secret) || catalogEditing}
                    options={(
                      [
                        "string",
                        "integer",
                        "decimal",
                        "boolean",
                        "json",
                        "datetime",
                      ] as const
                    ).map((value) => ({
                      value,
                      label: t(`settings.configs.types.${value}`),
                    }))}
                  />
                </Form.Item>
              )}
            />
            <Controller
              control={form.control}
              name="status"
              render={({ field }) => (
                <Form.Item label={t("settings.configs.fields.status")}>
                  <Select
                    {...field}
                    aria-label={t("settings.configs.fields.status")}
                    disabled={catalogEditing}
                    options={["active", "disabled"].map((value) => ({
                      value,
                      label: t(`settings.common.status.${value}`),
                    }))}
                  />
                </Form.Item>
              )}
            />
            <Controller
              control={form.control}
              name="sort_order"
              render={({ field }) => (
                <Form.Item label={t("settings.configs.fields.sort_order")}>
                  <InputNumber
                    {...field}
                    aria-label={t("settings.configs.fields.sort_order")}
                    className="ak-full-width"
                    disabled={catalogEditing}
                  />
                </Form.Item>
              )}
            />
            <Form.Item label={t("settings.configs.fields.flags")}>
              <Space wrap>
                <Controller
                  control={form.control}
                  name="is_secret"
                  render={({ field }) => (
                    <Checkbox
                      checked={field.value}
                      disabled={Boolean(editing)}
                      onChange={(event) => {
                        field.onChange(event.target.checked);
                        if (event.target.checked)
                          form.setValue("is_public", false);
                      }}
                    >
                      {t("settings.configs.fields.is_secret")}
                    </Checkbox>
                  )}
                />
                <Controller
                  control={form.control}
                  name="is_public"
                  render={({ field, fieldState }) => (
                    <Checkbox
                      checked={field.value}
                      disabled={isSecret || catalogEditing}
                      onChange={(event) => {
                        field.onChange(event.target.checked);
                      }}
                      className={fieldState.error ? "ak-field-error" : ""}
                    >
                      {t("settings.configs.fields.is_public")}
                    </Checkbox>
                  )}
                />
              </Space>
            </Form.Item>
          </div>
          {isSecret ? (
            !editing ? (
              <Controller
                control={form.control}
                name="secret_value"
                render={({ field, fieldState }) => (
                  <Form.Item
                    label={t("settings.configs.fields.secret_value")}
                    extra={t("settings.configs.secret.once")}
                    {...(fieldState.error
                      ? {
                          help: fieldState.error.message,
                          validateStatus: "error" as const,
                        }
                      : {})}
                  >
                    <Input.Password
                      {...field}
                      aria-label={t("settings.configs.fields.secret_value")}
                      autoComplete="new-password"
                    />
                  </Form.Item>
                )}
              />
            ) : (
              <Card size="small">
                <Typography.Text>
                  {t("settings.configs.secret.edit_notice")}
                </Typography.Text>
              </Card>
            )
          ) : (
            <>
              <Controller
                control={form.control}
                name="value_text"
                render={({ field, fieldState }) => (
                  <Form.Item
                    label={t("settings.configs.fields.value")}
                    {...(fieldState.error
                      ? {
                          help: fieldState.error.message,
                          validateStatus: "error" as const,
                        }
                      : {})}
                  >
                    {form.watch("value_type") === "boolean" ? (
                      <Select
                        {...field}
                        aria-label={t("settings.configs.fields.value")}
                        allowClear
                        options={[true, false].map((value) => ({
                          value: String(value),
                          label: t(
                            `settings.common.boolean.${value ? "true" : "false"}`,
                          ),
                        }))}
                      />
                    ) : (
                      <Input.TextArea
                        {...field}
                        aria-label={t("settings.configs.fields.value")}
                        autoSize={{ minRows: 2, maxRows: 8 }}
                      />
                    )}
                  </Form.Item>
                )}
              />
              <Controller
                control={form.control}
                name="default_value_text"
                render={({ field }) => (
                  <Form.Item label={t("settings.configs.fields.default_value")}>
                    <Input.TextArea
                      {...field}
                      aria-label={t("settings.configs.fields.default_value")}
                      autoSize={{ minRows: 2, maxRows: 6 }}
                      disabled={catalogEditing}
                    />
                  </Form.Item>
                )}
              />
            </>
          )}
          <Controller
            control={form.control}
            name="validation_schema"
            render={({ field }) => (
              <Form.Item label={t("settings.configs.fields.validation_schema")}>
                <Input.TextArea
                  {...field}
                  aria-label={t("settings.configs.fields.validation_schema")}
                  autoSize={{ minRows: 3, maxRows: 8 }}
                  disabled={catalogEditing}
                />
              </Form.Item>
            )}
          />
          <Controller
            control={form.control}
            name="description"
            render={({ field }) => (
              <Form.Item label={t("settings.configs.fields.description")}>
                <Input.TextArea
                  {...field}
                  aria-label={t("settings.configs.fields.description")}
                  rows={3}
                  disabled={catalogEditing}
                />
              </Form.Item>
            )}
          />
          <div className="ak-drawer-actions">
            <Button onClick={closeEditor}>{t("common.actions.cancel")}</Button>
            <Button
              htmlType="submit"
              loading={mutations.create.isPending || mutations.update.isPending}
              type="primary"
            >
              {t(editing ? "common.actions.save" : "common.actions.create")}
            </Button>
          </div>
        </Form>
      </Drawer>
      <Modal
        destroyOnHidden
        open={Boolean(rotating)}
        title={t("settings.configs.rotate.title")}
        okText={t("settings.configs.actions.rotate")}
        cancelText={t("common.actions.cancel")}
        confirmLoading={mutations.rotate.isPending}
        onCancel={() => {
          setRotating(null);
          secretForm.reset();
        }}
        onOk={() => void rotate()}
      >
        <Typography.Paragraph type="secondary">
          {t("settings.configs.rotate.description", {
            name: rotating ? itemLabel(rotating) : "",
          })}
        </Typography.Paragraph>
        <Form layout="vertical">
          <Controller
            control={secretForm.control}
            name="secret_value"
            render={({ field, fieldState }) => (
              <Form.Item
                label={t("settings.configs.fields.secret_value")}
                extra={t("settings.configs.secret.once")}
                {...(fieldState.error
                  ? {
                      help: fieldState.error.message,
                      validateStatus: "error" as const,
                    }
                  : {})}
              >
                <Input.Password
                  {...field}
                  aria-label={t("settings.configs.fields.secret_value")}
                  autoComplete="new-password"
                />
              </Form.Item>
            )}
          />
        </Form>
      </Modal>
    </div>
  );
}
