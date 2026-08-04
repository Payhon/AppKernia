import {
  Alert,
  Button,
  Card,
  Descriptions,
  Drawer,
  Grid,
  Input,
  Modal,
  Progress,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  type TableColumnsType,
} from "antd";
import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import type { AdminFile } from "../generated/api/types.gen";
import { useAuthStore } from "../features/auth/store";
import { authSession } from "../features/auth/store";
import {
  useAdminFileMutations,
  useAdminFiles,
  useAdminFileUsages,
} from "../features/files/hooks";

interface Filters { q: string; status: string; scan_status: string; media_type: string; page: number; page_size: number }
type UploadState = "idle" | "uploading" | "paused" | "error" | "completed" | "cancelled";
interface UploadTask { file: File; sessionId?: string; progress: number; state: UploadState }
const readFilters = (): Filters => { const p = new URLSearchParams(location.search); return { q:p.get("q")??"",status:p.get("status")??"",scan_status:p.get("scan_status")??"",media_type:p.get("media_type")??"",page:Number(p.get("page")??1),page_size:Number(p.get("page_size")??20) } };
const saveFilters = (f: Filters) => { const p=new URLSearchParams();Object.entries(f).forEach(([k,v])=>{if(v!==""&&!(k==="page"&&v===1)&&!(k==="page_size"&&v===20))p.set(k,String(v))});history.replaceState(history.state,"",`${location.pathname}${p.size?`?${p}`:""}`) };
const safe = (file: AdminFile) => file.status === "ready" && ["clean", "skipped"].includes(file.scan_status);
const sizeLabel = (bytes: number) => bytes < 1024 ? `${String(bytes)} B` : bytes < 1024*1024 ? `${(bytes/1024).toFixed(1)} KiB` : `${(bytes/1024/1024).toFixed(1)} MiB`;

