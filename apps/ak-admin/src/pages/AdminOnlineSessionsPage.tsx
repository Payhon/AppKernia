import { useNavigate } from '@tanstack/react-router'
import { Alert, Button, Card, Grid, Input, Modal, Select, Space, Table, Tag, Typography, type TableColumnsType } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import type { AdminOnlineSession } from '../generated/api/types.gen'
import { useAuthStore } from '../features/auth/store'
import { useOnlineSessions, useRevokeOnlineSession } from '../features/sessions/hooks'
import { ApiError } from '../shared/api/error'
import { useLocale } from '../shared/i18n'
import { formatAuditTime } from './AuditShared'

interface SessionSearch {
  q: string
  audience: string
  platform: string
  status: string
  ip: string
  from: string
  to: string
  page: number
  page_size: number
}

interface SessionFeedback {
  key: string
  error: boolean
}

function readSearch(): SessionSearch {
  const params = new URLSearchParams(location.search)
  const page = Number(params.get('page') ?? 1)
  const pageSize = Number(params.get('page_size') ?? 20)
  return {
    q: params.get('q') ?? '', audience: params.get('audience') ?? '', platform: params.get('platform') ?? '', status: params.get('status') ?? '', ip: params.get('ip') ?? '', from: params.get('from') ?? '', to: params.get('to') ?? '',
    page: Number.isInteger(page) && page > 0 ? page : 1,
    page_size: [10, 20, 50, 100].includes(pageSize) ? pageSize : 20,
  }
}

function persistSearch(search: SessionSearch) {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(search)) if (value !== '' && !(key === 'page' && value === 1) && !(key === 'page_size' && value === 20)) params.set(key, String(value))
  history.replaceState(history.state, '', `${location.pathname}${params.size ? `?${params.toString()}` : ''}`)
}

