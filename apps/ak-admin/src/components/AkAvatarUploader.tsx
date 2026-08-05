import { Avatar, Button, Progress, Space, Typography } from 'antd'
import type { ReactNode } from 'react'
import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { validateAvatarSource } from '../shared/media/image-crop'
import { AkImageCropper } from './AkImageCropper'

interface AkAvatarUploaderProps {
  alt: string
  currentSrc?: string | null
  disabled?: boolean
  fallback: ReactNode
  maxBytes?: number
  onUpload: (file: File, onProgress: (percent: number) => void) => Promise<void>
}

type AvatarFeedback = 'success' | 'invalid' | 'upload_error' | null

export function AkAvatarUploader({
  alt,
  currentSrc = null,
  disabled = false,
  fallback,
  maxBytes = 5 * 1024 * 1024,
  onUpload,
}: AkAvatarUploaderProps) {
  const { t } = useTranslation()
  const titleId = useId()
  const input = useRef<HTMLInputElement>(null)
  const [sourceFile, setSourceFile] = useState<File | null>(null)
  const [committedSourceFile, setCommittedSourceFile] = useState<File | null>(null)
  const [sourceUrl, setSourceUrl] = useState<string | null>(null)
  const [croppedFile, setCroppedFile] = useState<File | null>(null)
  const [croppedUrl, setCroppedUrl] = useState<string | null>(null)
  const [cropOpen, setCropOpen] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [feedback, setFeedback] = useState<AvatarFeedback>(null)

  useEffect(() => {
    if (!sourceFile) {
      setSourceUrl(null)
      return
    }
    const next = URL.createObjectURL(sourceFile)
    setSourceUrl(next)
    return () => { URL.revokeObjectURL(next) }
  }, [sourceFile])

  useEffect(() => {
    if (!croppedFile) {
      setCroppedUrl(null)
      return
    }
    const next = URL.createObjectURL(croppedFile)
    setCroppedUrl(next)
    return () => { URL.revokeObjectURL(next) }
  }, [croppedFile])

  const choose = (file: File | undefined) => {
    setFeedback(null)
    setProgress(0)
    if (!file) return
    if (!validateAvatarSource(file, maxBytes)) {
      setFeedback('invalid')
      return
    }
    setSourceFile(file)
    setCropOpen(true)
  }

  const upload = async () => {
    if (!croppedFile) return
    setUploading(true)
    setFeedback(null)
    setProgress(0)
    try {
      await onUpload(croppedFile, setProgress)
      setFeedback('success')
      setProgress(100)
    } catch {
      setFeedback('upload_error')
      setProgress(0)
    } finally {
      setUploading(false)
    }
  }

  return (
    <section className="ak-avatar-uploader" aria-labelledby={titleId}>
      <div className="ak-avatar-preview">
        <Avatar alt={alt} size={96} src={croppedUrl ?? currentSrc ?? undefined}>{fallback}</Avatar>
        <div>
          <Typography.Title id={titleId} level={2}>{t('common.avatar_upload.title')}</Typography.Title>
          <Typography.Paragraph type="secondary">{t('common.avatar_upload.description')}</Typography.Paragraph>
        </div>
      </div>
      <input
        accept="image/jpeg,image/png,image/webp"
        aria-label={t('common.avatar_upload.choose')}
        className="ak-sr-only"
        disabled={disabled || uploading}
        onChange={(event) => {
          choose(event.currentTarget.files?.[0])
          event.currentTarget.value = ''
        }}
        ref={input}
        type="file"
      />
      <Space className="ak-avatar-actions" wrap>
        <Button disabled={disabled || uploading} onClick={() => { input.current?.click() }}>
          {croppedFile ? t('common.avatar_upload.choose_another') : t('common.avatar_upload.choose')}
        </Button>
        {croppedFile ? <Button disabled={uploading} onClick={() => { setCropOpen(true) }}>{t('common.avatar_upload.crop_again')}</Button> : null}
        <Button disabled={!croppedFile || feedback === 'success'} loading={uploading} onClick={() => { void upload() }} type="primary">
          {t('common.avatar_upload.upload')}
        </Button>
      </Space>
      {croppedFile ? <Typography.Paragraph className="ak-avatar-ready" type="secondary">{t('common.avatar_upload.ready', { size: 512 })}</Typography.Paragraph> : null}
      {uploading || progress > 0 ? (
        <div aria-live="polite" className="ak-avatar-progress">
          <Progress aria-label={t('common.avatar_upload.progress', { percent: progress })} percent={progress} size="small" />
          <span>{t('common.avatar_upload.progress', { percent: progress })}</span>
        </div>
      ) : null}
      {feedback === 'success' ? <div className="ak-form-success" role="status">{t('common.avatar_upload.success')}</div> : null}
      {feedback === 'invalid' ? <div className="ak-form-error" role="alert">{t('common.avatar_upload.invalid')}</div> : null}
      {feedback === 'upload_error' ? <div className="ak-form-error" role="alert">{t('common.avatar_upload.upload_error')}</div> : null}
      {sourceFile && sourceUrl ? (
        <AkImageCropper
          file={sourceFile}
          onCancel={() => {
            setCropOpen(false)
            setSourceFile(committedSourceFile)
          }}
          onConfirm={(file) => {
            setCroppedFile(file)
            setCommittedSourceFile(sourceFile)
            setFeedback(null)
            setProgress(0)
            setCropOpen(false)
          }}
          open={cropOpen}
          sourceUrl={sourceUrl}
        />
      ) : null}
    </section>
  )
}