export function AdminFilesPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(useAuthStore((state) => state.context?.permissions ?? []));
  const [filters, setState] = useState(readFilters);
  const setFilters = (next: Filters) => { setState(next); saveFilters(next) };
  const files = useAdminFiles(filters);
  const mutations = useAdminFileMutations();
  const [detail, setDetail] = useState<AdminFile | null>(null);
  const usages = useAdminFileUsages(detail?.id ?? null);
  const [task, setTask] = useState<UploadTask | null>(null);
  const [feedback, setFeedback] = useState<{ key: string; error?: boolean } | null>(null);
  const controller = useRef<AbortController | null>(null);
  const fileInput = useRef<HTMLInputElement | null>(null);
  const formatter = useMemo(() => new Intl.DateTimeFormat(i18n.language,{dateStyle:"medium",timeStyle:"short"}),[i18n.language]);
  const startUpload = async (current: UploadTask) => {
    controller.current = new AbortController();
    setTask({ ...current, state: "uploading" }); setFeedback(null);
    try {
      await authSession.uploadAdminFile(current.file,{...(current.sessionId?{sessionId:current.sessionId}:{}),signal:controller.current.signal,onSession:(session)=>{ setTask((value)=>value?{...value,sessionId:session.id}:value); },onProgress:(progress)=>{ setTask((value)=>value?{...value,progress}:value); }});
      setTask((value)=>value?{...value,state:"completed",progress:100}:value);setFeedback({key:"system.files.upload.completed"});await files.refetch();
    } catch {
      if (controller.current.signal.aborted) setTask((value)=>value?{...value,state:"paused"}:value);
      else { setTask((value)=>value?{...value,state:"error"}:value);setFeedback({key:"system.files.upload.error",error:true}) }
    }
  };
  const chooseFile = (file: File | undefined) => { if(!file)return;if(file.size<=0||file.size>100*1024*1024){setFeedback({key:"system.files.upload.invalid",error:true});return};const next={file,progress:0,state:"idle" as const};setTask(next);void startUpload(next) };
  const cancelUpload = async () => { if(!task?.sessionId)return;controller.current?.abort();try{await authSession.cancelAdminFileUpload(task.sessionId);setTask((v)=>v?{...v,state:"cancelled"}:v);setFeedback({key:"system.files.upload.cancelled"})}catch{setFeedback({key:"system.files.upload.error",error:true})} };
  const download = async (file: AdminFile) => { try { const result=await authSession.downloadAdminFile(file.id);const url=URL.createObjectURL(result.blob);const anchor=document.createElement("a");anchor.href=url;anchor.download=result.file.original_name;anchor.click();URL.revokeObjectURL(url);setFeedback({key:"system.files.feedback.downloaded"}) } catch { setFeedback({key:"system.files.load_error",error:true}) } };
  const remove = async (file: AdminFile) => { try { const currentUsages=await authSession.adminFileUsages(file.id);if(currentUsages.items.length>0){Modal.warning({title:t("system.files.delete.title"),content:t("system.files.delete.in_use",{count:currentUsages.items.length}),okButtonProps:{type:"default"}});return}Modal.confirm({title:t("system.files.delete.title"),content:t("system.files.delete.impact"),okButtonProps:{danger:true},okText:t("common.actions.delete"),cancelText:t("common.actions.cancel"),onOk:async()=>{await mutations.remove.mutateAsync(file.id);setFeedback({key:"system.files.feedback.deleted"})}}) } catch { setFeedback({key:"system.files.load_error",error:true}) } };
  const columns: TableColumnsType<AdminFile> = [
    {title:t("system.files.columns.file"),key:"file",render:(_,x)=><div><strong>{x.original_name}</strong><div className="ak-table-secondary">{x.media_type}</div></div>},
    {title:t("system.files.columns.size"),dataIndex:"size_bytes",render:(v:number)=>sizeLabel(v),responsive:["sm"]},
    {title:t("system.files.columns.status"),dataIndex:"status",render:(v:AdminFile["status"])=><Tag className={v==="ready"?"ak-status-success":v==="quarantined"?"ak-status-error":"ak-status-warning"}>{t(`system.files.status.${v}`)}</Tag>},
    {title:t("system.files.columns.scan"),dataIndex:"scan_status",render:(v:AdminFile["scan_status"])=><Tag className={["clean","skipped"].includes(v)?"ak-status-success":v==="infected"?"ak-status-error":"ak-status-warning"}>{t(`system.files.scan.${v}`)}</Tag>},
    {title:t("system.files.columns.usages"),dataIndex:"usage_count",responsive:["md"]},
    {title:t("system.files.columns.created"),dataIndex:"created_at",render:(v:string)=>formatter.format(new Date(v)),responsive:["lg"]},
    {title:t("system.files.columns.actions"),key:"actions",width:screens.md?300:140,render:(_,file)=><Space wrap><Button size="small" onClick={()=>{ void navigate({to:"/system/storage/files/$fileId",params:{fileId:file.id}}); }}>{t("system.files.actions.details")}</Button>{permissions.has("storage.file.download")&&safe(file)?<Button size="small" onClick={()=>void download(file)}>{t("system.files.actions.download")}</Button>:null}{permissions.has("storage.file.delete")?<Button danger size="small" onClick={()=>void remove(file)}>{t("common.actions.delete")}</Button>:null}</Space>},
  ];
  return <div className="ak-page-container"><header className="ak-page-heading"><div><Typography.Title level={1}>{t("system.files.title")}</Typography.Title><Typography.Paragraph type="secondary">{t("system.files.description")}</Typography.Paragraph></div>{permissions.has("storage.file.upload")?<Button type="primary" onClick={()=>fileInput.current?.click()}>{t("system.files.actions.upload")}</Button>:null}</header><input ref={fileInput} className="ak-sr-only" type="file" aria-label={t("system.files.actions.choose")} onChange={(e)=>{ chooseFile(e.target.files?.[0]); }}/>{task?<Card className="ak-file-upload-card" title={t("system.files.upload.title")}><Typography.Paragraph>{task.file.name} · {sizeLabel(task.file.size)}</Typography.Paragraph><Progress aria-label={t("system.files.upload.progress",{percent:task.progress})} percent={task.progress}/><div aria-live="polite">{t(task.state==="paused"?"system.files.upload.paused":"system.files.upload.progress",{percent:task.progress})}</div><Space>{task.state==="uploading"?<Button onClick={()=>controller.current?.abort()}>{t("system.files.actions.pause")}</Button>:null}{["paused","error"].includes(task.state)?<Button type="primary" onClick={()=>void startUpload(task)}>{t("system.files.actions.resume")}</Button>:null}{task.sessionId&&!(["completed","cancelled"].includes(task.state))?<Button danger onClick={()=>void cancelUpload()}>{t("system.files.actions.cancel_upload")}</Button>:null}</Space></Card>:null}{feedback?<div className={feedback.error?"ak-form-error":"ak-org-feedback"} role={feedback.error?"alert":"status"}>{t(feedback.key)}</div>:null}<Card><div className="ak-settings-filters" role="search"><Input.Search allowClear aria-label={t("system.files.filters.query")} value={filters.q} onChange={(e)=>{ setFilters({...filters,q:e.target.value,page:1}); }}/><Select allowClear aria-label={t("system.files.filters.status")} placeholder={t("system.files.filters.status")} value={filters.status||undefined} onChange={(v)=>{ setFilters({...filters,status:v??"",page:1}); }} options={["pending","ready","quarantined"].map(v=>({value:v,label:t(`system.files.status.${v}`)}))}/><Select allowClear aria-label={t("system.files.filters.scan")} placeholder={t("system.files.filters.scan")} value={filters.scan_status||undefined} onChange={(v)=>{ setFilters({...filters,scan_status:v??"",page:1}); }} options={["pending","clean","infected","failed","skipped"].map(v=>({value:v,label:t(`system.files.scan.${v}`)}))}/><Input aria-label={t("system.files.filters.media")} placeholder={t("system.files.filters.media")} value={filters.media_type} onChange={(e)=>{ setFilters({...filters,media_type:e.target.value,page:1}); }}/></div>{files.isError?<Alert type="error" showIcon title={t("system.files.load_error")} action={<Button onClick={()=>void files.refetch()}>{t("common.actions.retry")}</Button>}/>:null}<div className="ak-table-scroll"><Table columns={columns} dataSource={files.data?.items??[]} loading={files.isPending} locale={{emptyText:t("system.files.empty")}} pagination={{current:filters.page,pageSize:filters.page_size,total:files.data?.total??0,showSizeChanger:true,onChange:(page,page_size)=>{ setFilters({...filters,page,page_size}); }}} rowKey="id" {...((files.data?.items.length??0)>0?{scroll:{x:screens.md?1040:620}}:{})}/></div></Card><Drawer destroyOnHidden open={Boolean(detail)} onClose={()=>{ setDetail(null); }} size="large" title={t("system.files.detail.title")}><Alert showIcon type="info" title={t("system.files.detail.scan_gate")}/>{detail?<Descriptions column={1} bordered items={[{key:"name",label:t("system.files.columns.file"),children:detail.original_name},{key:"type",label:t("system.files.filters.media"),children:detail.media_type},{key:"size",label:t("system.files.columns.size"),children:sizeLabel(detail.size_bytes)},{key:"status",label:t("system.files.columns.status"),children:t(`system.files.status.${detail.status}`)},{key:"scan",label:t("system.files.columns.scan"),children:t(`system.files.scan.${detail.scan_status}`)}]}/>:null}<Typography.Title level={2}>{t("system.files.detail.usages")}</Typography.Title><Table columns={[{title:t("system.files.detail.usages"),key:"usage",render:(_,x)=><span>{x.module_code} / {x.entity_type} / {x.field_name}</span>}]} dataSource={usages.data?.items??[]} loading={usages.isPending} locale={{emptyText:t("system.files.detail.no_usages")}} pagination={false} rowKey="id"/></Drawer></div>;
}