export function AdminOnlineSessionsPage() {
  const { t } = useTranslation()
  const { locale } = useLocale()
  const navigate = useNavigate()
  const screens = Grid.useBreakpoint()
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []))
  const [search, setSearch] = useState<SessionSearch>(readSearch)
  const [feedback, setFeedback] = useState<SessionFeedback | null>(null)
  const query = useOnlineSessions(search)
  const revoke = useRevokeOnlineSession()
  const patchSearch = (patch: Partial<SessionSearch>) => { setSearch((current) => ({ ...current, ...patch })); }
  useEffect(() => { persistSearch(search) }, [search])

  const confirmRevoke = (item: AdminOnlineSession) => {
    Modal.confirm({
      title: t('system.sessions.revoke.title'),
      content: <Space orientation="vertical"><span>{t('system.sessions.revoke.impact', { user: item.display_name, device: item.device_hint || t('system.sessions.values.unknown') })}</span>{item.current ? <Alert showIcon type="warning" title={t('system.sessions.revoke.current_impact')} /> : null}</Space>,
      okText: t('system.sessions.actions.revoke'), cancelText: t('common.actions.cancel'), okButtonProps: { danger: true },
      onOk: async () => {
        try {
          const result = await revoke.mutateAsync(item.id)
          if (result.current) { await navigate({ to: '/login', replace: true }); return }
          setFeedback({ key: 'system.sessions.revoke.success', error: false })
        } catch (error) {
          setFeedback({ key: error instanceof ApiError ? error.messageKey : 'errors.common.unknown', error: true })
        }
      },
    })
  }

  const columns: TableColumnsType<AdminOnlineSession> = [
    { title: t('system.sessions.fields.user'), key: 'user', render: (_, item) => <div><strong>{item.display_name}</strong> {item.current ? <Tag color="blue">{t('system.sessions.values.current')}</Tag> : null}<div className="ak-user-secondary">{item.user_hint}</div></div> },
    { title: t('system.sessions.fields.application'), key: 'application', render: (_, item) => <div>{t(`system.sessions.values.${item.audience}`)}<div className="ak-user-secondary">{t(`system.sessions.values.${item.platform}`)}</div></div> },
    { title: t('system.sessions.fields.device'), dataIndex: 'device_hint', responsive: ['md'], render: (value: string) => value || t('system.sessions.values.unknown') },
    { title: t('system.sessions.fields.ip'), dataIndex: 'ip_hint', responsive: ['lg'], render: (value: string) => value || '—' },
    { title: t('system.sessions.fields.status'), dataIndex: 'status', render: (value: AdminOnlineSession['status']) => <Tag className={`ak-session-status-${value}`}>{t(`system.sessions.values.${value}`)}</Tag> },
    { title: t('system.sessions.fields.last_seen'), dataIndex: 'last_seen_at', render: (value: string) => formatAuditTime(value, locale) },
    { title: t('system.sessions.fields.expires'), dataIndex: 'expires_at', responsive: ['xl'], render: (value: string) => formatAuditTime(value, locale) },
    { title: t('system.sessions.fields.actions'), key: 'actions', width: 128, ...(screens.md ? { fixed: 'right' as const } : {}), render: (_, item) => item.status === 'active' && permissions.has('iam.session.revoke') ? <Button danger type="link" loading={revoke.isPending && revoke.variables === item.id} onClick={() => { confirmRevoke(item); }}>{t('system.sessions.actions.revoke')}</Button> : '—' },
  ]

  return <div className="ak-page-container"><header className="ak-page-heading"><div><Typography.Title level={1}>{t('routes.system.monitoring.sessions.title')}</Typography.Title><Typography.Paragraph type="secondary">{t('system.sessions.description')}</Typography.Paragraph></div></header><Alert className="ak-session-current-warning" showIcon type="info" title={t('system.sessions.current_warning')} />{feedback ? <div className={feedback.error ? 'ak-form-error' : 'ak-org-feedback'} role={feedback.error ? 'alert' : 'status'}>{t(feedback.key)}</div> : null}<Card><div className="ak-session-filters" role="search"><Input allowClear aria-label={t('system.sessions.filters.search')} placeholder={t('system.sessions.filters.search')} value={search.q} onChange={(event) => { patchSearch({ q: event.target.value, page: 1 }); }}/><Select allowClear aria-label={t('system.sessions.filters.audience')} placeholder={t('system.sessions.filters.audience')} value={search.audience || undefined} onChange={(value) => { patchSearch({ audience: value ?? '', page: 1 }); }} options={['ak-mobile','ak-admin','ak-api'].map((value) => ({ value, label: t(`system.sessions.values.${value}`) }))}/><Select allowClear aria-label={t('system.sessions.filters.platform')} placeholder={t('system.sessions.filters.platform')} value={search.platform || undefined} onChange={(value) => { patchSearch({ platform: value ?? '', page: 1 }); }} options={['android','ios','harmonyos','web','desktop','unknown'].map((value) => ({ value, label: t(`system.sessions.values.${value}`) }))}/><Select allowClear aria-label={t('system.sessions.filters.status')} placeholder={t('system.sessions.filters.status')} value={search.status || undefined} onChange={(value) => { patchSearch({ status: value ?? '', page: 1 }); }} options={['active','revoked','expired'].map((value) => ({ value, label: t(`system.sessions.values.${value}`) }))}/><Input allowClear aria-label={t('system.sessions.filters.ip')} placeholder={t('system.sessions.filters.ip')} value={search.ip} onChange={(event) => { patchSearch({ ip: event.target.value, page: 1 }); }}/><Input type="date" aria-label={t('system.sessions.filters.from')} value={search.from} onChange={(event) => { patchSearch({ from: event.target.value, page: 1 }); }}/><Input type="date" aria-label={t('system.sessions.filters.to')} value={search.to} onChange={(event) => { patchSearch({ to: event.target.value, page: 1 }); }}/></div>{query.isError ? <div className="ak-error-panel" role="alert"><span>{t('errors.common.unknown')}</span><Button onClick={() => void query.refetch()}>{t('common.actions.retry')}</Button></div> : null}<Table columns={columns} dataSource={query.data?.items ?? []} loading={query.isPending} locale={{ emptyText: t('system.sessions.empty') }} pagination={{ current: search.page, pageSize: search.page_size, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (page, page_size) => { patchSearch({ page, page_size }); } }} rowClassName={(item) => item.current ? 'ak-session-current-row' : ''} rowKey="id" scroll={{ x: screens.md ? 1100 : 600 }}/></Card></div>
}
