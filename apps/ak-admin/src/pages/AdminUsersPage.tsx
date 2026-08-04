import { Link } from '@tanstack/react-router'
import { Button, Card, Drawer, Dropdown, Form, Grid, Input, Modal, Select, Space, Table, Tag, Tree, Typography, Upload, type MenuProps, type TableColumnsType, type TreeDataNode } from 'antd'
import type { TFunction } from 'i18next'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { Controller, useForm, type Control, type FieldValues, type Path, type UseFormSetError } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import type { AdminOrgUnit, AdminUser, AdminUserCreateRequestWritable, AdminUserExportRequest, AdminUserStatus, AdminUserUpdateRequest } from '../generated/api/types.gen'
import { authSession } from '../features/auth/store'
import { useAuthStore } from '../features/auth/store'
import { useOrgUnits } from '../features/org/hooks'
import { useAdminUserMutations, useAdminUsers } from '../features/users/hooks'
import { ApiError } from '../shared/api/error'

const createSchema = z.object({ email: z.email(), display_name: z.string().trim().min(1).max(120), locale: z.enum(['zh-CN', 'en-US']), time_zone: z.string().trim().min(1).max(64), temporary_password: z.string().min(12).max(256) })
const editSchema = z.object({ display_name: z.string().trim().min(1).max(120), locale: z.enum(['zh-CN', 'en-US']), time_zone: z.string().trim().min(1).max(64) })
type CreateValues = z.infer<typeof createSchema>
type EditValues = z.infer<typeof editSchema>

type UserSort = 'created_desc' | 'created_asc' | 'name_asc' | 'last_login_desc'
interface SearchState { q: string; status: AdminUserStatus | ''; unit_id: string; page: number; page_size: number; sort: UserSort }
function readSearch(): SearchState { const p=new URLSearchParams(window.location.search);const status=p.get('status');const sort=p.get('sort');return {q:p.get('q')??'',status:status==='active'||status==='disabled'||status==='pending'||status==='locked'?status:'',unit_id:p.get('unit_id')??'',page:Number(p.get('page')??1)||1,page_size:Number(p.get('page_size')??20)||20,sort:sort==='created_asc'||sort==='name_asc'||sort==='last_login_desc'?sort:'created_desc'} }
function unitTree(units: AdminOrgUnit[]): TreeDataNode[] { return units.map((unit) => ({key:unit.id,title:unit.name,children:unitTree(unit.children)})) }

