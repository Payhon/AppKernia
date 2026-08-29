import { AppstoreOutlined, CloseOutlined, DownOutlined, FileImageOutlined, FileOutlined, FullscreenExitOutlined, FullscreenOutlined, HolderOutlined, LeftOutlined, PictureOutlined, TableOutlined, VideoCameraOutlined } from "@ant-design/icons";
import { Button, Card, Col, DatePicker, Dropdown, Empty, Image, Input, Row, Select, Spin, Splitter, Table, Tag, Tooltip, Typography, type MenuProps } from "antd";
import type { Dayjs } from "dayjs";
import type { KeyboardEvent as ReactKeyboardEvent, PointerEvent as ReactPointerEvent } from "react";
import { useDeferredValue, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { AdminFile } from "../generated/api/types.gen";
import { authSession, useAuthStore } from "../features/auth/store";
import { useAdminFiles } from "../features/files/hooks";
import { AkFileUploader } from "./AkFileUploader";
import { AkModal } from "./AkModal";

type FileKind = "all" | "image" | "video";
type PickerFilter = FileKind | "other";
type FileView = "grid" | "table" | "thumbnail";
type ThumbnailVariant = "grid" | "table" | "thumbnail";
interface AkFilePickerProps { open: boolean; onClose: () => void; onSelect: (file: AdminFile) => void | Promise<void>; kind?: FileKind; }
const selectable = (file: AdminFile) => file.status === "ready" && ["clean", "skipped"].includes(file.scan_status);
const matchesFilter = (file: AdminFile, filter: PickerFilter) => filter === "all" || (filter === "image" && file.media_type.startsWith("image/")) || (filter === "video" && file.media_type.startsWith("video/")) || (filter === "other" && !file.media_type.startsWith("image/") && !file.media_type.startsWith("video/"));
const sizeLabel = (bytes: number) => bytes < 1024 ? `${String(bytes)} B` : bytes < 1024 * 1024 ? `${(bytes / 1024).toFixed(1)} KiB` : `${(bytes / 1024 / 1024).toFixed(1)} MiB`;

interface ViewportSize { width: number; height: number }
interface DialogRect extends ViewportSize { x: number; y: number }
interface PointerSession { pointerId: number; startX: number; startY: number; rect: DialogRect }

const DIALOG_MARGIN = 8;
const MIN_DIALOG_WIDTH = 800;
const MIN_DIALOG_HEIGHT = 520;
const getViewportSize = (): ViewportSize => typeof window === "undefined" ? { width: 1280, height: 800 } : { width: window.innerWidth, height: window.innerHeight };
const clampNumber = (value: number, minimum: number, maximum: number) => Math.min(Math.max(value, minimum), maximum);
const clampDialogRect = (rect: DialogRect, viewport: ViewportSize): DialogRect => {
  const maximumWidth = Math.max(280, viewport.width - DIALOG_MARGIN * 2);
  const maximumHeight = Math.max(320, viewport.height - DIALOG_MARGIN * 2);
  const width = clampNumber(rect.width, Math.min(MIN_DIALOG_WIDTH, maximumWidth), maximumWidth);
  const height = clampNumber(rect.height, Math.min(MIN_DIALOG_HEIGHT, maximumHeight), maximumHeight);
  return {
    width,
    height,
    x: clampNumber(rect.x, DIALOG_MARGIN, Math.max(DIALOG_MARGIN, viewport.width - width - DIALOG_MARGIN)),
    y: clampNumber(rect.y, DIALOG_MARGIN, Math.max(DIALOG_MARGIN, viewport.height - height - DIALOG_MARGIN)),
  };
};
const defaultDialogRect = (viewport: ViewportSize): DialogRect => {
  const width = Math.min(1120, viewport.width - DIALOG_MARGIN * 2);
  const height = Math.min(720, viewport.height - DIALOG_MARGIN * 2);
  return clampDialogRect({ width, height, x: (viewport.width - width) / 2, y: (viewport.height - height) / 2 }, viewport);
};

export function AkFilePicker({ open, onClose, onSelect, kind = "all" }: AkFilePickerProps) {
  const { t, i18n } = useTranslation();
  const [viewport, setViewport] = useState<ViewportSize>(getViewportSize);
  const [dialogRect, setDialogRect] = useState<DialogRect>(() => defaultDialogRect(getViewportSize()));
  const [maximized, setMaximized] = useState(false);
  const [previewOpen, setPreviewOpen] = useState(true);
  const [view, setView] = useState<FileView>("thumbnail");
  const [q, setQ] = useState("");
  const deferredQuery = useDeferredValue(q);
  const [filter, setFilter] = useState<PickerFilter>(kind);
  const [createdRange, setCreatedRange] = useState<[Dayjs | null, Dayjs | null] | null>(null);
  const [selected, setSelected] = useState<AdminFile | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewError, setPreviewError] = useState(false);
  const previewRef = useRef<string | null>(null);
  const restoreRectRef = useRef<DialogRect | null>(null);
  const dragSessionRef = useRef<PointerSession | null>(null);
  const canUpload = useAuthStore((state) => state.context?.permissions.includes("storage.file.upload") ?? false);
  const effectiveFilter = kind !== "all" && filter === "all" ? kind : filter;
  const files = useAdminFiles({ q: deferredQuery, status: "ready", media_type: effectiveFilter === "image" || effectiveFilter === "video" ? `${effectiveFilter}/` : "", created_from: createdRange?.[0]?.startOf("day").toISOString() ?? "", created_to: createdRange?.[1]?.endOf("day").toISOString() ?? "", page: 1, page_size: 80 });
  const visibleFiles = (files.data?.items ?? []).filter((file) => matchesFilter(file, effectiveFilter));
  const dateFormatter = new Intl.DateTimeFormat(i18n.language, { dateStyle: "short", timeStyle: "short" });
  const compactLayout = viewport.width < 900;
  const activeRect = maximized ? { x: DIALOG_MARGIN, y: DIALOG_MARGIN, width: viewport.width - DIALOG_MARGIN * 2, height: viewport.height - DIALOG_MARGIN * 2 } : dialogRect;
  const tableScrollY = compactLayout ? Math.max(160, Math.floor((activeRect.height - 300) * 0.6)) : Math.max(220, activeRect.height - 330);
  useEffect(() => {
    const handleResize = () => {
      const nextViewport = getViewportSize();
      setViewport(nextViewport);
      setDialogRect((current) => clampDialogRect(current, nextViewport));
      if (restoreRectRef.current) restoreRectRef.current = clampDialogRect(restoreRectRef.current, nextViewport);
    };
    window.addEventListener("resize", handleResize);
    return () => { window.removeEventListener("resize", handleResize); };
  }, []);
  useEffect(() => {
    if (open) {
      const nextViewport = getViewportSize();
      setViewport(nextViewport);
      setDialogRect(defaultDialogRect(nextViewport));
      setMaximized(false);
      setPreviewOpen(true);
      setView("thumbnail");
      restoreRectRef.current = null;
      setFilter(kind);
      setSelected(null);
      setQ("");
      setCreatedRange(null);
    }
  }, [kind, open]);
  useEffect(() => {
    let cancelled = false;
    setPreviewError(false);
    if (!previewOpen || !selected || (!selected.media_type.startsWith("image/") && !selected.media_type.startsWith("video/"))) { if (previewRef.current) URL.revokeObjectURL(previewRef.current); previewRef.current = null; setPreviewUrl(null); return; }
    void authSession.downloadAdminFile(selected.id).then(({ blob }) => { if (cancelled) return; const url = URL.createObjectURL(blob); if (previewRef.current) URL.revokeObjectURL(previewRef.current); previewRef.current = url; setPreviewUrl(url); }).catch(() => { if (!cancelled) { setPreviewUrl(null); setPreviewError(true); } });
    return () => { cancelled = true; };
  }, [previewOpen, selected]);
  useEffect(() => () => { if (previewRef.current) URL.revokeObjectURL(previewRef.current); }, []);
  useEffect(() => { if (!open && previewRef.current) { URL.revokeObjectURL(previewRef.current); previewRef.current = null; setPreviewUrl(null); } }, [open]);
  const preview = previewError ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t("system.files.picker.preview.error")} /> : selected?.media_type.startsWith("image/") && previewUrl ? <Image className="ak-file-picker-preview-media" src={previewUrl} alt={selected.original_name} /> : selected?.media_type.startsWith("video/") && previewUrl ? <video className="ak-file-picker-preview-media" controls src={previewUrl} /> : selected ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t("system.files.picker.preview.unsupported")} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t("system.files.picker.preview.empty")} />;
  const uploader = canUpload ? effectiveFilter === "image" || effectiveFilter === "video" ? <AkFileUploader compact mediaTypePrefix={`${effectiveFilter}/`} onUploaded={async (file) => { setSelected(file); await files.refetch(); }} /> : <AkFileUploader compact onUploaded={async (file) => { setSelected(file); await files.refetch(); }} /> : null;
  const moveDialog = (x: number, y: number) => {
    setDialogRect((current) => clampDialogRect({ ...current, x, y }, viewport));
  };
  const startDrag = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (maximized || compactLayout || event.button !== 0 || (event.target as HTMLElement).closest("button")) return;
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    dragSessionRef.current = { pointerId: event.pointerId, startX: event.clientX, startY: event.clientY, rect: dialogRect };
  };
  const dragDialog = (event: ReactPointerEvent<HTMLDivElement>) => {
    const session = dragSessionRef.current;
    if (session?.pointerId !== event.pointerId) return;
    moveDialog(session.rect.x + event.clientX - session.startX, session.rect.y + event.clientY - session.startY);
  };
  const endDrag = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (dragSessionRef.current?.pointerId !== event.pointerId) return;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
    dragSessionRef.current = null;
  };
  const moveDialogWithKeyboard = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (maximized || compactLayout || !event.altKey) return;
    const step = event.shiftKey ? 32 : 8;
    let x = dialogRect.x;
    let y = dialogRect.y;
    if (event.key === "ArrowLeft") x -= step;
    else if (event.key === "ArrowRight") x += step;
    else if (event.key === "ArrowUp") y -= step;
    else if (event.key === "ArrowDown") y += step;
    else return;
    event.preventDefault();
    moveDialog(x, y);
  };
  const toggleMaximized = () => {
    if (maximized) {
      setDialogRect(clampDialogRect(restoreRectRef.current ?? defaultDialogRect(viewport), viewport));
      setMaximized(false);
    } else {
      restoreRectRef.current = dialogRect;
      setMaximized(true);
    }
  };
  const rowSelection = { type: "radio" as const, selectedRowKeys: selected ? [selected.id] : [], getCheckboxProps: (file: AdminFile) => ({ disabled: !selectable(file) }), onSelect: (file: AdminFile) => { setSelected(file); } };
  const onFileRow = (file: AdminFile) => ({
    tabIndex: selectable(file) ? 0 : -1,
    "aria-selected": selected?.id === file.id,
    onClick: () => { if (selectable(file)) setSelected(file); },
    onKeyDown: (event: ReactKeyboardEvent<HTMLTableRowElement>) => {
      if (selectable(file) && (event.key === "Enter" || event.key === " ")) {
        event.preventDefault();
        setSelected(file);
      }
    },
  });
  const commonColumns = [
    { title: t("system.files.columns.uploaded"), dataIndex: "created_at", width: 142, render: (value: string) => <span className="ak-file-picker-date">{dateFormatter.format(new Date(value))}</span> },
    { title: t("system.files.columns.size"), dataIndex: "size_bytes", width: 78, render: (value: number) => <span className="ak-file-picker-size">{sizeLabel(value)}</span> },
    { title: t("system.files.columns.scan"), dataIndex: "scan_status", width: 118, render: (value: AdminFile["scan_status"]) => <Tag className={["clean", "skipped"].includes(value) ? "ak-status-success" : "ak-status-warning"}>{t(`system.files.scan.${value}`)}</Tag> },
  ];
  const thumbnailTable = <Table
    size="small"
    className="ak-file-picker-table ak-file-picker-thumbnail-table"
    columns={[{ title: t("system.files.columns.file"), dataIndex: "original_name", width: 138, render: (name: string, file: AdminFile) => <FileIdentity file={file} name={name} variant="thumbnail" /> }, ...commonColumns]}
    dataSource={visibleFiles}
    loading={files.isPending}
    locale={{ emptyText: files.isError ? t("system.files.picker.preview.error") : t("system.files.empty") }}
    pagination={false}
    rowKey="id"
    rowClassName={(file) => selectable(file) ? "ak-file-picker-row-selectable" : "ak-file-picker-row-disabled"}
    onRow={onFileRow}
    rowSelection={rowSelection}
    scroll={{ x: 540, y: tableScrollY }}
  />;
  const compactTable = <Table
    size="small"
    className="ak-file-picker-table ak-file-picker-compact-table"
    columns={[{ title: t("system.files.columns.file"), dataIndex: "original_name", width: 210, render: (name: string, file: AdminFile) => <FileIdentity file={file} name={name} variant="table" /> }, ...commonColumns]}
    dataSource={visibleFiles}
    loading={files.isPending}
    locale={{ emptyText: files.isError ? t("system.files.picker.preview.error") : t("system.files.empty") }}
    pagination={false}
    rowKey="id"
    rowClassName={(file) => selectable(file) ? "ak-file-picker-row-selectable" : "ak-file-picker-row-disabled"}
    onRow={onFileRow}
    rowSelection={rowSelection}
    scroll={{ x: 610, y: tableScrollY }}
  />;
  const gridView = <div className="ak-file-picker-grid">
    {files.isPending ? <div className="ak-file-picker-grid-state"><Spin /></div> : files.isError || visibleFiles.length === 0 ? <div className="ak-file-picker-grid-state"><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={files.isError ? t("system.files.picker.preview.error") : t("system.files.empty")} /></div> : visibleFiles.map((file) => <button
      type="button"
      key={file.id}
      className={selected?.id === file.id ? "ak-file-picker-grid-item ak-file-picker-grid-item-selected" : "ak-file-picker-grid-item"}
      disabled={!selectable(file)}
      aria-pressed={selected?.id === file.id}
      onClick={() => { setSelected(file); }}
    >
      <FileIdentity file={file} name={file.original_name} variant="grid" />
      <span className="ak-file-picker-grid-meta">
        <span>{file.media_type}</span>
        <span>{dateFormatter.format(new Date(file.created_at))}</span>
        <span>{sizeLabel(file.size_bytes)} · {t(`system.files.scan.${file.scan_status}`)}</span>
      </span>
    </button>)}
  </div>;
  const fileView = view === "grid" ? gridView : view === "table" ? compactTable : thumbnailTable;
  const viewItems: MenuProps["items"] = [
    { key: "grid", icon: <AppstoreOutlined />, label: t("system.files.picker.view.grid") },
    { key: "table", icon: <TableOutlined />, label: t("system.files.picker.view.table") },
    { key: "thumbnail", icon: <PictureOutlined />, label: t("system.files.picker.view.thumbnail") },
  ];
  const viewIcon = view === "grid" ? <AppstoreOutlined /> : view === "table" ? <TableOutlined /> : <PictureOutlined />;
  const previewCard = <Card size="small" title={t("system.files.picker.preview.title")} extra={<Tooltip title={t("system.files.picker.preview.close")}><Button type="text" size="small" icon={<CloseOutlined />} aria-label={t("system.files.picker.preview.close")} onClick={() => { setPreviewOpen(false); }} /></Tooltip>} className="ak-file-picker-preview">{selected ? <><Typography.Text strong>{selected.original_name}</Typography.Text><Typography.Paragraph type="secondary">{selected.media_type} · {sizeLabel(selected.size_bytes)}</Typography.Paragraph>{previewUrl === null && !previewError && (selected.media_type.startsWith("image/") || selected.media_type.startsWith("video/")) ? <Spin /> : preview}</> : preview}</Card>;
  const modalTitle = <div className="ak-file-picker-titlebar" tabIndex={maximized || compactLayout ? -1 : 0} title={t("system.files.picker.window.drag")} onPointerDown={startDrag} onPointerMove={dragDialog} onPointerUp={endDrag} onPointerCancel={endDrag} onKeyDown={moveDialogWithKeyboard}><span className="ak-file-picker-title">{t("system.files.picker.title")}</span><div className="ak-file-picker-window-actions" onPointerDown={(event) => { event.stopPropagation(); }}><Tooltip title={t(maximized ? "system.files.picker.window.restore" : "system.files.picker.window.maximize")}><Button type="text" size="small" icon={maximized ? <FullscreenExitOutlined /> : <FullscreenOutlined />} aria-label={t(maximized ? "system.files.picker.window.restore" : "system.files.picker.window.maximize")} onClick={toggleMaximized} /></Tooltip><Tooltip title={t("common.actions.close")}><Button type="text" size="small" icon={<CloseOutlined />} aria-label={t("common.actions.close")} onClick={onClose} /></Tooltip></div></div>;
  return <AkModal
    destroyOnHidden
    closable={false}
    open={open}
    rootClassName={maximized ? "ak-file-picker-modal ak-file-picker-modal-maximized" : "ak-file-picker-modal"}
    style={{ position: "fixed", left: activeRect.x, top: activeRect.y, margin: 0, maxWidth: "none", paddingBottom: 0 }}
    styles={{ container: { height: "100%", display: "flex", flexDirection: "column" }, body: { flex: 1, minHeight: 0, overflow: "hidden" } }}
    resizable={{
      width: activeRect.width,
      height: activeRect.height,
      minWidth: Math.min(MIN_DIALOG_WIDTH, viewport.width - DIALOG_MARGIN * 2),
      minHeight: Math.min(MIN_DIALOG_HEIGHT, viewport.height - DIALOG_MARGIN * 2),
      viewportPadding: DIALOG_MARGIN,
      disabled: maximized || compactLayout,
      onResize: ({ width, height }) => { setDialogRect((current) => clampDialogRect({ ...current, width, height }, viewport)); },
    }}
    resizeHandleLabel={t("system.files.picker.window.resize")}
    modalRender={(node) => <div className="ak-file-picker-window">{node}</div>}
    title={modalTitle}
    onCancel={onClose}
    okText={t("system.files.actions.select")}
    okButtonProps={{ disabled: !selected || !selectable(selected) }}
    onOk={() => { if (selected) void onSelect(selected); }}
    footer={(_, { CancelBtn, OkBtn }) => <div className="ak-file-picker-footer"><div className="ak-file-picker-footer-upload">{uploader}</div><div className="ak-file-picker-footer-actions"><CancelBtn /><OkBtn /></div></div>}
  >
    <div className="ak-file-picker-content">
      <Typography.Paragraph className="ak-file-picker-description" type="secondary">{t("system.files.picker.description")}</Typography.Paragraph>
      <Row gutter={[12, 12]} align="middle" className="ak-file-picker-filters"><Col flex="1 1 220px"><Input.Search allowClear aria-label={t("system.files.filters.query")} value={q} onChange={(event) => { setQ(event.target.value); }} /></Col><Col flex="0 1 340px"><DatePicker.RangePicker className="ak-file-picker-date-range" allowClear allowEmpty={[true, true]} aria-label={t("system.files.filters.uploaded_range")} value={createdRange} placeholder={[t("system.files.filters.uploaded_from"), t("system.files.filters.uploaded_to")]} onChange={(value) => { setCreatedRange(value ? [value[0], value[1]] : null); }} /></Col><Col flex="0 1 170px"><Select aria-label={t("system.files.picker.type.label")} value={effectiveFilter} onChange={setFilter} style={{ width: "100%" }} options={["all", "image", "video", "other"].map((value) => ({ value, label: t(`system.files.picker.type.${value}`), disabled: kind !== "all" && value !== kind }))} /></Col><Col flex="0 0 auto"><Dropdown trigger={["click"]} menu={{ items: viewItems, selectable: true, selectedKeys: [view], onClick: ({ key }) => { setView(key as FileView); } }}><Tooltip title={t("system.files.picker.view.label")}><Button className="ak-file-picker-view-switcher" icon={viewIcon} aria-label={t("system.files.picker.view.label")}><DownOutlined /></Button></Tooltip></Dropdown></Col></Row>
      {previewOpen ? <Splitter key={compactLayout ? "compact" : "desktop"} className="ak-file-picker-browser" orientation={compactLayout ? "vertical" : "horizontal"} lazy draggerIcon={<Tooltip title={t("system.files.picker.window.split")}><HolderOutlined /></Tooltip>}>
        <Splitter.Panel min={compactLayout ? 180 : 420} defaultSize={compactLayout ? "60%" : "64%"}><div className="ak-file-picker-list-panel">{fileView}</div></Splitter.Panel>
        <Splitter.Panel min={compactLayout ? 140 : 280} defaultSize={compactLayout ? "40%" : "36%"}><div className="ak-file-picker-preview-panel">{previewCard}</div></Splitter.Panel>
      </Splitter> : <div className="ak-file-picker-browser ak-file-picker-browser-preview-closed"><div className="ak-file-picker-list-panel">{fileView}</div><Tooltip placement="left" title={t("system.files.picker.preview.open")}><Button type="text" size="small" shape="circle" className="ak-file-picker-preview-open" icon={<LeftOutlined />} aria-label={t("system.files.picker.preview.open")} onClick={() => { setPreviewOpen(true); }} /></Tooltip></div>}
      {selected ? <div className="ak-file-picker-selected" role="status">{t("system.files.picker.selected", { name: selected.original_name })}</div> : null}
      <Button className="ak-sr-only" onClick={onClose}>{t("common.actions.close")}</Button>
    </div>
  </AkModal>;
}
function FileIdentity({ file, name, variant }: { file: AdminFile; name: string; variant: ThumbnailVariant }) {
  const image = file.media_type.startsWith("image/");
  const video = file.media_type.startsWith("video/");
  const [thumbnailUrl, setThumbnailUrl] = useState<string | null>(null);
  const [thumbnailError, setThumbnailError] = useState(false);
  const containerRef = useRef<HTMLSpanElement | null>(null);
  useEffect(() => {
    if (!image) return;
    let active = true;
    let objectUrl: string | null = null;
    let started = false;
    const load = () => {
      if (started) return;
      started = true;
      void authSession.downloadAdminFile(file.id).then(({ blob }) => {
        if (!active) return;
        objectUrl = URL.createObjectURL(blob);
        setThumbnailUrl(objectUrl);
      }).catch(() => { if (active) setThumbnailError(true); });
    };
    let observer: IntersectionObserver | null = null;
    if (typeof IntersectionObserver === "undefined" || !containerRef.current) load();
    else {
      observer = new IntersectionObserver((entries) => {
        if (entries.some((entry) => entry.isIntersecting)) { observer?.disconnect(); load(); }
      }, { rootMargin: "120px" });
      observer.observe(containerRef.current);
    }
    return () => { active = false; observer?.disconnect(); if (objectUrl) URL.revokeObjectURL(objectUrl); };
  }, [file.id, image]);
  const placeholder = image ? <FileImageOutlined /> : video ? <VideoCameraOutlined /> : <FileOutlined />;
  return <span className={`ak-file-picker-file ak-file-picker-file-${variant}`}><span ref={containerRef} className={`ak-file-picker-thumb ak-file-picker-thumb-${variant}`}>{thumbnailUrl ? <img className="ak-file-picker-thumb-image" loading="lazy" decoding="async" src={thumbnailUrl} alt={name} /> : image && !thumbnailError ? <Spin size="small" /> : placeholder}</span><span className="ak-file-picker-filename" title={name}>{name}</span></span>;
}
