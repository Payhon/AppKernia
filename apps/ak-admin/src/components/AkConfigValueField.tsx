import { Button, Form, Input, InputNumber, Select, Space, Switch, Tag } from "antd";
import type { TFunction } from "i18next";
import { useState } from "react";

import type { AdminConfigItem } from "../generated/api/types.gen";
import { AkFilePicker } from "./AkFilePicker";

export type AkConfigDraftValue = boolean | number | string | null;

interface AkConfigValueFieldProps {
  disabled: boolean;
  error?: string | undefined;
  item: AdminConfigItem;
  label: string;
  onChange: (value: AkConfigDraftValue) => void;
  t: TFunction;
  value: AkConfigDraftValue;
}

function schema(item: AdminConfigItem) {
  return item.validation_schema as Record<string, unknown>;
}

export function configDraftValue(item: AdminConfigItem): AkConfigDraftValue {
  if (item.is_secret) return "";
  const value = item.value ?? item.default_value;
  if (item.value_type === "boolean") return value === true;
  if (item.value_type === "integer" || item.value_type === "decimal")
    return typeof value === "number" ? value : null;
  if (item.value_type === "json")
    return value === undefined || value === null
      ? ""
      : JSON.stringify(value, null, 2);
  return typeof value === "string" ? value : "";
}

export function parseConfigDraftValue(
  item: AdminConfigItem,
  value: AkConfigDraftValue,
): unknown {
  if (item.is_secret) return typeof value === "string" ? value.trim() : "";
  if (item.value_type === "boolean") return value === true;
  if (item.value_type === "integer" || item.value_type === "decimal")
    return value;
  if (item.value_type === "json")
    return typeof value === "string" && value.trim()
      ? (JSON.parse(value) as unknown)
      : null;
  return typeof value === "string" ? value : "";
}

export function validateConfigDraftValue(
  item: AdminConfigItem,
  value: AkConfigDraftValue,
  t: TFunction,
): string | undefined {
  if (item.is_secret && value === "") return undefined;
  if (item.is_secret && typeof value === "string" && !value.trim())
    return t("settings.configs.form.validation.required");
  const rules = schema(item);
  const minimum = rules["minimum"];
  const maximum = rules["maximum"];
  const minLength = rules["minLength"];
  const maxLength = rules["maxLength"];
  const allowedValues = rules["enum"];
  if (item.value_type === "json") {
    try {
      parseConfigDraftValue(item, value);
    } catch {
      return t("settings.configs.form.validation.json");
    }
  }
  if (item.value_type === "integer" && value !== null && !Number.isInteger(value))
    return t("settings.configs.form.validation.integer");
  if (
    (item.value_type === "integer" || item.value_type === "decimal") &&
    value === null
  )
    return t("settings.configs.form.validation.required");
  if (typeof value === "number") {
    if (typeof minimum === "number" && value < minimum)
      return t("settings.configs.form.validation.minimum", {
        value: minimum,
      });
    if (typeof maximum === "number" && value > maximum)
      return t("settings.configs.form.validation.maximum", {
        value: maximum,
      });
  }
  if (typeof value === "string") {
    if (typeof minLength === "number" && value.length < minLength)
      return t("settings.configs.form.validation.min_length", {
        value: minLength,
      });
    if (typeof maxLength === "number" && value.length > maxLength)
      return t("settings.configs.form.validation.max_length", {
        value: maxLength,
      });
  }
  if (Array.isArray(allowedValues) && !allowedValues.includes(value))
    return t("settings.configs.form.validation.enum");
  return undefined;
}

