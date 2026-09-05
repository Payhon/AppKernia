import { Button, Card, Descriptions, Form, Input, Modal, Select, Space, Table, Tabs, Tag, Typography, type TableColumnsType } from 'antd'
import { useEffect, useMemo, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { withAdminBasePath } from '../app/base-path'
import type { AdminTenantMember, AdminTenantMemberAddRequest, AdminTenantStatus, AdminTenantUpdateRequest } from '../generated/api/types.gen'
import { useAuthStore } from '../features/auth/store'
import { useAdminTenant, useAdminTenantMembers, useAdminTenantMutations } from '../features/tenants/hooks'
import { ApiError } from '../shared/api/error'

const tenantSchema = z.object({ name: z.string().trim().min(1).max(120), status: z.enum(['active', 'disabled']) })
const memberSchema = z.object({ email: z.email(), display_name: z.string().trim().max(120) })
type TenantValues = z.infer<typeof tenantSchema>
type MemberValues = z.infer<typeof memberSchema>

export function AdminTenantDetailPage({ tenantId }: { tenantId: string }) {
  const { t, i18n } = useTranslation()
  const permissions = useMemo(() => new Set(useAuthStore.getState().context?.permissions ?? []), [])
  const currentUserId = useAuthStore((state) => state.context?.user.id)
  const tenant = useAdminTenant(tenantId)
  const members = useAdminTenantMembers(tenantId)
  const mutations = useAdminTenantMutations()
  const [memberOpen, setMemberOpen] = useState(false)
  const [feedback, setFeedback] = useState<string | null>(null)
  const tenantForm = useForm<TenantValues>({ defaultValues: { name: '', status: 'active' } })
  const memberForm = useForm<MemberValues>({ defaultValues: { email: '', display_name: '' } })

  useEffect(() => { if (tenant.data) tenantForm.reset({ name: tenant.data.name, status: tenant.data.status }) }, [tenant.data, tenantForm])
  const errorText = (error: unknown) => t(error instanceof ApiError ? error.messageKey : 'errors.common.unknown')
  const saveTenant = tenantForm.handleSubmit(async (values) => {
    const parsed = tenantSchema.safeParse(values)
    if (!parsed.success) { setFeedback(t('validation.invalid', { name: t('system.tenants.fields.name') })); return }
    try { await mutations.update.mutateAsync({ id: tenantId, input: parsed.data satisfies AdminTenantUpdateRequest }); setFeedback(t('system.tenants.feedback.saved')) } catch (error) { setFeedback(errorText(error)) }
  })
  const addMember = memberForm.handleSubmit(async (values) => {
    const parsed = memberSchema.safeParse(values)
    if (!parsed.success) { setFeedback(t('validation.invalid', { name: t('system.tenants.members.email') })); return }
    try { await mutations.addMember.mutateAsync({ id: tenantId, input: parsed.data satisfies AdminTenantMemberAddRequest }); setMemberOpen(false); memberForm.reset(); setFeedback(t('system.tenants.feedback.member_added')) } catch (error) { setFeedback(errorText(error)) }
  })
  const changeMember = (member: AdminTenantMember, status: 'active' | 'suspended' | 'left') => {
    Modal.confirm({ title: t(`system.tenants.members.confirm_${status}`), content: t('system.tenants.members.confirm_impact', { name: member.display_name }), okText: t('common.actions.confirm'), cancelText: t('common.actions.cancel'), okButtonProps: { danger: status !== 'active' }, onOk: async () => { try { if (status === 'left') await mutations.removeMember.mutateAsync({ id: tenantId, userId: member.user_id }); else await mutations.updateMember.mutateAsync({ id: tenantId, userId: member.user_id, input: { status } }); setFeedback(t('system.tenants.feedback.member_updated')) } catch (error) { setFeedback(errorText(error)) } } })
  }

  if (tenant.isPending) return <div className="ak-centered-state" aria-live="polite"><span className="ak-loading-indicator" /><span>{t('common.states.loading')}</span></div>
  if (tenant.isError) return <div className="ak-page-container"><div className="ak-form-error" role="alert">{t('system.tenants.load_error')} <Button onClick={() => { void tenant.refetch() }}>{t('common.actions.retry')}</Button></div></div>
  const value = tenant.data
  const columns: TableColumnsType<AdminTenantMember> = [
    { title: t('system.tenants.members.member'), key: 'member', render: (_, member) => <div><strong>{member.display_name}</strong><div className="ak-user-secondary">{member.email}</div></div> },
    { title: t('system.tenants.members.status'), dataIndex: 'status', render: (status: AdminTenantMember['status']) => <Tag className={status === 'active' ? 'ak-status-success' : 'ak-status-neutral'}>{t(`system.tenants.member_status.${status}`)}</Tag> },
    { title: t('system.tenants.members.roles'), dataIndex: 'role_codes', responsive: ['md'], render: (roles: string[]) => roles.map((role) => <Tag key={role}>{role}</Tag>) },
    { title: t('system.tenants.members.joined_at'), dataIndex: 'joined_at', responsive: ['lg'], render: (date: string) => new Intl.DateTimeFormat(i18n.language, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(date)) },
    { title: t('system.tenants.members.actions'), key: 'actions', render: (_, member) => <Space wrap>{member.status !== 'active' && permissions.has('iam.tenant.member.update') ? <Button onClick={() => { changeMember(member, 'active') }} size="small">{t('system.tenants.members.activate')}</Button> : null}{member.status === 'active' && permissions.has('iam.tenant.member.update') ? <Button disabled={member.user_id === currentUserId} onClick={() => { changeMember(member, 'suspended') }} size="small">{t('system.tenants.members.suspend')}</Button> : null}{member.status !== 'left' && permissions.has('iam.tenant.member.remove') ? <Button danger disabled={member.user_id === currentUserId} onClick={() => { changeMember(member, 'left') }} size="small">{t('system.tenants.members.remove')}</Button> : null}</Space> },
  ]

  const overview = <Card><Descriptions column={{ xs: 1, sm: 2 }} items={[{ key: 'code', label: t('system.tenants.fields.code'), children: value.code }, { key: 'status', label: t('system.tenants.fields.status'), children: <Tag className={value.status === 'active' ? 'ak-status-success' : 'ak-status-neutral'}>{t(`system.tenants.status.${value.status}`)}</Tag> }, { key: 'members', label: t('system.tenants.fields.members'), children: value.member_count }, { key: 'created', label: t('system.tenants.fields.created_at'), children: new Intl.DateTimeFormat(i18n.language, { dateStyle: 'long', timeStyle: 'short' }).format(new Date(value.created_at)) }]} />{permissions.has('iam.tenant.update') ? <Form className="ak-tenant-edit-form" layout="vertical" onFinish={() => { void saveTenant() }}><Controller control={tenantForm.control} name="name" render={({ field }) => <Form.Item label={t('system.tenants.fields.name')}><Input {...field} aria-label={t('system.tenants.fields.name')} /></Form.Item>} /><Controller control={tenantForm.control} name="status" render={({ field }) => <Form.Item label={t('system.tenants.fields.status')}><Select {...field} aria-label={t('system.tenants.fields.status')} options={(['active', 'disabled'] as AdminTenantStatus[]).map((status) => ({ label: t(`system.tenants.status.${status}`), value: status }))} /></Form.Item>} /><Button htmlType="submit" loading={mutations.update.isPending} type="primary">{t('common.actions.save')}</Button></Form> : null}</Card>
  const memberPanel = <Card extra={permissions.has('iam.tenant.member.invite') ? <Button onClick={() => { setMemberOpen(true); setFeedback(null) }} type="primary">{t('system.tenants.members.add')}</Button> : null}><Table columns={columns} dataSource={members.data ?? []} loading={members.isPending} locale={{ emptyText: t('system.tenants.members.empty') }} pagination={false} rowKey="user_id" scroll={{ x: 720 }} /></Card>

  return <div className="ak-page-container"><a className="ak-back-link" href={withAdminBasePath('/system/users/tenants')} onClick={(event) => { event.preventDefault(); window.history.back() }}>{t('system.tenants.detail.back')}</a><header className="ak-page-heading ak-user-detail-heading"><div><Typography.Title level={1}>{value.name}</Typography.Title><Typography.Paragraph type="secondary">{value.code}</Typography.Paragraph></div><Tag className={value.status === 'active' ? 'ak-status-success' : 'ak-status-neutral'}>{t(`system.tenants.status.${value.status}`)}</Tag></header>{feedback ? <div className="ak-org-feedback" role="status">{feedback}</div> : null}<Tabs items={[{ key: 'overview', label: t('system.tenants.detail.overview'), children: overview }, { key: 'members', label: t('system.tenants.detail.members'), children: memberPanel }]} /><Modal footer={null} onCancel={() => { setMemberOpen(false) }} open={memberOpen} title={t('system.tenants.members.add_title')}><Form layout="vertical" onFinish={() => { void addMember() }}><Controller control={memberForm.control} name="email" render={({ field, fieldState }) => <Form.Item label={t('system.tenants.members.email')} {...(fieldState.error ? { help: fieldState.error.message, validateStatus: 'error' as const } : {})}><Input {...field} aria-label={t('system.tenants.members.email')} type="email" /></Form.Item>} /><Controller control={memberForm.control} name="display_name" render={({ field }) => <Form.Item label={t('system.tenants.members.display_name')}><Input {...field} aria-label={t('system.tenants.members.display_name')} /></Form.Item>} /><Button htmlType="submit" loading={mutations.addMember.isPending} type="primary">{t('system.tenants.members.add')}</Button></Form></Modal></div>
}
