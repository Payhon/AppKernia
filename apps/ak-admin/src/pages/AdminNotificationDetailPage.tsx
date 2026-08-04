import { Link } from "@tanstack/react-router";
import { Alert, Button, Card, Descriptions, Grid, Tag, Typography } from "antd";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useAuthStore } from "../features/auth/store";
import { type NotificationKind, useNotificationMessage, useNotificationRecipientStats } from "../features/notifications/hooks";
import { sanitizeNotificationHtml } from "../features/notifications/sanitize";

export function AdminNotificationDetailPage({ kind, id }: { kind: NotificationKind; id: string }) {
  const { t, i18n } = useTranslation();
  const screens = Grid.useBreakpoint();
  const permissions = new Set(useAuthStore((state)=>state.context?.permissions??[]));
  const message = useNotificationMessage(kind, id);
  const stats = useNotificationRecipientStats(kind, id, permissions.has("notify.recipient.read"));
  const formatter=useMemo(()=>new Intl.DateTimeFormat(i18n.language,{dateStyle:"long",timeStyle:"short"}),[i18n.language]);
  const back=kind==="notices"?"/system/notifications/notices":"/system/notifications/messages";
  if(message.isPending)return <div className="ak-centered-state" aria-live="polite"><span className="ak-loading-indicator"/><span>{t("common.states.loading")}</span></div>;
  if(message.isError)return <div className="ak-page-container"><div className="ak-form-error" role="alert">{t("notifications.feedback.load_error")} <Button onClick={()=>void message.refetch()}>{t("common.actions.retry")}</Button></div></div>;
  const current=message.data;
  return <div className="ak-page-container"><Link className="ak-back-link" to={back}>{t("common.actions.back")}</Link><header className="ak-page-heading"><div><Typography.Title level={1}>{current.title}</Typography.Title><Typography.Paragraph type="secondary">{formatter.format(new Date(current.created_at))}</Typography.Paragraph></div><Tag className={current.status==="published"?"ak-status-success":current.status==="cancelled"?"ak-status-error":"ak-status-warning"}>{t(`notifications.status.${current.status}`)}</Tag></header>{current.body_format==="html"?<Alert showIcon type="info" title={t("notifications.detail.sanitized")}/>:null}<Card><Descriptions bordered column={{xs:1,sm:2}} items={[{key:"type",label:t("notifications.columns.type"),children:t(`notifications.type.${current.message_type}`)},{key:"format",label:t("notifications.editor.format"),children:t(`notifications.format.${current.body_format}`)},{key:"audience",label:t("notifications.columns.audience"),children:t(`notifications.audience.${current.audience_scope}`,{count:current.audience_user_ids.length})},{key:"scheduled",label:t("notifications.editor.scheduled_at"),children:current.scheduled_at?formatter.format(new Date(current.scheduled_at)):"—"}]}/></Card><Typography.Title level={2}>{t("notifications.editor.preview")}</Typography.Title><Card><div className="ak-notification-preview">{current.body_format==="html"?<div dangerouslySetInnerHTML={{__html:sanitizeNotificationHtml(current.body)}}/>:<pre>{current.body}</pre>}</div></Card>{permissions.has("notify.recipient.read")?<><Typography.Title level={2}>{t("notifications.detail.recipient_stats")}</Typography.Title>{stats.isError?<Alert type="error" showIcon title={t("notifications.feedback.load_error")}/>:<Card loading={stats.isPending}><Descriptions column={screens.md?5:2} items={Object.entries(stats.data??{}).map(([key,value])=>({key,label:t(`notifications.stats.${key}`),children:value}))}/></Card>}</>:null}</div>;
}
