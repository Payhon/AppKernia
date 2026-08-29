import type { ICommand, TextAreaTextApi } from "@uiw/react-md-editor/commands";
import rehypeSanitize from "rehype-sanitize";
import { Button, Card, Drawer, Form, Image, Input, InputNumber, Select, Space, Switch, Tabs, Typography } from "antd";
import { lazy, Suspense, useCallback, useEffect, useRef, useState } from "react";
import { Controller, type useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { authSession } from "../features/auth/store";
import type { AdminFile } from "../generated/api/types.gen";
import type { ArticleInput, ContentCategory, ContentTag, ContentTopic } from "../features/content/model";
import { AkLocalizedFormTabs } from "./AkLocalizedFormTabs";
import type { AdminLocale } from "../shared/i18n";
import type { SystemLanguagesState } from "../features/settings/system-languages";

type PickerKind = "cover" | "video" | "gallery" | "inline" | null;
const MDEditor = lazy(async () => import("@uiw/react-md-editor/nohighlight"));
const AkFilePicker = lazy(async () => import("./AkFilePicker").then((module) => ({ default: module.AkFilePicker })));

export function ArticleDrawer({ open, fullScreen, form, categories, topics, tags, languages, activeLocale, onActiveLocaleChange, onClose, onSave }: { open: boolean; fullScreen: boolean; form: ReturnType<typeof useForm<ArticleInput>>; categories: ContentCategory[]; topics: ContentTopic[]; tags: ContentTag[]; languages: SystemLanguagesState; activeLocale: AdminLocale; onActiveLocaleChange: (x: AdminLocale) => void; onClose: () => void; onSave: () => void }) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language === "en-US" ? "en-US" : "zh-CN";
  const [picker, setPicker] = useState<PickerKind>(null);
  const [editorTab, setEditorTab] = useState("meta");
  const [previewUrls, setPreviewUrls] = useState<Record<string, string>>({});
  const [previewErrors, setPreviewErrors] = useState<Record<string, boolean>>({});
  const previewUrlsRef = useRef(previewUrls);
  const previewLoadsRef = useRef(new Set<string>());
  const previewSessionRef = useRef(0);
  const editorApi = useRef<TextAreaTextApi | null>(null);
  const type = form.watch("content_type");
  const media = form.watch("media");
  const videoSource = form.watch("video_source_type");
  const coverId = form.watch("cover_file_id");
  const videoFileId = form.watch("video_file_id");
  const externalVideo = form.watch("video_external_url");
  const errors = form.formState.errors;

  useEffect(() => { previewUrlsRef.current = previewUrls; }, [previewUrls]);
  useEffect(() => () => { Object.values(previewUrlsRef.current).forEach((url) => { URL.revokeObjectURL(url); }); }, []);
  useEffect(() => { if (!open) { previewSessionRef.current += 1; Object.values(previewUrlsRef.current).forEach((url) => { URL.revokeObjectURL(url); }); previewUrlsRef.current = {}; previewLoadsRef.current.clear(); setPreviewUrls({}); setPreviewErrors({}); editorApi.current = null; } }, [open]);
  useEffect(() => { if (errors.slug || errors.category_ids) setEditorTab("meta"); else if (errors.media || errors.video_source_type || errors.video_file_id || errors.video_external_url || errors.translations) setEditorTab("content"); }, [errors]);

  const loadPreview = useCallback(async (fileId: string) => {
    if (previewUrlsRef.current[fileId] || previewLoadsRef.current.has(fileId)) return;
    const previewSession = previewSessionRef.current;
    previewLoadsRef.current.add(fileId);
    setPreviewErrors((current) => ({ ...current, [fileId]: false }));
    try {
      const result = await authSession.downloadAdminFile(fileId);
      const url = URL.createObjectURL(result.blob);
      if (previewSession !== previewSessionRef.current) {
        URL.revokeObjectURL(url);
        return;
      }
      const next = { ...previewUrlsRef.current, [fileId]: url };
      previewUrlsRef.current = next;
      setPreviewUrls(next);
    } catch {
      if (previewSession === previewSessionRef.current) setPreviewErrors((current) => ({ ...current, [fileId]: true }));
    } finally {
      previewLoadsRef.current.delete(fileId);
    }
  }, []);
  useEffect(() => {
    if (!open) return;
    const fileIds = new Set([coverId, videoFileId, ...media.map((item) => item.file_id)].filter((fileId): fileId is string => Boolean(fileId)));
    fileIds.forEach((fileId) => { void loadPreview(fileId); });
  }, [coverId, loadPreview, media, open, videoFileId]);
  const selectFile = async (file: AdminFile) => {
    await loadPreview(file.id);
    if (picker === "cover") form.setValue("cover_file_id", file.id, { shouldDirty: true });
    if (picker === "video") { form.setValue("video_file_id", file.id, { shouldDirty: true }); form.setValue("video_source_type", "upload", { shouldDirty: true }); }
    if (picker === "gallery") form.setValue("media", [...media, { id: crypto.randomUUID(), file_id: file.id, role: "gallery", sort_order: media.length, translations: { "zh-CN": { alt_text: file.original_name }, "en-US": { alt_text: file.original_name } } }], { shouldDirty: true });
    if (picker === "inline" && !media.some((item) => item.file_id === file.id)) {
      const markdown = `![${file.original_name}](/api/v1/public/content/assets/${file.id})`;
      const bodyPath = `translations.${activeLocale}.body` as const;
      const replaced = editorApi.current?.replaceSelection(markdown);
      form.setValue(bodyPath, replaced?.text ?? `${form.getValues(bodyPath)}\n\n${markdown}`, { shouldDirty: true });
      form.setValue("media", [...media, { id: crypto.randomUUID(), file_id: file.id, role: "inline", sort_order: media.length, translations: { "zh-CN": { alt_text: file.original_name }, "en-US": { alt_text: file.original_name } } }], { shouldDirty: true });
    }
    setPicker(null);
  };
  const moveMedia = (index: number, delta: number) => { const target = index + delta; if (target < 0 || target >= media.length) return; const next = [...media]; const current = next[index]; const replacement = next[target]; if (!current || !replacement) return; next[index] = replacement; next[target] = current; form.setValue("media", next.map((item, sortOrder) => ({ ...item, sort_order: sortOrder })), { shouldDirty: true }); };
  const imageCommand: ICommand = { name: "asset-image", keyCommand: "asset-image", icon: <span aria-hidden="true">▧</span>, buttonProps: { "aria-label": t("content.article.actions.add_inline_image") }, execute: (_state, api) => { editorApi.current = api; setPicker("inline"); } };
  const imageUrlTransform = (url: string) => { const match = /^\/api\/v1\/public\/content\/assets\/([^/?#]+)/.exec(url); return match ? previewUrls[match[1] ?? ""] ?? url : url; };
  const validation = (message?: string) => message ? { validateStatus: "error" as const, help: message } : {};
  return <Drawer open={open} size={fullScreen ? "100%" : "large"} destroyOnHidden title={t("content.article.editor.edit")} onClose={onClose} extra={<Button type="primary" onClick={onSave}>{t("common.actions.save")}</Button>}>
    <Tabs activeKey={editorTab} onChange={setEditorTab} items={[{ key: "meta", label: t("content.article.editor.tabs.meta"), children: <Form layout="vertical">
      <Form.Item label={t("content.article.fields.type")}><Controller control={form.control} name="content_type" render={({ field }) => <Select {...field} options={["article", "gallery", "video"].map((value) => ({ value, label: t(`content.type.${value}`) }))} />} /></Form.Item>
      <Form.Item label="Slug" {...validation(errors.slug?.message)}><Controller control={form.control} name="slug" render={({ field }) => <Input {...field} />} /></Form.Item>
      <Space wrap><Form.Item label={t("content.article.fields.reading_minutes")}><Controller control={form.control} name="reading_minutes" render={({ field }) => <InputNumber min={1} max={120} value={field.value} onChange={(value) => { field.onChange(value ?? 1); }} />} /></Form.Item><Form.Item label={t("content.article.fields.sort")}><Controller control={form.control} name="sort_order" render={({ field }) => <InputNumber min={0} value={field.value} onChange={(value) => { field.onChange(value ?? 0); }} />} /></Form.Item></Space>
      <Form.Item label={t("content.article.fields.categories")}><Controller control={form.control} name="category_ids" render={({ field }) => <Select mode="multiple" maxCount={10} {...field} options={categories.filter((x) => x.status === "active").map((x) => ({ value: x.id, label: x.translations[locale].name }))} />} /></Form.Item>
      <Form.Item label={t("content.article.fields.topic")}><Controller control={form.control} name="topic_id" render={({ field }) => <Select allowClear {...field} value={field.value ?? undefined} onChange={(x) => { field.onChange(x ?? null); }} options={topics.filter((x) => x.status === "active").map((x) => ({ value: x.id, label: x.translations[locale].name }))} />} /></Form.Item>
      <Form.Item label={t("content.article.fields.tags")}><Controller control={form.control} name="tag_ids" render={({ field }) => <Select mode="multiple" maxCount={10} {...field} options={tags.filter((x) => x.status === "active").map((x) => ({ value: x.id, label: `#${x.name}` }))} />} /></Form.Item>
      <div className="ak-content-meta-cover-row">
        <Button onClick={() => { setPicker("cover"); }}>{t("content.actions.choose_cover")}</Button>
        {coverId && previewUrls[coverId] ? <Image className="ak-content-cover-preview" src={previewUrls[coverId]} alt={t("content.article.preview.cover")} /> : null}
      </div>
      <div className="ak-content-meta-options-row">
        <div className="ak-content-meta-option">
          <Typography.Text>{t("content.article.fields.comments")}</Typography.Text>
          <Controller control={form.control} name="allow_comments" render={({ field }) => <Switch aria-label={t("content.article.fields.comments")} checked={field.value} checkedChildren={t("content.values.yes")} unCheckedChildren={t("content.values.no")} onChange={field.onChange} />} />
        </div>
        <div className="ak-content-meta-option">
          <Typography.Text>{t("content.article.fields.pinned")}</Typography.Text>
          <Controller control={form.control} name="pinned" render={({ field }) => <Switch aria-label={t("content.article.fields.pinned")} checked={field.value} checkedChildren={t("content.values.yes")} unCheckedChildren={t("content.values.no")} onChange={field.onChange} />} />
        </div>
        <div className="ak-content-meta-option">
          <Typography.Text>{t("content.article.fields.featured")}</Typography.Text>
          <Controller control={form.control} name="featured" render={({ field }) => <Switch aria-label={t("content.article.fields.featured")} checked={field.value} checkedChildren={t("content.values.yes")} unCheckedChildren={t("content.values.no")} onChange={field.onChange} />} />
        </div>
        <div className="ak-content-meta-option">
          <Typography.Text>{t("content.article.fields.latest")}</Typography.Text>
          <Controller control={form.control} name="latest" render={({ field }) => <Switch aria-label={t("content.article.fields.latest")} checked={field.value} checkedChildren={t("content.values.yes")} unCheckedChildren={t("content.values.no")} onChange={field.onChange} />} />
        </div>
      </div>
    </Form> }, { key: "content", label: t("content.article.editor.tabs.content"), children: <>
      {type === "video" ? <><Form.Item label={t("content.article.fields.video_source")}><Controller control={form.control} name="video_source_type" render={({ field }) => <Select {...field} value={field.value ?? undefined} options={["upload", "external"].map((value) => ({ value, label: t(`content.video_source.${value}`) }))} />} /></Form.Item>{videoSource === "upload" ? <><Button onClick={() => { setPicker("video"); }}>{t("content.article.actions.choose_video")}</Button>{videoFileId && previewUrls[videoFileId] ? <video className="ak-content-video-preview" controls src={previewUrls[videoFileId]} /> : null}</> : <Form.Item label={t("content.article.fields.video_external_url")} {...validation(errors.video_external_url?.message)}><Controller control={form.control} name="video_external_url" render={({ field }) => <Input {...field} value={field.value ?? ""} placeholder="https://...mp4" />} /></Form.Item>}{videoSource === "external" && externalVideo ? <video className="ak-content-video-preview" controls src={externalVideo} /> : null}</> : null}
      {type === "gallery" ? <Form.Item label={t("content.article.fields.gallery")}><Button disabled={media.length >= 9} onClick={() => { setPicker("gallery"); }}>{t("content.article.actions.add_image")}</Button><div className="ak-content-gallery-grid">{media.map((item, index) => <Card key={item.id} size="small" cover={<div className="ak-content-gallery-preview">{previewUrls[item.file_id] ? <Image preview src={previewUrls[item.file_id] ?? ""} alt={item.translations[locale].alt_text} /> : previewErrors[item.file_id] ? <Typography.Text type="secondary">{t("system.files.picker.preview.error")}</Typography.Text> : <Typography.Text type="secondary">{t("content.article.preview.loading")}</Typography.Text>}</div>}><Typography.Text ellipsis title={item.translations[locale].alt_text}>{index + 1}. {item.translations[locale].alt_text}</Typography.Text><Space><Button size="small" aria-label={t("content.article.actions.move_up")} title={t("content.article.actions.move_up")} disabled={index === 0} onClick={() => { moveMedia(index, -1); }}>↑</Button><Button size="small" aria-label={t("content.article.actions.move_down")} title={t("content.article.actions.move_down")} disabled={index === media.length - 1} onClick={() => { moveMedia(index, 1); }}>↓</Button><Button size="small" danger onClick={() => { form.setValue("media", media.filter((x) => x.id !== item.id).map((x, sortOrder) => ({ ...x, sort_order: sortOrder })), { shouldDirty: true }); }}>{t("common.actions.delete")}</Button></Space></Card>)}</div></Form.Item> : null}
      <AkLocalizedFormTabs activeLocale={activeLocale} errorLocales={{ "zh-CN": Boolean(errors.translations?.["zh-CN"]), "en-US": Boolean(errors.translations?.["en-US"]) }} languages={languages} onActiveLocaleChange={(next) => { editorApi.current = null; onActiveLocaleChange(next); }} renderFields={(language, label) => <><Form.Item label={`${label} ${t("content.article.fields.title")}`}><Controller control={form.control} name={`translations.${language}.title`} render={({ field }) => <Input {...field} />} /></Form.Item><Form.Item label={`${label} ${t("content.article.fields.summary")}`}><Controller control={form.control} name={`translations.${language}.summary`} render={({ field }) => <Input.TextArea {...field} rows={4} showCount maxLength={type === "gallery" ? 3000 : 1000} />} /></Form.Item>{type === "article" ? <Controller control={form.control} name={`translations.${language}.body`} render={({ field }) => <div className="ak-markdown-editor"><Suspense fallback={<Typography.Paragraph type="secondary">{t("content.article.preview.loading")}</Typography.Paragraph>}><MDEditor value={field.value} onChange={(value) => { field.onChange(value ?? ""); }} preview="live" height={500} extraCommands={[imageCommand]} previewOptions={{ rehypePlugins: [[rehypeSanitize]], urlTransform: imageUrlTransform }} textareaProps={{ "aria-label": t("content.article.fields.body") }} /></Suspense></div>} /> : null}</>} />
    </> }]} />
    <Suspense fallback={null}><AkFilePicker open={picker !== null} kind={picker === "inline" || picker === "gallery" || picker === "cover" ? "image" : picker === "video" ? "video" : "all"} onClose={() => { setPicker(null); }} onSelect={(file) => void selectFile(file)} /></Suspense>
  </Drawer>;
}
