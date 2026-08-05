export type QuarterTurn = 0 | 90 | 180 | 270

export interface CropPoint {
  x: number
  y: number
}

export interface CropSize {
  height: number
  width: number
}

export interface CropMetrics {
  baseScale: number
  offset: CropPoint
  scale: number
  source: { height: number; width: number; x: number; y: number }
}

export interface CropImageOptions {
  fileName: string
  naturalSize: CropSize
  offset: CropPoint
  outputSize: number
  rotation: QuarterTurn
  sourceUrl: string
  viewportSize: number
  zoom: number
}

const imageTypes = new Set(['image/jpeg', 'image/png', 'image/webp'])

export function validateAvatarSource(file: File, maxBytes: number): boolean {
  return imageTypes.has(file.type) && file.size > 0 && file.size <= maxBytes
}

export function rotatedSize(size: CropSize, rotation: QuarterTurn): CropSize {
  return rotation === 90 || rotation === 270
    ? { height: size.width, width: size.height }
    : size
}

export function cropMetrics(
  naturalSize: CropSize,
  viewportSize: number,
  zoom: number,
  rotation: QuarterTurn,
  requestedOffset: CropPoint,
): CropMetrics {
  const oriented = rotatedSize(naturalSize, rotation)
  const safeViewport = Math.max(1, viewportSize)
  const safeZoom = Math.max(1, zoom)
  const baseScale = Math.max(safeViewport / oriented.width, safeViewport / oriented.height)
  const scale = baseScale * safeZoom
  const maxOffsetX = Math.max(0, (oriented.width * scale - safeViewport) / 2)
  const maxOffsetY = Math.max(0, (oriented.height * scale - safeViewport) / 2)
  const offset = {
    x: clamp(requestedOffset.x, -maxOffsetX, maxOffsetX),
    y: clamp(requestedOffset.y, -maxOffsetY, maxOffsetY),
  }
  const sourceSize = Math.min(oriented.width, oriented.height, safeViewport / scale)
  return {
    baseScale,
    offset,
    scale,
    source: {
      height: sourceSize,
      width: sourceSize,
      x: clamp((oriented.width - sourceSize) / 2 - offset.x / scale, 0, oriented.width - sourceSize),
      y: clamp((oriented.height - sourceSize) / 2 - offset.y / scale, 0, oriented.height - sourceSize),
    },
  }
}

export async function cropImageToFile(options: CropImageOptions): Promise<File> {
  const image = await loadImage(options.sourceUrl)
  const oriented = orientedCanvas(image, options.rotation)
  const metrics = cropMetrics(
    options.naturalSize,
    options.viewportSize,
    options.zoom,
    options.rotation,
    options.offset,
  )
  const outputSize = Math.max(128, Math.min(1024, Math.round(options.outputSize)))
  const output = document.createElement('canvas')
  output.width = outputSize
  output.height = outputSize
  const context = output.getContext('2d')
  if (!context) throw new Error('canvas 2d context is unavailable')
  context.imageSmoothingEnabled = true
  context.imageSmoothingQuality = 'high'
  context.drawImage(
    oriented,
    metrics.source.x,
    metrics.source.y,
    metrics.source.width,
    metrics.source.height,
    0,
    0,
    outputSize,
    outputSize,
  )
  const blob = await canvasBlob(output, 'image/png')
  return new File([blob], avatarFileName(options.fileName), {
    type: 'image/png',
    lastModified: Date.now(),
  })
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value))
}

function avatarFileName(name: string): string {
  const stem = name.replace(/\.[^.]+$/, '').trim().replace(/[^a-zA-Z0-9_-]+/g, '-') || 'avatar'
  return `${stem}-cropped.png`
}

function loadImage(sourceUrl: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => { resolve(image) }
    image.onerror = () => { reject(new Error('image decode failed')) }
    image.src = sourceUrl
  })
}

function orientedCanvas(image: HTMLImageElement, rotation: QuarterTurn): HTMLCanvasElement {
  const swap = rotation === 90 || rotation === 270
  const canvas = document.createElement('canvas')
  canvas.width = swap ? image.naturalHeight : image.naturalWidth
  canvas.height = swap ? image.naturalWidth : image.naturalHeight
  const context = canvas.getContext('2d')
  if (!context) throw new Error('canvas 2d context is unavailable')
  if (rotation === 90) {
    context.translate(canvas.width, 0)
    context.rotate(Math.PI / 2)
  } else if (rotation === 180) {
    context.translate(canvas.width, canvas.height)
    context.rotate(Math.PI)
  } else if (rotation === 270) {
    context.translate(0, canvas.height)
    context.rotate(-Math.PI / 2)
  }
  context.drawImage(image, 0, 0)
  return canvas
}

function canvasBlob(canvas: HTMLCanvasElement, type: string): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (blob) resolve(blob)
      else reject(new Error('image export failed'))
    }, type)
  })
}
