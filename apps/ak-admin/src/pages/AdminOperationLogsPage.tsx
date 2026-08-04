import { Alert, Button, Card, Descriptions, Drawer, Grid, Select, Table, Tag, Typography, type TableColumnsType } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import type { AdminAuditOperation } from '../generated/api/types.gen'
import { useAuditOperations } from '../features/audit/hooks'
import { useLocale } from '../shared/i18n'
import {
  AuditBaseFilters,
  AuditJSON,
  AuditResultTag,
  AuditRetry,
  formatAuditTime,
  persistSearch,
  readBaseSearch,
  shortID,
  type AuditBaseSearch,
} from './AuditShared'

interface SearchState extends AuditBaseSearch {
  module_code: string
  result: '' | 'success' | 'failure'
}

function initialSearch(): SearchState {
  const params = new URLSearchParams(location.search)
  const result = params.get('result')
  return {
    ...readBaseSearch(params),
    module_code: params.get('module_code') ?? '',
    result: result === 'success' || result === 'failure' ? result : '',
  }
}

export function AdminOperationLogsPage() {
  const { t } = useTranslation()
  const { locale } = useLocale()
  const screens = Grid.useBreakpoint()
  const [search, setSearch] = useState(initialSearch)
  const [selected, setSelected] = useState<AdminAuditOperation | null>(null)
  const query = useAuditOperations(search)
  const patchSearch = (patch: Partial<SearchState>) => {
    setSearch((current) => ({ ...current, ...patch }))
  }
  useEffect(() => {
    persistSearch({ ...search })
  }, [search])
  const modules = [...new Set((query.data?.items ?? []).map((item) => item.module_code))].sort()
  const columns: TableColumnsType<AdminAuditOperation> = [
    {
      title: t('system.audit.fields.action'),
      key: 'action',
      render: (_, item) => <div><strong>{item.action_name}</strong><div className="ak-user-secondary">{item.module_code}</div></div>,
    },
    {
      title: t('system.audit.fields.resource'),
      key: 'resource',
      responsive: ['md'],
      render: (_, item) => <span>{item.resource_type || '—'} {item.resource_id ? `· ${shortID(item.resource_id)}` : ''}</span>,
    },
    {
      title: t('system.audit.fields.request_id'),
      dataIndex: 'request_id',
      responsive: ['lg'],
      render: (value: string) => <code className="ak-audit-code">{value}</code>,
    },
    {
      title: t('system.audit.fields.result'),
      dataIndex: 'succeeded',
      render: (value: boolean) => <AuditResultTag value={value ? 'success' : 'failure'} t={t} />,
    },
    {
      title: t('system.audit.fields.time'),
      dataIndex: 'occurred_at',
      render: (value: string) => formatAuditTime(value, locale),
    },
    {
      title: t('system.audit.fields.actions'),
      key: 'actions',
      width: 120,
      render: (_, item) => <Button type="link" onClick={() => { setSelected(item) }}>{t('system.audit.actions.view')}</Button>,
    },
  ]

  return <div className="ak-page-container">
    <header className="ak-page-heading">
      <div>
        <Typography.Title level={1}>{t('routes.system.security.operation-logs.title')}</Typography.Title>
        <Typography.Paragraph type="secondary">{t('system.audit.operations.description')}</Typography.Paragraph>
      </div>
    </header>
    <Card>
      <div className="ak-audit-filters" role="search">
        <AuditBaseFilters search={search} setSearch={patchSearch} t={t}/>
        <Select
          allowClear
          aria-label={t('system.audit.filters.module')}
          placeholder={t('system.audit.filters.module')}
          value={search.module_code || undefined}
          onChange={(value) => { patchSearch({ module_code: value ?? '', page: 1 }) }}
          options={modules.map((value) => ({ value, label: value }))}
        />
        <Select
          allowClear
          aria-label={t('system.audit.filters.result')}
          placeholder={t('system.audit.filters.result')}
          value={search.result || undefined}
          onChange={(value) => { patchSearch({ result: value ?? '', page: 1 }) }}
          options={['success', 'failure'].map((value) => ({ value, label: t(`system.audit.values.${value}`) }))}
        />
      </div>
      {query.isError ? <AuditRetry retry={() => { void query.refetch() }} t={t}/> : null}
      <Table
        columns={columns}
        dataSource={query.data?.items ?? []}
        loading={query.isPending}
        locale={{ emptyText: t('system.audit.operations.empty') }}
        pagination={{
          current: search.page,
          pageSize: search.page_size,
          total: query.data?.total ?? 0,
          showSizeChanger: true,
          onChange: (page, page_size) => { patchSearch({ page, page_size }) },
        }}
        rowKey="id"
        scroll={{ x: screens.md ? 1050 : 560 }}
      />
    </Card>
    <Drawer
      destroyOnHidden
      open={Boolean(selected)}
      onClose={() => { setSelected(null) }}
      size="large"
      title={t('system.audit.operation_detail.title')}
    >
      {selected ? <div className="ak-audit-detail">
        <Alert showIcon type="info" title={t('system.audit.operation_detail.redaction_notice')}/>
        <Descriptions
          bordered
          column={1}
          size="small"
          items={[
            { key: 'action', label: t('system.audit.fields.action'), children: selected.action_name },
            { key: 'request', label: t('system.audit.fields.request_id'), children: selected.request_id },
            { key: 'user', label: t('system.audit.fields.user_id'), children: selected.user_id ?? t('system.audit.values.unknown') },
            { key: 'ip', label: t('system.audit.fields.ip'), children: selected.client_ip || t('system.audit.values.unknown') },
            { key: 'result', label: t('system.audit.fields.result'), children: <Tag>{t(`system.audit.values.${selected.succeeded ? 'success' : 'failure'}`)}</Tag> },
          ]}
        />
        <Typography.Title level={2}>{t('system.audit.operation_detail.request_summary')}</Typography.Title>
        <AuditJSON
          value={selected.request_summary}
          empty={t('system.audit.operation_detail.empty_json')}
          label={t('system.audit.operation_detail.request_summary')}
        />
        <div className="ak-audit-diff">
          <div>
            <Typography.Title level={2}>{t('system.audit.operation_detail.before')}</Typography.Title>
            <AuditJSON value={selected.before_data} empty={t('system.audit.operation_detail.empty_json')} label={t('system.audit.operation_detail.before')}/>
          </div>
          <div>
            <Typography.Title level={2}>{t('system.audit.operation_detail.after')}</Typography.Title>
            <AuditJSON value={selected.after_data} empty={t('system.audit.operation_detail.empty_json')} label={t('system.audit.operation_detail.after')}/>
          </div>
        </div>
      </div> : null}
    </Drawer>
  </div>
}
