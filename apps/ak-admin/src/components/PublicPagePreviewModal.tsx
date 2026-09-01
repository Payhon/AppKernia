import { CloseOutlined, CopyOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Input, Space, Spin, Tooltip, Typography } from "antd";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { publicPageURL, previewChannel, previewMessage } from "../shared/public-page";
import { AkModal } from "./AkModal";
import { usePublicPageCopy } from "./usePublicPageCopy";

interface Props { open: boolean; url: string | undefined; title: string; onClose: () => void }

export function PublicPagePreviewModal({ open, url, title, onClose }: Props) {
  const { i18n } = useTranslation();
  const href = publicPageURL(url, i18n.language);
  return open && href ? <PreviewDialog key={href} href={href} title={title} onClose={onClose} /> : null;
}

function PreviewDialog({ href, title, onClose }: { href: string; title: string; onClose: () => void }) {
  const { t } = useTranslation();
  const frame = useRef<HTMLIFrameElement>(null);
  const [stageNode, setStageNode] = useState<HTMLDivElement | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const loadId = useRef(crypto.randomUUID());
  const closeRef = useRef(onClose);
  closeRef.current = onClose;
  const [revision, setRevision] = useState(0);
  const [scale, setScale] = useState(1);
  const [state, setState] = useState<"loading" | "ready" | "unavailable" | "timeout">("loading");
  const { result, copy } = usePublicPageCopy();
  const origin = new URL(href).origin;

  useEffect(() => {
    const node = stageNode;
    if (!node) return;
    const resize = () => {
      setScale(Math.max(0.1, Math.min(1, node.clientWidth / 413, node.clientHeight / 872)));
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(node);
    return () => {
      observer.disconnect();
    };
  }, [stageNode]);

  useEffect(() => {
    const receive = (event: MessageEvent<unknown>) => {
      const type = previewMessage(event, frame.current?.contentWindow ?? null, origin, loadId.current);
      if (!type) return;
      if (type === "close") { closeRef.current(); return; }
      clearTimeout(timer.current);
      setState(type);
    };
    window.addEventListener("message", receive);
    timer.current = setTimeout(() => { setState("timeout"); }, 10000);
    return () => { window.removeEventListener("message", receive); clearTimeout(timer.current); };
  }, [origin, revision]);

  const loaded = () => {
    clearTimeout(timer.current);
    loadId.current = crypto.randomUUID();
    setState("loading");
    timer.current = setTimeout(() => { setState("timeout"); }, 10000);
    frame.current?.contentWindow?.postMessage({ channel: previewChannel, type: "init", loadId: loadId.current }, origin);
  };
  const refresh = () => {
    loadId.current = crypto.randomUUID();
    setState("loading");
    setRevision(value => value + 1);
  };

  return <AkModal open centered footer={null} closable={false} mask={{ closable: false }} onCancel={onClose}
    className="ak-public-preview-modal" width="min(520px, calc(100vw - 24px))"
    title={<div className="ak-public-preview-toolbar"><span className="ak-public-preview-title">{title}</span><Space size={0}>
      <Tooltip title={t("apps.public_web.preview_refresh")}><Button type="text" icon={<ReloadOutlined />} aria-label={t("apps.public_web.preview_refresh")} onClick={refresh} /></Tooltip>
      <Tooltip title={t("apps.public_web.copy")}><Button type="text" icon={<CopyOutlined />} aria-label={t("apps.public_web.copy")} onClick={() => { void copy(href); }} /></Tooltip>
      <Tooltip title={t("apps.public_web.preview_close")}><Button type="text" icon={<CloseOutlined />} aria-label={t("apps.public_web.preview_close")} onClick={onClose} /></Tooltip>
    </Space></div>}
  >
    <Typography.Paragraph className="ak-public-preview-hint" type="secondary">{t("apps.public_web.preview_hint")}</Typography.Paragraph>
    {result?.url === href ? <div role="status" className="ak-public-preview-copy"><Typography.Text>{t(result.ok ? "apps.public_web.copied" : "apps.public_web.copy_error")}</Typography.Text>{!result.ok ? <Input readOnly value={href} aria-label={t("apps.public_web.copy")} onFocus={event => { event.target.select(); }} /> : null}</div> : null}
    {state === "timeout" ? <Alert type="warning" showIcon title={t("apps.public_web.preview_timeout")} action={<Space wrap><Button size="small" onClick={refresh}>{t("common.actions.retry")}</Button><Button size="small" href={href} target="_blank" rel="noopener noreferrer">{t("apps.public_web.view")}</Button></Space>} /> : null}
    <div ref={setStageNode} className="ak-public-preview-stage">
      <div className="ak-phone-space" style={{ width: 413 * scale, height: 872 * scale }}>
        <div className="ak-phone-device" style={{ transform: `scale(${String(scale)})` }}>
          <div className="ak-phone-side ak-phone-side-left" aria-hidden="true" /><div className="ak-phone-side ak-phone-side-right" aria-hidden="true" />
          <div className="ak-phone-screen">
            <div className="ak-phone-status" aria-hidden="true"><div className="ak-phone-island" /></div>
            <div className="ak-phone-content" aria-busy={state === "loading"}>
              <iframe key={revision} ref={frame} src={href} title={t("apps.public_web.preview_frame", { name: title })} onLoad={loaded}
                sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox"
                allow="camera 'none'; microphone 'none'; geolocation 'none'" referrerPolicy="no-referrer" />
              {state === "loading" ? <div className="ak-public-preview-loading" role="status"><Spin /><span>{t("apps.public_web.preview_loading")}</span></div> : null}
            </div>
            <div className="ak-phone-bottom" aria-hidden="true"><span /></div>
          </div>
        </div>
      </div>
    </div>
  </AkModal>;
}
