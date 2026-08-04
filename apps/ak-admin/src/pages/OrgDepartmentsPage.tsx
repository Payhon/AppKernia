import { Button, Card, Descriptions, Empty, Form, Input, InputNumber, Modal, Select, Space, Tag, Tree, Typography, type TreeDataNode } from 'antd'
import type { TFunction } from 'i18next'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import type { AdminOrgUnit, AdminOrgUnitRequest } from '../generated/api/types.gen'
import { useAuthStore } from '../features/auth/store'
import { useOrgUnitMutations, useOrgUnits } from '../features/org/hooks'
import { ApiError } from '../shared/api/error'

const unitSchema = z.object({ parent_id: z.string().nullable(), code: z.string().trim().min(2).max(64), name: z.string().trim().min(1).max(160), unit_type: z.enum(['company', 'division', 'department', 'team', 'group']), phone: z.string().trim().max(32), email: z.union([z.literal(''), z.email()]), sort_order: z.number().int(), status: z.enum(['active', 'disabled']) })
type UnitValues = z.infer<typeof unitSchema>

function flatten(units: AdminOrgUnit[]): AdminOrgUnit[] { return units.flatMap((unit) => [unit, ...flatten(unit.children)]) }
function treeData(units: AdminOrgUnit[]): TreeDataNode[] { return units.map((unit) => ({ key: unit.id, title: `${unit.name} (${unit.code})`, children: treeData(unit.children) })) }
function initialSearch() { const params = new URLSearchParams(window.location.search); return { q: params.get('q') ?? '', status: params.get('status') ?? '' } }

