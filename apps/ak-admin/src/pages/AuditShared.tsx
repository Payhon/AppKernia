import { Button, Input, Select } from 'antd'
import type { TFunction } from 'i18next'

export interface AuditBaseSearch {
  q: string
  from: string
  to: string
  page: number
  page_size: number
}

export function readBaseSearch(params = new URLSearchParams(location.search)): AuditBaseSearch {
  return { q: params.get('q') ?? '', from: params.get('from') ?? '', to: params.get('to') ?? '', page: Number(params.get('page')) || 1, page_size: Number(params.get('page_size')) || 20 }
}

export function persistSearch(search: Record<string, string | number>) {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(search)) {
    if (value !== '' && !((key === 'page' && value === 1) || (key === 'page_size' && value === 20))) params.set(key, String(value))
  }
  history.replaceState(null, '', `${location.pathname}${params.size ? `?${params}` : ''}`)
}

export function AuditBaseFilters({ search, setSearch, t }: { search: AuditBaseSearch; setSearch: (patch: Partial<AuditBaseSearch>) => void; t: TFunction }) {
  return <>
    <Input.Search allowClear aria-label={t('system.audit.filters.query')} placeholder={t('system.audit.filters.query')} value={search.q} onChange={(event) => { setSearch({ q: event.target.value, page: 1 }) }} />
    <Input aria-label={t('system.audit.filters.from')} type="date" value={search.from} onChange={(event) => { setSearch({ from: event.target.value, page: 1 }) }} />
    <Input aria-label={t('system.audit.filters.to')} type="date" value={search.to} onChange={(event) => { setSearch({ to: event.target.value, page: 1 }) }} />
  </>
}

export function AuditResultTag({ value, t }: { value: 'success' | 'failure' | 'blocked'; t: TFunction }) {
  return <span className={`ak-audit-result ak-audit-result-${value}`}>{t(`system.audit.values.${value}`)}</span>
}

export function AuditRetry({ retry, t }: { retry: () => void; t: TFunction }) {
  return <div className="ak-form-error" role="alert">{t('system.audit.load_error')} <Button onClick={retry}>{t('common.actions.retry')}</Button></div>
}

export function AuditSelect({ label, value, values, onChange, t, keyPrefix = 'system.audit.values' }: { label: string; value: string; values: string[]; onChange: (value: string) => void; t: TFunction; keyPrefix?: string }) {
  return <Select allowClear aria-label={label} placeholder={label} value={value || undefined} onChange={(next) => { onChange(next ?? '') }} options={values.map((item) => ({ value: item, label: t(`${keyPrefix}.${item}`) }))} />
}

export function AuditJSON({ value, empty, label }: { value: Record<string, unknown>; empty: string; label: string }) {
  const hasFields = Object.keys(value).length > 0
  return <section className="ak-audit-json" aria-label={label}>{hasFields ? <pre>{JSON.stringify(value, null, 2)}</pre> : <p>{empty}</p>}</section>
}

export function formatAuditTime(value: string, locale: string) {
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'medium' }).format(new Date(value))
}

export function shortID(value: string | null) {
  return value ? value.slice(0, 8) : '—'
}
