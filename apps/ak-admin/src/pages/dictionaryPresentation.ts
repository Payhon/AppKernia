import type {
  AdminDictionaryItem,
  AdminDictionaryType,
} from "../generated/api/types.gen";
export interface DictionaryTypeCategory {
  code: string;
  label: string;
  types: AdminDictionaryType[];
}

export function dictionaryCategoryCode(code: string): string {
  const separator = code.indexOf(".");
  return separator > 0 ? code.slice(0, separator) : "custom";
}

export function groupDictionaryTypes(
  types: AdminDictionaryType[],
  labelFor: (code: string) => string,
): DictionaryTypeCategory[] {
  const groups = new Map<string, AdminDictionaryType[]>();
  for (const type of types) {
    const code = dictionaryCategoryCode(type.code);
    const current = groups.get(code);
    if (current) current.push(type);
    else groups.set(code, [type]);
  }
  return Array.from(groups, ([code, groupedTypes]) => ({
    code,
    label: labelFor(code),
    types: groupedTypes,
  })).sort((left, right) => left.label.localeCompare(right.label));
}

export function findTenantOverride(
  source: AdminDictionaryItem,
  items: AdminDictionaryItem[],
): AdminDictionaryItem | undefined {
  if (!source.is_locked) return source;
  return items.find(
    (candidate) =>
      candidate.tenant_id !== null &&
      candidate.item_value === source.item_value &&
      candidate.locale === source.locale,
  );
}