export function AdminUsersPage() {
  const { t, i18n } = useTranslation()
  const screens = Grid.useBreakpoint()
  const initial = useMemo(readSearch, [])
  const [search, setSearch] = useState(initial)
  const [selected, setSelected] = useState<React.Key[]>([])
  const [editor, setEditor] = useState<'create' | AdminUser | null>(null)
  const [resetTarget, setResetTarget] = useState<AdminUser | null>(null)
  const [feedback, setFeedback] = useState<string | null>(null)
  const [busyBulk, setBusyBulk] = useState(false)
  const permissions = useMemo(() => new Set(useAuthStore.getState().context?.permissions ?? []), [])
  const users = useAdminUsers(search)
  const units = useOrgUnits('', '')
  const mutations = useAdminUserMutations()
  const createForm = useForm<CreateValues>({defaultValues:{email:'',display_name:'',locale:'zh-CN',time_zone:'Asia/Shanghai',temporary_password:''}})
  const editForm = useForm<EditValues>({defaultValues:{display_name:'',locale:'zh-CN',time_zone:'Asia/Shanghai'}})
  const resetForm = useForm<{temporary_password:string}>({defaultValues:{temporary_password:''}})

  useEffect(() => { const p=new URLSearchParams();Object.entries(search).forEach(([key,value])=>{if(value!==''&&!(key==='page'&&value===1)&&!(key==='page_size'&&value===20)&&!(key==='sort'&&value==='created_desc'))p.set(key,String(value))});window.history.replaceState(null,'',`${window.location.pathname}${p.size?`?${p}`:''}`) }, [search])
  const patchSearch = (value: Partial<SearchState>) => { setSelected([]);setSearch((current)=>({...current,...value,page:value.page??1})) }
  const errorText = (error: unknown) => t(error instanceof ApiError ? error.messageKey : 'errors.common.unknown')
  const openCreate = () => { createForm.reset({email:'',display_name:'',locale:'zh-CN',time_zone:'Asia/Shanghai',temporary_password:''});setEditor('create');setFeedback(null) }
  const openEdit = (user: AdminUser) => { editForm.reset({display_name:user.display_name,locale:user.locale,time_zone:user.time_zone});setEditor(user);setFeedback(null) }
  const submitCreate = createForm.handleSubmit(async (values) => { const parsed=createSchema.safeParse(values);if(!parsed.success){setSchemaErrors(parsed.error,createForm.setError,t);return}try{await mutations.create.mutateAsync(parsed.data satisfies AdminUserCreateRequestWritable);setEditor(null);setFeedback(t('system.users.feedback.created'))}catch(error){setFeedback(errorText(error))} })
  const submitEdit = editForm.handleSubmit(async (values) => { if(editor==='create'||editor===null)return;const parsed=editSchema.safeParse(values);if(!parsed.success){setSchemaErrors(parsed.error,editForm.setError,t);return}try{await mutations.update.mutateAsync({id:editor.id,input:parsed.data satisfies AdminUserUpdateRequest});setEditor(null);setFeedback(t('system.users.feedback.saved'))}catch(error){setFeedback(errorText(error))} })
  const setStatus = (user: AdminUser, enabled: boolean) => { Modal.confirm({title:t(enabled?'system.users.enable.title':'system.users.disable.title'),content:t(enabled?'system.users.enable.impact':'system.users.disable.impact',{name:user.display_name,sessions:user.active_session_count}),okText:t(enabled?'system.users.actions.enable':'system.users.actions.disable'),cancelText:t('common.actions.cancel'),okButtonProps:{danger:!enabled},onOk:async()=>{try{await mutations.status.mutateAsync({id:user.id,enabled});setFeedback(t('system.users.feedback.status'))}catch(error){setFeedback(errorText(error))}}}) }
  const bulkStatus = (enabled: boolean) => { const targets=(users.data?.items??[]).filter((item)=>selected.includes(item.id));Modal.confirm({title:t(enabled?'system.users.bulk.enable_title':'system.users.bulk.disable_title',{count:targets.length}),content:t(enabled?'system.users.bulk.enable_impact':'system.users.bulk.disable_impact',{count:targets.length}),okText:t(enabled?'system.users.actions.enable':'system.users.actions.disable'),cancelText:t('common.actions.cancel'),okButtonProps:{danger:!enabled},onOk:async()=>{setBusyBulk(true);try{for(const user of targets)await authSession.setAdminUserEnabled(user.id,enabled);await users.refetch();setSelected([]);setFeedback(t('system.users.feedback.bulk',{count:targets.length}))}catch(error){setFeedback(errorText(error))}finally{setBusyBulk(false)}}}) }
  const submitReset = resetForm.handleSubmit(async (values)=>{if(!resetTarget)return;if(values.temporary_password.length<12){resetForm.setError('temporary_password',{message:t('validation.invalid',{name:t('system.users.fields.temporary_password')})});return}try{const result=await mutations.resetPassword.mutateAsync({id:resetTarget.id,input:values});setResetTarget(null);resetForm.reset();setFeedback(t('system.users.feedback.password_reset',{count:result.sessions_revoked}))}catch(error){setFeedback(errorText(error))}})
  const importFile = async (file: File) => { try{const result=await authSession.importAdminUsers(await file.text());await users.refetch();setFeedback(t('system.users.feedback.imported',{created:result.created,failed:result.failed}))}catch(error){setFeedback(errorText(error))}return false }
  const exportUsers = async () => { try{const input:AdminUserExportRequest={sort:search.sort,...(search.q?{q:search.q}:{}),...(search.status?{status:search.status}:{}),...(search.unit_id?{unit_id:search.unit_id}:{})};const blob=await authSession.exportAdminUsers(input);const url=URL.createObjectURL(blob);const link=document.createElement('a');link.href=url;link.download='users.csv';link.click();URL.revokeObjectURL(url);setFeedback(t('system.users.feedback.exported'))}catch(error){setFeedback(errorText(error))} }
  const statusTag = (value:AdminUser['status']) => <Tag className={value==='active'?'ak-status-success':value==='locked'?'ak-status-warning':'ak-status-neutral'}>{t(`system.users.status.${value}`)}</Tag>
  const mobileActions = (user: AdminUser): MenuProps => ({items:[permissions.has('iam.user.update')?{key:'edit',label:t('common.actions.edit')}:null,user.status==='active'&&permissions.has('iam.user.disable')?{key:'disable',danger:true,label:t('system.users.actions.disable')}:null,user.status!=='active'&&permissions.has('iam.user.enable')?{key:'enable',label:t('system.users.actions.enable')}:null,permissions.has('iam.user.reset_password')?{key:'reset',label:t('system.users.actions.reset_password')}:null].filter((item):item is Exclude<typeof item,null>=>item!==null),onClick:({key})=>{if(key==='edit')openEdit(user);if(key==='disable')setStatus(user,false);if(key==='enable')setStatus(user,true);if(key==='reset'){resetForm.reset();setResetTarget(user)}}})
  const columns: TableColumnsType<AdminUser> = [
    {title:t('system.users.fields.user'),key:'user',...(screens.md?{}:{width:174}),render:(_,user)=><div><Link to="/system/users/accounts/$userId" params={{userId:user.id}}><strong>{user.display_name}</strong></Link><div className="ak-user-secondary">{user.email}</div>{screens.md?null:<div className="ak-user-compact-status">{statusTag(user.status)}</div>}</div>},
    {title:t('system.users.fields.status'),dataIndex:'status',responsive:['md'],render:(value:AdminUser['status'])=>statusTag(value)},
    {title:t('system.users.fields.department'),key:'units',responsive:['md'],render:(_,user)=>user.units.map((item)=>item.name).join(' · ')||t('common.states.empty')},
    {title:t('system.users.fields.roles'),key:'roles',responsive:['md'],render:(_,user)=>user.roles.map((item)=><Tag key={item.id}>{item.name}</Tag>)},
    {title:t('system.users.fields.last_login'),dataIndex:'last_login_at',responsive:['md'],render:(value:string|null)=>value?new Intl.DateTimeFormat(i18n.language,{dateStyle:'medium',timeStyle:'short'}).format(new Date(value)):t('common.states.empty')},
    {title:t('system.users.actions.label'),key:'actions',...(screens.md?{fixed:'right' as const,width:260}:{width:80}),render:(_,user)=>screens.md?<Space wrap>{permissions.has('iam.user.update')?<Button onClick={()=>{openEdit(user)}} size="small">{t('common.actions.edit')}</Button>:null}{user.status==='active'&&permissions.has('iam.user.disable')?<Button danger onClick={()=>{setStatus(user,false)}} size="small">{t('system.users.actions.disable')}</Button>:null}{user.status!=='active'&&permissions.has('iam.user.enable')?<Button onClick={()=>{setStatus(user,true)}} size="small">{t('system.users.actions.enable')}</Button>:null}{permissions.has('iam.user.reset_password')?<Button onClick={()=>{resetForm.reset();setResetTarget(user)}} size="small">{t('system.users.actions.reset_password')}</Button>:null}</Space>:<Dropdown menu={mobileActions(user)} trigger={['click']}><Button aria-label={t('system.users.actions.label')} size="small">{t('system.users.actions.more')}</Button></Dropdown>},
  ]

  return <div className="ak-page-container"><header className="ak-page-heading ak-users-heading"><div><Typography.Title level={1}>{t('routes.system.users.accounts.title')}</Typography.Title><Typography.Paragraph type="secondary">{t('system.users.description')}</Typography.Paragraph></div><Space wrap>{permissions.has('iam.user.import')?<Upload accept=".csv,text/csv" beforeUpload={importFile} maxCount={1} showUploadList={false}><Button>{t('system.users.actions.import')}</Button></Upload>:null}{permissions.has('iam.user.export')?<Button onClick={()=>{void exportUsers()}}>{t('system.users.actions.export')}</Button>:null}{permissions.has('iam.user.create')?<Button onClick={openCreate} type="primary">{t('system.users.actions.create')}</Button>:null}</Space></header>
    <div className="ak-users-layout"><Card className="ak-users-department-card" title={t('system.users.filters.department')}><Tree aria-label={t('system.users.filters.department')} defaultExpandAll onSelect={(keys)=>{patchSearch({unit_id:String(keys[0]??'')})}} selectedKeys={search.unit_id?[search.unit_id]:[]} treeData={unitTree(units.data??[])} /><Button block onClick={()=>{patchSearch({unit_id:''})}} type={search.unit_id?'default':'primary'}>{t('system.users.filters.all_departments')}</Button></Card><div className="ak-users-main"><div className="ak-users-filters" role="search"><Input.Search allowClear aria-label={t('system.users.filters.query')} onChange={(event)=>{patchSearch({q:event.target.value})}} placeholder={t('system.users.filters.query')} value={search.q} /><Select allowClear aria-label={t('system.users.filters.status')} onChange={(value)=>{patchSearch({status:value??''})}} options={['active','disabled','pending','locked'].map((value)=>({label:t(`system.users.status.${value}`),value}))} placeholder={t('system.users.filters.status')} value={search.status||undefined} /><Select aria-label={t('system.users.filters.sort')} onChange={(value)=>{patchSearch({sort:value})}} options={['created_desc','created_asc','name_asc','last_login_desc'].map((value)=>({label:t(`system.users.sort.${value}`),value}))} value={search.sort} /></div>
      {feedback?<div className="ak-org-feedback" role="status">{feedback}</div>:null}{users.isError?<div className="ak-form-error" role="alert">{t('system.users.load_error')} <Button onClick={()=>{void users.refetch()}}>{t('common.actions.retry')}</Button></div>:null}
      {selected.length>0?<div className="ak-bulk-bar" role="status"><span>{t('system.users.bulk.selected',{count:selected.length})}</span><Space>{permissions.has('iam.user.enable')?<Button disabled={busyBulk} onClick={()=>{bulkStatus(true)}}>{t('system.users.actions.enable')}</Button>:null}{permissions.has('iam.user.disable')?<Button danger disabled={busyBulk} onClick={()=>{bulkStatus(false)}}>{t('system.users.actions.disable')}</Button>:null}<Button onClick={()=>{setSelected([])}}>{t('common.actions.cancel')}</Button></Space></div>:null}
      <Card>
        <Table<AdminUser>
          columns={columns}
          dataSource={users.data?.items??[]}
          loading={users.isPending}
          locale={{emptyText:t('system.users.empty')}}
          pagination={{
            current:search.page,
            pageSize:search.page_size,
            total:users.data?.total??0,
            showSizeChanger:true,
            onChange:(page,pageSize)=>{setSearch((current)=>({...current,page,page_size:pageSize}))},
          }}
          rowKey="id"
          rowSelection={{
            selectedRowKeys:selected,
            onChange:setSelected,
            getCheckboxProps:(user)=>({disabled:user.is_system,'aria-label':t('system.users.bulk.select_user',{name:user.display_name})}),
          }}
          scroll={{x:screens.md?1050:286}}
        />
      </Card></div></div>
    <Drawer destroyOnHidden onClose={()=>{setEditor(null)}} open={editor!==null} size="large" title={t(editor==='create'?'system.users.editor.create_title':'system.users.editor.edit_title')}><Form layout="vertical" onFinish={()=>{void (editor==='create'?submitCreate():submitEdit())}} requiredMark={false}>{editor==='create'?<CreateForm control={createForm.control} t={t}/>:<EditForm control={editForm.control} t={t}/>}<Button htmlType="submit" loading={createForm.formState.isSubmitting||editForm.formState.isSubmitting} type="primary">{t('common.actions.save')}</Button></Form></Drawer>
    <Modal footer={null} onCancel={()=>{setResetTarget(null)}} open={resetTarget!==null} title={t('system.users.reset.title')}><Typography.Paragraph>{t('system.users.reset.impact',{name:resetTarget?.display_name??'',sessions:resetTarget?.active_session_count??0})}</Typography.Paragraph><Form layout="vertical" onFinish={()=>{void submitReset()}}><Controller control={resetForm.control} name="temporary_password" render={({field,fieldState})=><Form.Item label={t('system.users.fields.temporary_password')} {...(fieldState.error?{help:fieldState.error.message,validateStatus:'error' as const}:{})}><Input.Password {...field} aria-label={t('system.users.fields.temporary_password')} autoComplete="new-password" /></Form.Item>} /><Button danger htmlType="submit" loading={resetForm.formState.isSubmitting} type="primary">{t('system.users.actions.reset_password')}</Button></Form></Modal>
  </div>
}

