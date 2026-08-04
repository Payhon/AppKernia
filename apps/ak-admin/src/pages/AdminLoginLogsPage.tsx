import { Button, Card, Grid, Select, Table, Typography, type TableColumnsType } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import type { AdminAuditLogin } from '../generated/api/types.gen'
import { useAuditLogins } from '../features/audit/hooks'
import { useLocale } from '../shared/i18n'
import { AuditBaseFilters, AuditResultTag, AuditRetry, formatAuditTime, persistSearch, readBaseSearch, shortID, type AuditBaseSearch } from './AuditShared'

interface SearchState extends AuditBaseSearch { result: '' | 'success' | 'failure' | 'blocked'; audience: string; auth_method: string }
function initialSearch(): SearchState { const params = new URLSearchParams(location.search); const result = params.get('result'); return { ...readBaseSearch(params), result: result === 'success' || result === 'failure' || result === 'blocked' ? result : '', audience: params.get('audience') ?? '', auth_method: params.get('auth_method') ?? '' } }

export function AdminLoginLogsPage() {
  const { t } = useTranslation(); const { locale } = useLocale(); const screens = Grid.useBreakpoint(); const [search, setSearch] = useState(initialSearch); const [copied, setCopied] = useState(false)
  const query = useAuditLogins(search); const patchSearch = (patch: Partial<SearchState>) => { setSearch((current) => ({ ...current, ...patch })) }
  useEffect(() => { persistSearch({ ...search }) }, [search])
  const columns: TableColumnsType<AdminAuditLogin> = [
    { title: t('system.audit.fields.identifier_hint'), key: 'identifier', render: (_, item) => <div><strong>{item.login_identifier_hint || t('system.audit.values.unknown')}</strong><div className="ak-user-secondary">{t('system.audit.fields.user_id')}: {shortID(item.user_id)}</div></div> },
    { title: t('system.audit.fields.result'), dataIndex: 'result', render: (value: AdminAuditLogin['result']) => <AuditResultTag value={value} t={t}/> },
    { title: t('system.audit.fields.auth_method'), dataIndex: 'auth_method', responsive: ['md'] },
    { title: t('system.audit.fields.audience'), dataIndex: 'audience', responsive: ['lg'] },
    { title: t('system.audit.fields.ip'), dataIndex: 'client_ip', responsive: ['lg'], render: (value: string) => value || t('system.audit.values.unknown') },
    { title: t('system.audit.fields.failure_reason'), dataIndex: 'failure_reason', responsive: ['xl'], render: (value: string) => value || '—' },
    { title: t('system.audit.fields.time'), dataIndex: 'occurred_at', render: (value: string) => formatAuditTime(value, locale) },
    { title: t('system.audit.fields.actions'), key: 'actions', width: 150, render: (_, item) => <Button disabled={!item.request_id} type="link" onClick={() => { void navigator.clipboard.writeText(item.request_id).then(() => { setCopied(true) }) }}>{t('system.audit.actions.copy_request_id')}</Button> },
  ]
  return <div className="ak-page-container"><header className="ak-page-heading"><div><Typography.Title level={1}>{t('routes.system.security.login-logs.title')}</Typography.Title><Typography.Paragraph type="secondary">{t('system.audit.logins.description')}</Typography.Paragraph></div></header><Card><div className="ak-audit-filters" role="search"><AuditBaseFilters search={search} setSearch={patchSearch} t={t}/><Select allowClear aria-label={t('system.audit.filters.result')} placeholder={t('system.audit.filters.result')} value={search.result || undefined} onChange={(value) => { patchSearch({ result: value ?? '', page: 1 }) }} options={['success', 'failure', 'blocked'].map((value) => ({ value, label: t(`system.audit.values.${value}`) }))}/><Select allowClear aria-label={t('system.audit.filters.audience')} placeholder={t('system.audit.filters.audience')} value={search.audience || undefined} onChange={(value) => { patchSearch({ audience: value ?? '', page: 1 }) }} options={['ak-admin', 'ak-mobile', 'ak-api'].map((value) => ({ value, label: value }))}/><Select allowClear aria-label={t('system.audit.filters.auth_method')} placeholder={t('system.audit.filters.auth_method')} value={search.auth_method || undefined} onChange={(value) => { patchSearch({ auth_method: value ?? '', page: 1 }) }} options={['password', 'email_otp', 'sms_otp', 'oauth', 'refresh_token', 'api_secret', 'mfa', 'tenant_switch'].map((value) => ({ value, label: value }))}/></div>{copied ? <div className="ak-org-feedback" role="status">{t('system.audit.feedback.request_id_copied')}</div> : null}{query.isError ? <AuditRetry retry={() => { void query.refetch() }} t={t}/> : null}<Table columns={columns} dataSource={query.data?.items ?? []} loading={query.isPending} locale={{ emptyText: t('system.audit.logins.empty') }} pagination={{ current: search.page, pageSize: search.page_size, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (page, page_size) => { patchSearch({ page, page_size }) } }} rowKey="id" scroll={{ x: screens.md ? 1200 : 640 }}/></Card></div>
}
