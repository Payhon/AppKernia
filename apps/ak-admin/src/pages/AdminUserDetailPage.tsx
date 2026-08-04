import { Link } from '@tanstack/react-router'
import { Button, Card, Descriptions, Empty, Select, Space, Tabs, Tag, Typography } from 'antd'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import type { AdminOrgUnit } from '../generated/api/types.gen'
import { useAuthStore } from '../features/auth/store'
import { useOrgPositions, useOrgUnits } from '../features/org/hooks'
import { useAdminUser, useAdminUserMutations, useAdminUserRoleOptions, useAdminUserSessions } from '../features/users/hooks'
import { ApiError } from '../shared/api/error'

function flatUnits(units: AdminOrgUnit[]): AdminOrgUnit[] { return units.flatMap((unit) => [unit, ...flatUnits(unit.children)]) }

export function AdminUserDetailPage({ userId }: { userId: string }) {
  const { t, i18n } = useTranslation()
  const permissions = useMemo(() => new Set(useAuthStore.getState().context?.permissions ?? []), [])
  const user = useAdminUser(userId)
  const roles = useAdminUserRoleOptions(permissions.has('iam.user.assign_role'))
  const units = useOrgUnits('', '')
  const positions = useOrgPositions('', '', '')
  const sessions = useAdminUserSessions(userId, permissions.has('iam.session.read'))
  const mutations = useAdminUserMutations()
  const [roleIDs, setRoleIDs] = useState<string[]>([])
  const [unitIDs, setUnitIDs] = useState<string[]>([])
  const [positionIDs, setPositionIDs] = useState<string[]>([])
  const [feedback, setFeedback] = useState<string | null>(null)

  useEffect(() => { if(user.data){setRoleIDs(user.data.roles.map((item)=>item.id));setUnitIDs(user.data.units.map((item)=>item.id));setPositionIDs(user.data.positions.map((item)=>item.id))} }, [user.data])
  const errorText = (error: unknown) => t(error instanceof ApiError ? error.messageKey : 'errors.common.unknown')
  const saveRoles = async () => { try{await mutations.roles.mutateAsync({id:userId,input:{role_ids:roleIDs}});setFeedback(t('system.users.feedback.roles_saved'))}catch(error){setFeedback(errorText(error))} }
  const saveAssignments = async () => { try{await mutations.assignments.mutateAsync({id:userId,input:{unit_ids:unitIDs,primary_unit_id:unitIDs[0]??null,position_ids:positionIDs,primary_position_id:positionIDs[0]??null}});setFeedback(t('system.users.feedback.assignments_saved'))}catch(error){setFeedback(errorText(error))} }
  const revoke = async (sessionId: string) => { try{await mutations.revokeSession.mutateAsync({id:userId,sessionId});await sessions.refetch();setFeedback(t('system.users.feedback.session_revoked'))}catch(error){setFeedback(errorText(error))} }

  if(user.isPending)return <div className="ak-centered-state" aria-live="polite"><span className="ak-loading-indicator"/><span>{t('common.states.loading')}</span></div>
  if(user.isError)return <div className="ak-page-container"><div className="ak-form-error" role="alert">{t('system.users.load_error')} <Button onClick={()=>{void user.refetch()}}>{t('common.actions.retry')}</Button></div></div>
  const current=user.data
  const overview=<Card><Descriptions column={{xs:1,sm:2}} items={[
    {key:'email',label:t('system.users.fields.email'),children:current.email},
    {key:'status',label:t('system.users.fields.status'),children:<Tag className={current.status==='active'?'ak-status-success':'ak-status-neutral'}>{t(`system.users.status.${current.status}`)}</Tag>},
    {key:'locale',label:t('system.users.fields.locale'),children:t(`common.locales.${current.locale}`)},
    {key:'time_zone',label:t('system.users.fields.time_zone'),children:current.time_zone},
    {key:'created',label:t('system.users.fields.created_at'),children:new Intl.DateTimeFormat(i18n.language,{dateStyle:'long',timeStyle:'short'}).format(new Date(current.created_at))},
    {key:'last_login',label:t('system.users.fields.last_login'),children:current.last_login_at?new Intl.DateTimeFormat(i18n.language,{dateStyle:'long',timeStyle:'short'}).format(new Date(current.last_login_at)):t('common.states.empty')},
  ]}/></Card>
  const access=<div className="ak-detail-grid"><Card title={t('system.users.detail.roles')}><Select aria-label={t('system.users.detail.roles')} loading={roles.isPending} mode="multiple" onChange={setRoleIDs} options={(roles.data?.items??[]).map((item)=>({label:`${item.name} (${item.code})`,value:item.id}))} value={roleIDs}/>{permissions.has('iam.user.assign_role')?<Button loading={mutations.roles.isPending} onClick={()=>{void saveRoles()}} type="primary">{t('common.actions.save')}</Button>:null}</Card><Card title={t('system.users.detail.assignments')}><Select aria-label={t('system.users.fields.department')} mode="multiple" onChange={setUnitIDs} options={flatUnits(units.data??[]).map((item)=>({label:item.name,value:item.id}))} placeholder={t('system.users.fields.department')} value={unitIDs}/><Select aria-label={t('system.users.fields.positions')} mode="multiple" onChange={setPositionIDs} options={(positions.data?.items??[]).map((item)=>({label:item.name,value:item.id}))} placeholder={t('system.users.fields.positions')} value={positionIDs}/>{permissions.has('org.assignment.update')?<Button loading={mutations.assignments.isPending} onClick={()=>{void saveAssignments()}} type="primary">{t('common.actions.save')}</Button>:null}</Card></div>
  const sessionPanel=<div className="ak-session-grid">{sessions.isPending?<div aria-live="polite">{t('common.states.loading')}</div>:null}{sessions.data?.items.length===0?<Card><Empty description={t('system.users.sessions.empty')}/></Card>:sessions.data?.items.map((session)=><Card key={session.id} title={<Space><span>{session.audience}</span><Tag className={session.status==='active'?'ak-status-success':'ak-status-neutral'}>{t(`system.users.sessions.status.${session.status}`)}</Tag></Space>}><Descriptions column={1} items={[{key:'ip',label:t('system.users.sessions.ip'),children:session.ip_address||t('common.states.empty')},{key:'seen',label:t('system.users.sessions.last_seen'),children:new Intl.DateTimeFormat(i18n.language,{dateStyle:'medium',timeStyle:'short'}).format(new Date(session.last_seen_at))},{key:'agent',label:t('system.users.sessions.user_agent'),children:session.user_agent||t('common.states.empty')}]} />{session.status==='active'&&!session.current&&permissions.has('iam.session.revoke')?<Button danger loading={mutations.revokeSession.isPending} onClick={()=>{void revoke(session.id)}}>{t('system.users.sessions.revoke')}</Button>:null}</Card>)}</div>

  return <div className="ak-page-container"><Link className="ak-back-link" to="/system/users/accounts">{t('system.users.detail.back')}</Link><header className="ak-page-heading ak-user-detail-heading"><div><Typography.Title level={1}>{current.display_name}</Typography.Title><Typography.Paragraph type="secondary">{current.email}</Typography.Paragraph></div><Tag className={current.status==='active'?'ak-status-success':'ak-status-neutral'}>{t(`system.users.status.${current.status}`)}</Tag></header>{feedback?<div className="ak-org-feedback" role="status">{feedback}</div>:null}<Tabs items={[{key:'overview',label:t('system.users.detail.overview'),children:overview},{key:'access',label:t('system.users.detail.access'),children:access},...(permissions.has('iam.session.read')?[{key:'sessions',label:t('system.users.detail.sessions'),children:sessionPanel}]:[])]}/></div>
}
