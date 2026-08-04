import { Button, Card, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography } from 'antd'
import type { TFunction } from 'i18next'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import type { AdminOrgPosition, AdminOrgPositionRequest, AdminOrgUnit } from '../generated/api/types.gen'
import { useAuthStore } from '../features/auth/store'
import { useOrgPositionMutations, useOrgPositions, useOrgUnits } from '../features/org/hooks'
import { ApiError } from '../shared/api/error'

const positionSchema = z.object({ code: z.string().trim().min(2).max(64), name: z.string().trim().min(1).max(120), description: z.string().trim().max(500), sort_order: z.number().int(), status: z.enum(['active', 'disabled']) })
type PositionValues = z.infer<typeof positionSchema>
function flatUnits(units: AdminOrgUnit[]): AdminOrgUnit[] { return units.flatMap((unit) => [unit, ...flatUnits(unit.children)]) }
function readFilters() { const params = new URLSearchParams(window.location.search); return { q: params.get('q') ?? '', status: params.get('status') ?? '', unit: params.get('unit_id') ?? '' } }

export function OrgPositionsPage() {
  const { t } = useTranslation()
  const permissions = useMemo(() => new Set(useAuthStore.getState().context?.permissions ?? []), [])
  const initial = useMemo(readFilters, [])
  const [query, setQuery] = useState(initial.q), [status, setStatus] = useState(initial.status), [unitID, setUnitID] = useState(initial.unit)
  const [editing, setEditing] = useState<AdminOrgPosition | null | 'create'>(null)
  const [feedback, setFeedback] = useState<string | null>(null)
  const positions = useOrgPositions(query, status, unitID)
  const units = useOrgUnits('', '')
  const mutations = useOrgPositionMutations()
  const unitOptions = useMemo(() => flatUnits(units.data ?? []).map((unit) => ({ label: unit.name, value: unit.id })), [units.data])
  const { control, handleSubmit, reset, setError, formState } = useForm<PositionValues>({ defaultValues: { code: '', name: '', description: '', sort_order: 0, status: 'active' } })
  useEffect(() => { const params = new URLSearchParams(); if (query) params.set('q', query); if (status) params.set('status', status); if (unitID) params.set('unit_id', unitID); window.history.replaceState(null, '', `${window.location.pathname}${params.size ? `?${params}` : ''}`) }, [query, status, unitID])

  const open = (value: AdminOrgPosition | 'create') => { setFeedback(null); setEditing(value); reset(value === 'create' ? { code: '', name: '', description: '', sort_order: 0, status: 'active' } : { code: value.code, name: value.name, description: value.description, sort_order: value.sort_order, status: value.status }) }
  const submit = handleSubmit(async (values) => { const parsed = positionSchema.safeParse(values); if (!parsed.success) { for (const issue of parsed.error.issues) { const field = issue.path[0] as keyof PositionValues; setError(field, { message: t('validation.invalid', { name: t(`system.org.positions.fields.${field}`) }) }) }; return }; try { const input = parsed.data satisfies AdminOrgPositionRequest; if (editing === 'create') await mutations.create.mutateAsync(input); else if (editing) await mutations.update.mutateAsync({ id: editing.id, input }); setEditing(null); setFeedback(t('system.org.feedback.saved')) } catch (error) { setFeedback(t(error instanceof ApiError ? error.messageKey : 'errors.common.unknown')) } })
  const remove = (item: AdminOrgPosition) => { Modal.confirm({ title: t('system.org.positions.delete.title'), content: t('system.org.positions.delete.impact', { members: item.member_count }), okText: t('common.actions.delete'), cancelText: t('common.actions.cancel'), okButtonProps: { danger: true }, onOk: async () => { try { await mutations.remove.mutateAsync(item.id); setFeedback(t('system.org.feedback.deleted')) } catch (error) { setFeedback(t(error instanceof ApiError ? error.messageKey : 'errors.common.unknown')) } } }) }

  if (positions.isPending) return <div className="ak-centered-state" aria-live="polite"><span className="ak-loading-indicator" /><span>{t('common.states.loading')}</span></div>
  return <div className="ak-page-container"><header className="ak-page-heading ak-org-heading"><div><Typography.Title level={1}>{t('routes.system.users.positions.title')}</Typography.Title><Typography.Paragraph type="secondary">{t('system.org.positions.description')}</Typography.Paragraph></div>{permissions.has('org.position.create') ? <Button onClick={() => { open('create') }} type="primary">{t('system.org.positions.actions.create')}</Button> : null}</header>
    <div className="ak-org-filters" role="search"><Input allowClear aria-label={t('system.org.filters.query')} onChange={(event) => { setQuery(event.target.value) }} placeholder={t('system.org.filters.query')} value={query} /><Select allowClear aria-label={t('system.org.positions.filters.department')} loading={units.isPending} onChange={(value) => { setUnitID(value ?? '') }} options={unitOptions} placeholder={t('system.org.positions.filters.department')} showSearch value={unitID || undefined} /><Select allowClear aria-label={t('system.org.filters.status')} onChange={(value) => { setStatus(value ?? '') }} options={[{ label: t('common.states.enabled'), value: 'active' }, { label: t('common.states.disabled'), value: 'disabled' }]} placeholder={t('system.org.filters.status')} value={status || undefined} /></div>
    {feedback ? <div className="ak-org-feedback" role="status">{feedback}</div> : null}{positions.isError ? <div className="ak-form-error" role="alert">{t('system.org.load_error')} <Button onClick={() => { void positions.refetch() }}>{t('common.actions.retry')}</Button></div> : null}
    <Card><Table<AdminOrgPosition> columns={[{ title: t('system.org.positions.fields.name'), dataIndex: 'name' }, { title: t('system.org.positions.fields.code'), dataIndex: 'code' }, { title: t('system.org.positions.fields.status'), dataIndex: 'status', render: (value: AdminOrgPosition['status']) => <Tag className={value === 'active' ? 'ak-status-success' : 'ak-status-neutral'}>{t(value === 'active' ? 'common.states.enabled' : 'common.states.disabled')}</Tag> }, { title: t('system.org.positions.fields.member_count'), dataIndex: 'member_count' }, { title: t('system.org.positions.actions.label'), key: 'actions', render: (_, item) => <Space wrap>{permissions.has('org.position.update') ? <Button onClick={() => { open(item) }} size="small">{t('common.actions.edit')}</Button> : null}{permissions.has('org.position.delete') ? <Button danger onClick={() => { remove(item) }} size="small">{t('common.actions.delete')}</Button> : null}</Space> }]} dataSource={positions.data?.items ?? []} locale={{ emptyText: t('system.org.positions.empty') }} pagination={{ pageSize: 20, showSizeChanger: false }} rowKey="id" scroll={{ x: 720 }} /></Card>
    <Modal destroyOnHidden footer={null} onCancel={() => { setEditing(null) }} open={editing !== null} title={t(editing === 'create' ? 'system.org.positions.editor.create_title' : 'system.org.positions.editor.edit_title')}><Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}><PositionForm control={control} t={t} /><Button htmlType="submit" loading={formState.isSubmitting} type="primary">{t('common.actions.save')}</Button></Form></Modal>
  </div>
}

