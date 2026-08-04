import { Link } from '@tanstack/react-router'
import { Card, Grid, Input, Select, Table, Tag, Typography, type TableColumnsType } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import type { AdminAuditSecurityEvent } from '../generated/api/types.gen'
import { useAuditSecurityEvents } from '../features/audit/hooks'
import { useLocale } from '../shared/i18n'
import { AuditBaseFilters, AuditRetry, formatAuditTime, persistSearch, readBaseSearch, type AuditBaseSearch } from './AuditShared'

interface SearchState extends AuditBaseSearch { severity: string; source: string; status: '' | 'open' | 'resolved' }
function initialSearch(): SearchState { const params = new URLSearchParams(location.search); const status = params.get('status'); return { ...readBaseSearch(params), severity: params.get('severity') ?? '', source: params.get('source') ?? '', status: status === 'open' || status === 'resolved' ? status : '' } }

export function AdminSecurityEventsPage() {
  const { t } = useTranslation(); const { locale } = useLocale(); const screens = Grid.useBreakpoint(); const [search, setSearch] = useState(initialSearch)
  const query = useAuditSecurityEvents(search); const patchSearch = (patch: Partial<SearchState>) => { setSearch((current) => ({ ...current, ...patch })) }
  useEffect(() => { persistSearch({ ...search }) }, [search])
  const columns: TableColumnsType<AdminAuditSecurityEvent> = [
    { title: t('system.audit.fields.event_type'), key: 'event', render: (_, item) => <div><Link to="/system/security/events/$eventId" params={{ eventId: item.id }}><strong>{item.event_type}</strong></Link><div className="ak-user-secondary">{item.source}</div></div> },
    { title: t('system.audit.fields.severity'), dataIndex: 'severity', render: (value: AdminAuditSecurityEvent['severity']) => <Tag className={`ak-audit-severity-${value}`}>{t(`system.audit.values.${value}`)}</Tag> },
    { title: t('system.audit.fields.status'), dataIndex: 'resolved_at', render: (value: string | null) => <Tag className={value ? 'ak-status-success' : 'ak-status-warning'}>{t(`system.audit.values.${value ? 'resolved' : 'open'}`)}</Tag> },
    { title: t('system.audit.fields.ip'), dataIndex: 'client_ip', responsive: ['lg'], render: (value: string) => value || t('system.audit.values.unknown') },
    { title: t('system.audit.fields.user_id'), dataIndex: 'user_id', responsive: ['xl'], render: (value: string | null) => value?.slice(0, 8) ?? '—' },
    { title: t('system.audit.fields.time'), dataIndex: 'occurred_at', render: (value: string) => formatAuditTime(value, locale) },
    { title: t('system.audit.fields.actions'), key: 'actions', width: 120, render: (_, item) => <Link to="/system/security/events/$eventId" params={{ eventId: item.id }}>{t('system.audit.actions.view')}</Link> },
  ]
  return <div className="ak-page-container"><header className="ak-page-heading"><div><Typography.Title level={1}>{t('routes.system.security.events.title')}</Typography.Title><Typography.Paragraph type="secondary">{t('system.audit.events.description')}</Typography.Paragraph></div></header><Card><div className="ak-audit-filters" role="search"><AuditBaseFilters search={search} setSearch={patchSearch} t={t}/><Select allowClear aria-label={t('system.audit.filters.severity')} placeholder={t('system.audit.filters.severity')} value={search.severity || undefined} onChange={(value) => { patchSearch({ severity: value ?? '', page: 1 }) }} options={['info', 'low', 'medium', 'high', 'critical'].map((value) => ({ value, label: t(`system.audit.values.${value}`) }))}/><Input allowClear aria-label={t('system.audit.filters.source')} placeholder={t('system.audit.filters.source')} value={search.source} onChange={(event) => { patchSearch({ source: event.target.value, page: 1 }) }}/><Select allowClear aria-label={t('system.audit.filters.status')} placeholder={t('system.audit.filters.status')} value={search.status || undefined} onChange={(value) => { patchSearch({ status: value ?? '', page: 1 }) }} options={['open', 'resolved'].map((value) => ({ value, label: t(`system.audit.values.${value}`) }))}/></div>{query.isError ? <AuditRetry retry={() => { void query.refetch() }} t={t}/> : null}<Table columns={columns} dataSource={query.data?.items ?? []} loading={query.isPending} locale={{ emptyText: t('system.audit.events.empty') }} pagination={{ current: search.page, pageSize: search.page_size, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (page, page_size) => { patchSearch({ page, page_size }) } }} rowKey="id" scroll={{ x: screens.md ? 1050 : 540 }}/></Card></div>
}
