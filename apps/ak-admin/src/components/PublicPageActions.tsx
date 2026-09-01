import { MobileOutlined } from "@ant-design/icons";
import { Button, Input, Space, Typography } from "antd";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useTenantKey } from "../features/tenants/hooks";
import { publicPageURL } from "../shared/public-page";
import { PublicPagePreviewModal } from "./PublicPagePreviewModal";
import { usePublicPageCopy } from "./usePublicPageCopy";

export function PublicPageActions({ url, title, onPreview }: { url: string | undefined; title?: string; onPreview?: (url: string, title: string) => void }) {
  const { t, i18n } = useTranslation();
  const tenant = useTenantKey();
  const href = publicPageURL(url, i18n.language);
  return href ? <PublicPageControls key={`${tenant}:${href}`} href={href} title={title ?? t("apps.public_web.preview")} {...(onPreview ? { onPreview } : {})} /> : null;
}

function PublicPageControls({ href, title, onPreview }: { href: string; title: string; onPreview?: (url: string, title: string) => void }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const { result, copy } = usePublicPageCopy();
  return <><Space wrap>
    <Button ref={trigger} data-public-preview-url={href} size="small" icon={<MobileOutlined aria-hidden />} onClick={() => { if (onPreview) onPreview(href, title); else setOpen(true); }}>{t("apps.public_web.preview")}</Button>
    <Button size="small" href={href} target="_blank" rel="noopener noreferrer">{t("apps.public_web.view")}</Button>
    <Button size="small" onClick={() => { void copy(href); }}>{t("apps.public_web.copy")}</Button>
    {result ? <Typography.Text role="status">{t(result.ok ? "apps.public_web.copied" : "apps.public_web.copy_error")}</Typography.Text> : null}
    {result && !result.ok ? <Input readOnly value={href} aria-label={t("apps.public_web.copy")} onFocus={event => { event.target.select(); }} /> : null}
  </Space><PublicPagePreviewModal open={open} url={href} title={title} onClose={() => { setOpen(false); requestAnimationFrame(() => trigger.current?.focus()); }} /></>;
}