export function OrgDepartmentsPage() {
  const { t } = useTranslation()
  const permissions = useMemo(() => new Set(useAuthStore.getState().context?.permissions ?? []), [])
  const initial = useMemo(initialSearch, [])
  const [query, setQuery] = useState(initial.q)
  const [status, setStatus] = useState(initial.status)
  const [selectedID, setSelectedID] = useState<string | null>(null)
  const [editor, setEditor] = useState<'create' | 'edit' | null>(null)
  const [moveOpen, setMoveOpen] = useState(false)
  const [moveParent, setMoveParent] = useState<string | null>(null)
  const [moveSort, setMoveSort] = useState(0)
  const [feedback, setFeedback] = useState<string | null>(null)
  const units = useOrgUnits(query, status)
  const mutations = useOrgUnitMutations()
  const allUnits = useMemo(() => flatten(units.data ?? []), [units.data])
  const selected = allUnits.find((unit) => unit.id === selectedID) ?? allUnits[0]
  const { control, handleSubmit, reset, setError, formState } = useForm<UnitValues>({ defaultValues: { parent_id: null, code: '', name: '', unit_type: 'department', phone: '', email: '', sort_order: 0, status: 'active' } })

  useEffect(() => { const params = new URLSearchParams(); if (query) params.set('q', query); if (status) params.set('status', status); window.history.replaceState(null, '', `${window.location.pathname}${params.size ? `?${params}` : ''}`) }, [query, status])
  useEffect(() => { if (selected && !selectedID) setSelectedID(selected.id) }, [selected, selectedID])

  const openEditor = (mode: 'create' | 'edit') => {
    setFeedback(null); setEditor(mode)
    reset(mode === 'edit' && selected ? { parent_id: selected.parent_id, code: selected.code, name: selected.name, unit_type: selected.unit_type, phone: selected.phone, email: selected.email, sort_order: selected.sort_order, status: selected.status } : { parent_id: selected?.id ?? null, code: '', name: '', unit_type: 'department', phone: '', email: '', sort_order: 0, status: 'active' })
  }
  const submit = handleSubmit(async (values) => {
    const parsed = unitSchema.safeParse(values)
    if (!parsed.success) { for (const issue of parsed.error.issues) { const field = issue.path[0] as keyof UnitValues; setError(field, { message: t('validation.invalid', { name: t(`system.org.units.fields.${field}`) }) }) }; return }
    try {
      const input = parsed.data satisfies AdminOrgUnitRequest
      if (editor === 'edit' && selected) await mutations.update.mutateAsync({ id: selected.id, input }); else await mutations.create.mutateAsync(input)
      setEditor(null); setFeedback(t('system.org.feedback.saved'))
    } catch (error) { setFeedback(t(error instanceof ApiError ? error.messageKey : 'errors.common.unknown')) }
  })
  const submitMove = async () => { if (!selected) return; try { await mutations.move.mutateAsync({ id: selected.id, input: { parent_id: moveParent, sort_order: moveSort } }); setMoveOpen(false); setFeedback(t('system.org.feedback.moved')) } catch (error) { setFeedback(t(error instanceof ApiError ? error.messageKey : 'errors.common.unknown')) } }
  const remove = () => { if (!selected) return; Modal.confirm({ title: t('system.org.units.delete.title'), content: t('system.org.units.delete.impact', { children: selected.child_count, members: selected.direct_member_count }), okText: t('common.actions.delete'), cancelText: t('common.actions.cancel'), okButtonProps: { danger: true }, onOk: async () => { try { await mutations.remove.mutateAsync(selected.id); setSelectedID(null); setFeedback(t('system.org.feedback.deleted')) } catch (error) { setFeedback(t(error instanceof ApiError ? error.messageKey : 'errors.common.unknown')) } } }) }

  if (units.isPending) return <div className="ak-centered-state" aria-live="polite"><span className="ak-loading-indicator" /><span>{t('common.states.loading')}</span></div>
  return <div className="ak-page-container">
    <header className="ak-page-heading ak-org-heading"><div><Typography.Title level={1}>{t('routes.system.users.departments.title')}</Typography.Title><Typography.Paragraph type="secondary">{t('system.org.units.description')}</Typography.Paragraph></div>{permissions.has('org.unit.create') ? <Button onClick={() => { openEditor('create') }} type="primary">{t('system.org.units.actions.create')}</Button> : null}</header>
    <div className="ak-org-filters" role="search"><Input allowClear aria-label={t('system.org.filters.query')} onChange={(event) => { setQuery(event.target.value) }} placeholder={t('system.org.filters.query')} value={query} /><Select allowClear aria-label={t('system.org.filters.status')} onChange={(value) => { setStatus(value ?? '') }} options={[{ label: t('common.states.enabled'), value: 'active' }, { label: t('common.states.disabled'), value: 'disabled' }]} placeholder={t('system.org.filters.status')} value={status || undefined} /></div>
    {feedback ? <div className="ak-org-feedback" role="status">{feedback}</div> : null}
    {units.isError ? <div className="ak-form-error" role="alert">{t('system.org.load_error')} <Button onClick={() => { void units.refetch() }}>{t('common.actions.retry')}</Button></div> : null}
    {!units.isError && allUnits.length === 0 ? <Card><Empty description={t('system.org.units.empty')} /></Card> : null}
    {allUnits.length > 0 ? <div className="ak-org-split"><Card title={t('system.org.units.tree_label')}><Tree aria-label={t('system.org.units.tree_label')} defaultExpandAll onSelect={(keys) => { setSelectedID(String(keys[0] ?? '')) }} selectedKeys={selected ? [selected.id] : []} treeData={treeData(units.data ?? [])} /></Card><Card title={selected?.name} extra={selected ? <Tag className={selected.status === 'active' ? 'ak-status-success' : 'ak-status-neutral'}>{t(selected.status === 'active' ? 'common.states.enabled' : 'common.states.disabled')}</Tag> : null}>{selected ? <><Descriptions column={1} items={[{ key: 'code', label: t('system.org.units.fields.code'), children: selected.code }, { key: 'type', label: t('system.org.units.fields.unit_type'), children: t(`system.org.unit_types.${selected.unit_type}`) }, { key: 'members', label: t('system.org.units.fields.direct_member_count'), children: selected.direct_member_count }, { key: 'children', label: t('system.org.units.fields.child_count'), children: selected.child_count }, { key: 'contact', label: t('system.org.units.fields.contact'), children: selected.email || selected.phone || t('common.states.empty') }]} /><Space wrap>{permissions.has('org.unit.update') ? <Button onClick={() => { openEditor('edit') }}>{t('common.actions.edit')}</Button> : null}{permissions.has('org.unit.move') ? <Button onClick={() => { setMoveParent(selected.parent_id); setMoveSort(selected.sort_order); setMoveOpen(true) }}>{t('system.org.units.actions.move')}</Button> : null}{permissions.has('org.unit.delete') ? <Button danger onClick={remove}>{t('common.actions.delete')}</Button> : null}</Space></> : null}</Card></div> : null}
    <Modal destroyOnHidden footer={null} onCancel={() => { setEditor(null) }} open={editor !== null} title={t(editor === 'edit' ? 'system.org.units.editor.edit_title' : 'system.org.units.editor.create_title')}><Form layout="vertical" onFinish={() => { void submit() }} requiredMark={false}><UnitForm control={control} parentOptions={allUnits.filter((item) => item.id !== selected?.id).map((item) => ({ label: item.name, value: item.id }))} t={t} /><Button htmlType="submit" loading={formState.isSubmitting} type="primary">{t('common.actions.save')}</Button></Form></Modal>
    <Modal confirmLoading={mutations.move.isPending} onCancel={() => { setMoveOpen(false) }} onOk={() => { void submitMove() }} open={moveOpen} title={t('system.org.units.move.title')}><Typography.Paragraph>{t('system.org.units.move.description')}</Typography.Paragraph><Form layout="vertical"><Form.Item label={t('system.org.units.fields.parent_id')}><Select allowClear aria-label={t('system.org.units.fields.parent_id')} onChange={(value) => { setMoveParent(value ?? null) }} options={allUnits.filter((item) => item.id !== selected?.id).map((item) => ({ label: item.name, value: item.id }))} placeholder={t('system.org.units.move.root')} value={moveParent ?? undefined} /></Form.Item><Form.Item label={t('system.org.units.fields.sort_order')}><InputNumber aria-label={t('system.org.units.fields.sort_order')} onChange={(value) => { setMoveSort(value ?? 0) }} value={moveSort} /></Form.Item></Form></Modal>
  </div>
}

