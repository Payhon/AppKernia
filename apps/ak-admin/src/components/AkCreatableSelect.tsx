import { Select, type SelectProps } from "antd";
import type { ReactNode } from "react";

const DEFAULT_VALUE = "__appkernia_default_value__";

export interface AkCreatableSelectOption {
  label: ReactNode;
  searchText: string;
  selectedLabel: ReactNode;
  value: string;
}

interface AkCreatableSelectProps {
  "aria-label": string;
  defaultLabel: ReactNode;
  disabled?: boolean;
  onBlur?: () => void;
  onChange: (value: string) => void;
  options: AkCreatableSelectOption[];
  value: string;
}

export function AkCreatableSelect({
  "aria-label": ariaLabel,
  defaultLabel,
  disabled = false,
  onBlur,
  onChange,
  options,
  value,
}: AkCreatableSelectProps) {
  const selectOptions: SelectProps["options"] = [
    {
      label: defaultLabel,
      searchText: typeof defaultLabel === "string" ? defaultLabel : "",
      selectedLabel: defaultLabel,
      value: DEFAULT_VALUE,
    },
    ...options,
  ];
  const selected = value ? [value] : [];

  return (
    <Select
      aria-label={ariaLabel}
      className="ak-creatable-select"
      disabled={disabled}
      getPopupContainer={(trigger: HTMLElement) =>
        trigger.parentElement ?? document.body
      }
      mode="tags"
      {...(onBlur ? { onBlur } : {})}
      optionLabelProp="selectedLabel"
      options={selectOptions}
      {...(value ? {} : { placeholder: defaultLabel })}
      showSearch={{ optionFilterProp: "searchText" }}
      value={selected}
      onChange={(next: string[]) => {
        const nextValue = next.at(-1) ?? DEFAULT_VALUE;
        onChange(nextValue === DEFAULT_VALUE ? "" : nextValue);
      }}
    />
  );
}
