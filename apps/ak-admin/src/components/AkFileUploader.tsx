import { Alert, Button, Card, Progress, Space, Typography } from "antd";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import type { AdminFile } from "../generated/api/types.gen";
import { useAdminFileUploadPolicy } from "../features/files/hooks";
import { useAdminDictionary } from "../features/settings/hooks";
import { authSession } from "../features/auth/store";

type UploadState = "idle" | "uploading" | "paused" | "error" | "completed" | "cancelled";
interface UploadTask { file: File; sessionId?: string; progress: number; state: UploadState }
interface AkFileUploaderProps { onUploaded?: (file: AdminFile) => void | Promise<void>; compact?: boolean }

function sizeLabel(bytes: number) {
  if (bytes < 1024) return `${String(bytes)} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MiB`;
}

export function AkFileUploader({ onUploaded, compact = false }: AkFileUploaderProps) {
  const { t } = useTranslation();
  const policy = useAdminFileUploadPolicy();
  const storageDrivers = useAdminDictionary("storage.driver");
  const providerLabel = storageDrivers.data?.items.find((item) => item.value === policy.data?.provider)?.label ?? policy.data?.provider;
  const [task, setTask] = useState<UploadTask | null>(null);
  const [feedback, setFeedback] = useState<{ key: string; error?: boolean } | null>(null);
  const controller = useRef<AbortController | null>(null);
  const input = useRef<HTMLInputElement | null>(null);

  const startUpload = async (current: UploadTask) => {
    controller.current = new AbortController();
    setTask({ ...current, state: "uploading" });
    setFeedback(null);
    try {
      const uploaded = await authSession.uploadAdminFile(current.file, {
        ...(current.sessionId ? { sessionId: current.sessionId } : {}),
        signal: controller.current.signal,
        onSession: (session) => { setTask((value) => value ? { ...value, sessionId: session.id } : value); },
        onProgress: (progress) => { setTask((value) => value ? { ...value, progress } : value); },
      });
      setTask((value) => value ? { ...value, state: "completed", progress: 100 } : value);
      setFeedback({ key: "system.files.upload.completed" });
      try {
        await onUploaded?.(uploaded);
      } catch {
        // The object is already committed. A consumer refresh failure must not
        // turn the completed upload into a resumable error state.
      }
    } catch {
      if (controller.current.signal.aborted) setTask((value) => value ? { ...value, state: "paused" } : value);
      else {
        setTask((value) => value ? { ...value, state: "error" } : value);
        setFeedback({ key: "system.files.upload.error", error: true });
      }
    }
  };

  const choose = (file: File | undefined) => {
    if (!file || !policy.data) return;
    const mediaType = file.type || "application/octet-stream";
    if (file.size <= 0 || file.size > policy.data.max_file_bytes || !policy.data.file_media_types.includes(mediaType)) {
      setFeedback({ key: "system.files.upload.invalid_policy", error: true });
      return;
    }
    const next = { file, progress: 0, state: "idle" as const };
    setTask(next);
    void startUpload(next);
  };

  const cancel = async () => {
    if (!task?.sessionId) return;
    controller.current?.abort();
    try {
      await authSession.cancelAdminFileUpload(task.sessionId);
      setTask((value) => value ? { ...value, state: "cancelled" } : value);
      setFeedback({ key: "system.files.upload.cancelled" });
    } catch {
      setFeedback({ key: "system.files.upload.error", error: true });
    }
  };

  return (
    <div className={compact ? "ak-file-uploader ak-file-uploader-compact" : "ak-file-uploader"}>
      <Space wrap>
        <input ref={input} className="ak-sr-only" type="file" accept={policy.data?.file_media_types.join(",")} aria-label={t("system.files.actions.choose")} onChange={(event) => { choose(event.target.files?.[0]); event.currentTarget.value = ""; }} />
        <Button type="primary" loading={policy.isPending} disabled={!policy.data?.configuration_safe} onClick={() => input.current?.click()}>{t("system.files.actions.upload")}</Button>
        {policy.data ? <Typography.Text type="secondary">{t("system.files.upload.policy", { provider: providerLabel, size: sizeLabel(policy.data.max_file_bytes) })}</Typography.Text> : null}
      </Space>
      {policy.isError ? <Alert showIcon type="error" title={t("system.files.upload.policy_error")} action={<Button onClick={() => void policy.refetch()}>{t("common.actions.retry")}</Button>} /> : null}
      {task ? <Card size="small" className="ak-file-upload-card" title={t("system.files.upload.title")}>
        <Typography.Paragraph>{task.file.name} · {sizeLabel(task.file.size)}</Typography.Paragraph>
        <Progress aria-label={t("system.files.upload.progress", { percent: task.progress })} percent={task.progress} />
        <div aria-live="polite">{t(task.state === "paused" ? "system.files.upload.paused" : "system.files.upload.progress", { percent: task.progress })}</div>
        <Space wrap>
          {task.state === "uploading" ? <Button onClick={() => controller.current?.abort()}>{t("system.files.actions.pause")}</Button> : null}
          {["paused", "error"].includes(task.state) ? <Button type="primary" onClick={() => void startUpload(task)}>{t("system.files.actions.resume")}</Button> : null}
          {task.sessionId && !["completed", "cancelled"].includes(task.state) ? <Button danger onClick={() => void cancel()}>{t("system.files.actions.cancel_upload")}</Button> : null}
        </Space>
      </Card> : null}
      {feedback ? <div className={feedback.error ? "ak-form-error" : "ak-org-feedback"} role={feedback.error ? "alert" : "status"}>{t(feedback.key)}</div> : null}
    </div>
  );
}