function setSchemaErrors<T extends FieldValues>(error:z.ZodError<T>,setError:UseFormSetError<T>,t:TFunction){for(const issue of error.issues){const name=String(issue.path[0]) as Path<T>;setError(name,{message:t('validation.invalid',{name:t(`system.users.fields.${name}`)})})}}
function controlledField<T extends FieldValues>({control,name,t,node}:{control:Control<T>;name:Path<T>;t:TFunction;node:(field:{value:unknown;onChange:(value:unknown)=>void})=>ReactNode}){return <Controller control={control} name={name} render={({field,fieldState})=><Form.Item label={t(`system.users.fields.${name}`)} {...(fieldState.error?{help:fieldState.error.message,validateStatus:'error' as const}:{})}>{node(field)}</Form.Item>}/>} 
function EditForm({control,t}:{control:ReturnType<typeof useForm<EditValues>>['control'];t:TFunction}){return <>{controlledField({control,name:'display_name',t,node:(field)=><Input {...field} aria-label={t('system.users.fields.display_name')} value={field.value as string}/>})}{controlledField({control,name:'locale',t,node:(field)=><Select {...field} aria-label={t('system.users.fields.locale')} options={['zh-CN','en-US'].map((value)=>({label:t(`common.locales.${value}`),value}))}/>})}{controlledField({control,name:'time_zone',t,node:(field)=><Input {...field} aria-label={t('system.users.fields.time_zone')} value={field.value as string}/>})}</>}
function CreateForm({control,t}:{control:ReturnType<typeof useForm<CreateValues>>['control'];t:TFunction}){return <>{controlledField({control,name:'email',t,node:(field)=><Input {...field} aria-label={t('system.users.fields.email')} autoComplete="off" type="email" value={field.value as string}/>})}{controlledField({control,name:'display_name',t,node:(field)=><Input {...field} aria-label={t('system.users.fields.display_name')} value={field.value as string}/>})}{controlledField({control,name:'locale',t,node:(field)=><Select {...field} aria-label={t('system.users.fields.locale')} options={['zh-CN','en-US'].map((value)=>({label:t(`common.locales.${value}`),value}))}/>})}{controlledField({control,name:'time_zone',t,node:(field)=><Input {...field} aria-label={t('system.users.fields.time_zone')} value={field.value as string}/>})}{controlledField({control,name:'temporary_password',t,node:(field)=><Input.Password {...field} aria-label={t('system.users.fields.temporary_password')} autoComplete="new-password" value={field.value as string}/>})}</>}