export function AkConfigValueField({
  disabled,
  error,
  item,
  label,
  onChange,
  t,
  value,
}: AkConfigValueFieldProps) {
  const [filePickerOpen, setFilePickerOpen] = useState(false);
  const rules = schema(item);
  const maximum = rules["maximum"];
  const minimum = rules["minimum"];
  const maxLength = rules["maxLength"];
  const format = rules["format"];
  const allowedValues = rules["enum"];
  const fieldId = `config-value-${item.id}`;
  const enumOptions = Array.isArray(allowedValues)
    ? allowedValues.map((option) => ({
        label:
          typeof option === "boolean"
            ? t(`settings.common.boolean.${option ? "true" : "false"}`)
            : String(option),
        value: option as boolean | number | string,
      }))
    : null;
  const extra = (
    <span className="ak-config-field-extra">
      <code>{item.config_key}</code>
      {item.is_locked ? (
        <Tag>{t("settings.common.system_locked")}</Tag>
      ) : item.status === "disabled" ? (
        <Tag>{t("settings.common.status.disabled")}</Tag>
      ) : null}
      {item.is_secret ? (
        <Tag className={item.secret_configured ? "ak-status-success" : "ak-status-warning"}>
          {t(
            item.secret_configured
              ? "settings.configs.secret.configured"
              : "settings.configs.secret.missing",
          )}
        </Tag>
      ) : null}
    </span>
  );
  let control;
  if (item.is_secret) {
    control = (
      <Input.Password
        autoComplete="new-password"
        disabled={disabled}
        id={fieldId}
        placeholder={t("settings.configs.form.secret_placeholder")}
        value={typeof value === "string" ? value : ""}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      />
    );
  } else if (enumOptions) {
    control = (
      <Select
        aria-labelledby={`${fieldId}-label`}
        disabled={disabled}
        id={fieldId}
        options={enumOptions}
        value={value}
        onChange={onChange}
      />
    );
  } else if (item.value_type === "boolean") {
    control = (
      <Switch
        checked={value === true}
        checkedChildren={t("settings.common.boolean.true")}
        disabled={disabled}
        id={fieldId}
        unCheckedChildren={t("settings.common.boolean.false")}
        onChange={onChange}
      />
    );
  } else if (item.value_type === "integer" || item.value_type === "decimal") {
    control = (
      <InputNumber
        className="ak-full-width"
        disabled={disabled}
        id={fieldId}
        {...(typeof maximum === "number" ? { max: maximum } : {})}
        {...(typeof minimum === "number" ? { min: minimum } : {})}
        {...(item.value_type === "integer" ? { precision: 0 } : {})}
        value={typeof value === "number" ? value : null}
        onChange={(next) => {
          onChange(next);
        }}
      />
    );
  } else if (item.value_type === "json") {
    control = (
      <Input.TextArea
        disabled={disabled}
        id={fieldId}
        rows={5}
        spellCheck={false}
        value={typeof value === "string" ? value : ""}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      />
    );
  } else if (item.value_type === "datetime") {
    control = (
      <Input
        disabled={disabled}
        id={fieldId}
        type="datetime-local"
        value={typeof value === "string" ? value : ""}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      />
    );
  } else if (format === "uuid-or-empty") {
    control = (
      <>
        <Space.Compact block>
          <Input
            disabled={disabled}
            id={fieldId}
            value={typeof value === "string" ? value : ""}
            onChange={(event) => {
              onChange(event.target.value);
            }}
          />
          <Button
            disabled={disabled}
            onClick={() => {
              setFilePickerOpen(true);
            }}
          >
            {t("system.files.actions.select")}
          </Button>
        </Space.Compact>
        {filePickerOpen ? (
          <AkFilePicker
            open
            onClose={() => {
              setFilePickerOpen(false);
            }}
            onSelect={(file) => {
              onChange(file.id);
              setFilePickerOpen(false);
            }}
          />
        ) : null}
      </>
    );
  } else if (
    format !== "email" &&
    format !== "uri" &&
    typeof maxLength === "number" &&
    maxLength > 200
  ) {
    control = (
      <Input.TextArea
        disabled={disabled}
        id={fieldId}
        maxLength={maxLength}
        rows={3}
        value={typeof value === "string" ? value : ""}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      />
    );
  } else {
    control = (
      <Input
        disabled={disabled}
        id={fieldId}
        {...(typeof maxLength === "number" ? { maxLength } : {})}
        type={format === "email" ? "email" : format === "uri" ? "url" : "text"}
        value={typeof value === "string" ? value : ""}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      />
    );
  }
  return (
    <Form.Item
      className="ak-config-direct-field"
      extra={extra}
      htmlFor={fieldId}
      label={<span id={`${fieldId}-label`}>{label}</span>}
      {...(error ? { help: error, validateStatus: "error" as const } : {})}
    >
      {control}
    </Form.Item>
  );
}