function PositionForm({ control, t }: { control: ReturnType<typeof useForm<PositionValues>>['control']; t: TFunction }) {
  const field = (name: keyof PositionValues, node: (value: { value: unknown; onChange: (value: unknown) => void }) => ReactNode) => <Controller control={control} name={name} render={({ field: binding, fieldState }) => <Form.Item label={t(`system.org.positions.fields.${name}`)} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>{node(binding)}</Form.Item>} />
  return <>{field('code', (binding) => <Input {...binding} aria-label={t('system.org.positions.fields.code')} maxLength={64} value={binding.value as string} />)}{field('name', (binding) => <Input {...binding} aria-label={t('system.org.positions.fields.name')} maxLength={120} value={binding.value as string} />)}{field('description', (binding) => <Input.TextArea {...binding} aria-label={t('system.org.positions.fields.description')} maxLength={500} rows={4} value={binding.value as string} />)}{field('sort_order', (binding) => <InputNumber {...binding} aria-label={t('system.org.positions.fields.sort_order')} value={binding.value as number} />)}{field('status', (binding) => <Select {...binding} aria-label={t('system.org.positions.fields.status')} options={[{ label: t('common.states.enabled'), value: 'active' }, { label: t('common.states.disabled'), value: 'disabled' }]} />)}</>
}
