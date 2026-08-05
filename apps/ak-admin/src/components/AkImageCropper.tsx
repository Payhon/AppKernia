import {
  ArrowDownOutlined,
  ArrowLeftOutlined,
  ArrowRightOutlined,
  ArrowUpOutlined,
  RotateLeftOutlined,
  RotateRightOutlined,
} from '@ant-design/icons'
import { Alert, Button, Modal, Slider, Space, Typography } from 'antd'
import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  cropImageToFile,
  cropMetrics,
  type CropPoint,
  type CropSize,
  type QuarterTurn,
} from '../shared/media/image-crop'

interface AkImageCropperProps {
  file: File
  onCancel: () => void
  onConfirm: (file: File) => void
  open: boolean
  outputSize?: number
  sourceUrl: string
}

interface DragState {
  offset: CropPoint
  pointerId: number
  x: number
  y: number
}

const initialOffset: CropPoint = { x: 0, y: 0 }

export function AkImageCropper({ file, onCancel, onConfirm, open, outputSize = 512, sourceUrl }: AkImageCropperProps) {
  const { t } = useTranslation()
  const zoomId = useId()
  const viewport = useRef<HTMLDivElement>(null)
  const drag = useRef<DragState | null>(null)
  const [naturalSize, setNaturalSize] = useState<CropSize | null>(null)
  const [viewportSize, setViewportSize] = useState(320)
  const [offset, setOffset] = useState<CropPoint>(initialOffset)
  const [zoom, setZoom] = useState(1)
  const [rotation, setRotation] = useState<QuarterTurn>(0)
  const [dragging, setDragging] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(false)

  useEffect(() => {
    if (!open) return
    setNaturalSize(null)
    setOffset(initialOffset)
    setZoom(1)
    setRotation(0)
    setError(false)
  }, [file, open])

  useEffect(() => {
    const element = viewport.current
    if (!element) return
    const update = () => { setViewportSize(Math.max(1, element.getBoundingClientRect().width)) }
    update()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(update)
    observer.observe(element)
    return () => { observer.disconnect() }
  }, [open])

  const metrics = useMemo(() => naturalSize
    ? cropMetrics(naturalSize, viewportSize, zoom, rotation, offset)
    : null, [naturalSize, offset, rotation, viewportSize, zoom])

  useEffect(() => {
    if (!metrics || (metrics.offset.x === offset.x && metrics.offset.y === offset.y)) return
    setOffset(metrics.offset)
  }, [metrics, offset])

  const move = (x: number, y: number) => {
    if (!naturalSize) return
    setOffset((current) => cropMetrics(naturalSize, viewportSize, zoom, rotation, {
      x: current.x + x,
      y: current.y + y,
    }).offset)
  }

  const rotate = (direction: -1 | 1) => {
    setRotation((current) => (((current + direction * 90) % 360 + 360) % 360) as QuarterTurn)
    setOffset(initialOffset)
  }

  const confirm = async () => {
    if (!naturalSize) return
    setBusy(true)
    setError(false)
    try {
      onConfirm(await cropImageToFile({
        fileName: file.name,
        naturalSize,
        offset,
        outputSize,
        rotation,
        sourceUrl,
        viewportSize,
        zoom,
      }))
    } catch {
      setError(true)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal
      className="ak-image-crop-modal"
      destroyOnHidden
      mask={{ closable: !busy }}
      okButtonProps={{ disabled: !naturalSize, loading: busy }}
      okText={t('common.avatar_upload.crop.confirm')}
      onCancel={() => { if (!busy) onCancel() }}
      onOk={() => { void confirm() }}
      open={open}
      title={t('common.avatar_upload.crop.title')}
    >
      <Typography.Paragraph type="secondary">{t('common.avatar_upload.crop.description')}</Typography.Paragraph>
      <div
        aria-label={t('common.avatar_upload.crop.area')}
        className={dragging ? 'ak-image-crop-viewport is-dragging' : 'ak-image-crop-viewport'}
        onPointerDown={(event) => {
          if (!naturalSize) return
          event.currentTarget.setPointerCapture(event.pointerId)
          drag.current = { offset, pointerId: event.pointerId, x: event.clientX, y: event.clientY }
          setDragging(true)
        }}
        onPointerMove={(event) => {
          const current = drag.current
          if (current?.pointerId !== event.pointerId || !naturalSize) return
          setOffset(cropMetrics(naturalSize, viewportSize, zoom, rotation, {
            x: current.offset.x + event.clientX - current.x,
            y: current.offset.y + event.clientY - current.y,
          }).offset)
        }}
        onPointerUp={(event) => {
          if (drag.current?.pointerId === event.pointerId) drag.current = null
          setDragging(false)
        }}
        ref={viewport}
        role="img"
      >
        <img
          alt=""
          aria-hidden="true"
          draggable={false}
          onError={() => { setError(true) }}
          onLoad={(event) => {
            const image = event.currentTarget
            if (image.naturalWidth <= 0 || image.naturalHeight <= 0 || image.naturalWidth > 8192 || image.naturalHeight > 8192 || image.naturalWidth * image.naturalHeight > 32_000_000) {
              setError(true)
              return
            }
            setNaturalSize({ height: image.naturalHeight, width: image.naturalWidth })
          }}
          src={sourceUrl}
          style={naturalSize && metrics ? {
            height: naturalSize.height,
            left: `calc(50% + ${String(metrics.offset.x)}px)`,
            top: `calc(50% + ${String(metrics.offset.y)}px)`,
            transform: `translate(-50%, -50%) rotate(${String(rotation)}deg) scale(${String(metrics.scale)})`,
            width: naturalSize.width,
          } : undefined}
        />
        <div aria-hidden="true" className="ak-image-crop-mask" />
        <div aria-hidden="true" className="ak-image-crop-grid" />
      </div>
      <div className="ak-image-crop-controls">
        <div className="ak-image-crop-slider">
          <label htmlFor={zoomId}>{t('common.avatar_upload.crop.zoom')}</label>
          <Slider
            ariaLabelForHandle={t('common.avatar_upload.crop.zoom')}
            disabled={!naturalSize}
            id={zoomId}
            max={3}
            min={1}
            onChange={setZoom}
            step={0.05}
            value={zoom}
          />
        </div>
        <Space className="ak-image-crop-tool-row" wrap>
          <Button aria-label={t('common.avatar_upload.crop.move_left')} disabled={!naturalSize} icon={<ArrowLeftOutlined />} onClick={() => { move(-8, 0) }} />
          <Button aria-label={t('common.avatar_upload.crop.move_up')} disabled={!naturalSize} icon={<ArrowUpOutlined />} onClick={() => { move(0, -8) }} />
          <Button aria-label={t('common.avatar_upload.crop.move_down')} disabled={!naturalSize} icon={<ArrowDownOutlined />} onClick={() => { move(0, 8) }} />
          <Button aria-label={t('common.avatar_upload.crop.move_right')} disabled={!naturalSize} icon={<ArrowRightOutlined />} onClick={() => { move(8, 0) }} />
          <Button aria-label={t('common.avatar_upload.crop.rotate_left')} disabled={!naturalSize} icon={<RotateLeftOutlined />} onClick={() => { rotate(-1) }} />
          <Button aria-label={t('common.avatar_upload.crop.rotate_right')} disabled={!naturalSize} icon={<RotateRightOutlined />} onClick={() => { rotate(1) }} />
        </Space>
      </div>
      {error ? <Alert role="alert" showIcon title={t('common.avatar_upload.crop.error')} type="error" /> : null}
    </Modal>
  )
}
