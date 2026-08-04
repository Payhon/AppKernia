import { Link } from '@tanstack/react-router'
import { Button, Card, Drawer, Form, Grid, Input, Select, Space, Table, Tag, Typography, type TableColumnsType } from 'antd'
import { useEffect, useMemo, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import type { AdminTenant, AdminTenantCreateRequest, AdminTenantStatus } from '../generated/api/types.gen'
import { useAuthStore } from '../features/auth/store'
import { useAdminTenantMutations, useAdminTenants } from '../features/tenants/hooks'
import { ApiError } from '../shared/api/error'

const createSchema = z.object({ code: z.string().regex(/^[a-z0-9][a-z0-9-]{1,63}$/), name: z.string().trim().min(1).max(120) })
type CreateValues = z.infer<typeof createSchema>
interface SearchState { q: string; status: AdminTenantStatus | ''; page: number; page_size: number; sort: 'created_desc' }

function readSearch(): SearchState {
  const params = new URLSearchParams(window.location.search)
  return { q: params.get('q') ?? '', status: params.get('status') === 'disabled' ? 'disabled' : params.get('status') === 'active' ? 'active' : '', page: Number(params.get('page') ?? 1) || 1, page_size: Number(params.get('page_size') ?? 20) || 20, sort: 'created_desc' }
}

export function AdminTenantsPage() {
  const { t, i18n } = useTranslation()
  const screens = Grid.useBreakpoint()
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []))
  const refreshContext = useAuthStore((state) => state.refreshContext)
  const initial = useMemo(readSearch, [])
  const [search, setSearch] = useState(initial)
  const [open, setOpen] = useState(false)
  const [feedback, setFeedback] = useState<string | null>(null)
  const tenants = useAdminTenants(search)
  const mutations = useAdminTenantMutations()
  const form = useForm<CreateValues>({ defaultValues: { code: '', name: '' } })

  useEffect(() => {
    const params = new URLSearchParams()
    if (search.q) params.set('q', search.q)
    if (search.status) params.set('status', search.status)
    if (search.page !== 1) params.set('page', String(search.page))
    if (search.page_size !== 20) params.set('page_size', String(search.page_size))
    window.history.replaceState(null, '', `${window.location.pathname}${params.size ? `?${params}` : ''}`)
  }, [search])

  const errorText = (error: unknown) => t(error instanceof ApiError ? error.messageKey : 'errors.common.unknown')
  const submit = form.handleSubmit(async (values) => {
    const parsed = createSchema.safeParse(values)
    if (!parsed.success) {
      for (const issue of parsed.error.issues) form.setError(issue.path[0] as keyof CreateValues, { message: t('validation.invalid', { name: t(`system.tenants.fields.${String(issue.path[0])}`) }) })
      return
    }
    try {
      const created = await mutations.create.mutateAsync(parsed.data satisfies AdminTenantCreateRequest)
      await refreshContext()
      setOpen(false)
      form.reset()
      setFeedback(t('system.tenants.feedback.created', { name: created.name }))
    } catch (error) { setFeedback(errorText(error)) }
  })

  const columns: TableColumnsType<AdminTenant> = [
    { title: t('system.tenants.fields.name'), key: 'name', render: (_, tenant) => <div><Link to="/system/users/tenants/$tenantId" params={{ tenantId: tenant.id }}><strong>{tenant.name}</strong></Link><div className="ak-user-secondary">{tenant.code}</div></div> },
    { title: t('system.tenants.fields.status'), dataIndex: 'status', render: (status: AdminTenantStatus) => <Tag className={status === 'active' ? 'ak-status-success' : 'ak-status-neutral'}>{t(`system.tenants.status.${status}`)}</Tag> },
    { title: t('system.tenants.fields.members'), dataIndex: 'member_count', responsive: ['md'] },
    { title: t('system.tenants.fields.plan'), dataIndex: 'plan_code', responsive: ['lg'], render: (value: string) => value || t('common.states.empty') },
    { title: t('system.tenants.fields.created_at'), dataIndex: 'created_at', responsive: ['lg'], render: (value: string) => new Intl.DateTimeFormat(i18n.language, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) },
    { title: t('common.actions.edit'), key: 'action', width: screens.md ? 100 : 76, render: (_, tenant) => <Link to="/system/users/tenants/$tenantId" params={{ tenantId: tenant.id }}>{t('common.actions.edit')}</Link> },
  ]

  return <div className="ak-page-container">
    <header className="ak-page-heading ak-users-heading"><div><Typography.Title level={1}>{t('routes.system.users.tenants.title')}</Typography.Title><Typography.Paragraph type="secondary">{t('system.tenants.description')}</Typography.Paragraph></div>{permissions.has('iam.tenant.create') ? <Button onClick={() => { setOpen(true); setFeedback(null) }} type="primary">{t('system.tenants.actions.create')}</Button> : null}</header>
    <Card>
      <div className="ak-users-filters" role="search"><Input.Search allowClear aria-label={t('system.tenants.filters.query')} onChange={(event) => { setSearch((current) => ({ ...current, q: event.target.value, page: 1 })) }} placeholder={t('system.tenants.filters.query')} value={search.q} /><Select allowClear aria-label={t('system.tenants.filters.status')} onChange={(value) => { setSearch((current) => ({ ...current, status: value ?? '', page: 1 })) }} options={(['active', 'disabled'] as const).map((value) => ({ label: t(`system.tenants.status.${value}`), value }))} placeholder={t('system.tenants.filters.status')} value={search.status || undefined} /></div>
      {feedback ? <div className="ak-org-feedback" role="status">{feedback}</div> : null}
      {tenants.isError ? <div className="ak-form-error" role="alert">{t('system.tenants.load_error')} <Button onClick={() => { void tenants.refetch() }}>{t('common.actions.retry')}</Button></div> : null}
      <Table columns={columns} dataSource={tenants.data?.items ?? []} loading={tenants.isPending} locale={{ emptyText: t('system.tenants.empty') }} pagination={{ current: search.page, pageSize: search.page_size, total: tenants.data?.total ?? 0, showSizeChanger: true, onChange: (page, pageSize) => { setSearch((current) => ({ ...current, page, page_size: pageSize })) } }} rowKey="id" scroll={{ x: screens.md ? 820 : 360 }} />
    </Card>
    <Drawer destroyOnHidden onClose={() => { setOpen(false) }} open={open} size="large" title={t('system.tenants.editor.create_title')}><Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}><Controller control={form.control} name="code" render={({ field, fieldState }) => <Form.Item label={t('system.tenants.fields.code')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input {...field} aria-label={t('system.tenants.fields.code')} autoComplete="off" /></Form.Item>} /><Controller control={form.control} name="name" render={({ field, fieldState }) => <Form.Item label={t('system.tenants.fields.name')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input {...field} aria-label={t('system.tenants.fields.name')} /></Form.Item>} /><Space><Button onClick={() => { setOpen(false) }}>{t('common.actions.cancel')}</Button><Button htmlType="submit" loading={mutations.create.isPending} type="primary">{t('common.actions.create')}</Button></Space></Form></Drawer>
  </div>
}