function UnitForm({ control, parentOptions, t }: { control: ReturnType<typeof useForm<UnitValues>>['control']; parentOptions: {label: string; value: string}[]; t: TFunction }) {
  const field = (name: keyof UnitValues, node: (value: { value: unknown; onChange: (value: unknown) => void }) => ReactNode) => <Controller control={control} name={name} render={({ field: binding, fieldState }) => <Form.Item label={t(`system.org.units.fields.${name}`)} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}>{node(binding)}</Form.Item>} />
  return <>{field('parent_id', (binding) => <Select {...binding} allowClear aria-label={t('system.org.units.fields.parent_id')} options={parentOptions} placeholder={t('system.org.units.move.root')} />)}{field('code', (binding) => <Input {...binding} aria-label={t('system.org.units.fields.code')} maxLength={64} value={binding.value as string} />)}{field('name', (binding) => <Input {...binding} aria-label={t('system.org.units.fields.name')} maxLength={160} value={binding.value as string} />)}{field('unit_type', (binding) => <Select {...binding} aria-label={t('system.org.units.fields.unit_type')} options={['company','division','department','team','group'].map((value) => ({ label: t(`system.org.unit_types.${value}`), value }))} />)}{field('phone', (binding) => <Input {...binding} aria-label={t('system.org.units.fields.phone')} maxLength={32} value={binding.value as string} />)}{field('email', (binding) => <Input {...binding} aria-label={t('system.org.units.fields.email')} maxLength={320} type="email" value={binding.value as string} />)}{field('sort_order', (binding) => <InputNumber {...binding} aria-label={t('system.org.units.fields.sort_order')} value={binding.value as number} />)}{field('status', (binding) => <Select {...binding} aria-label={t('system.org.units.fields.status')} options={[{ label: t('common.states.enabled'), value: 'active' }, { label: t('common.states.disabled'), value: 'disabled' }]} />)}</>
}
