// @vitest-environment jsdom

import { describe, expect, it } from 'vitest'

import { cropMetrics, rotatedSize, validateAvatarSource } from './image-crop'

describe('image crop helpers', () => {
  it('rotates dimensions for quarter turns', () => {
    expect(rotatedSize({ width: 1200, height: 800 }, 0)).toEqual({ width: 1200, height: 800 })
    expect(rotatedSize({ width: 1200, height: 800 }, 90)).toEqual({ width: 800, height: 1200 })
    expect(rotatedSize({ width: 1200, height: 800 }, 270)).toEqual({ width: 800, height: 1200 })
  })

  it('clamps offsets and calculates the visible square in source pixels', () => {
    const metrics = cropMetrics({ width: 1200, height: 800 }, 320, 1, 0, { x: 100, y: 50 })
    expect(metrics.baseScale).toBe(0.4)
    expect(metrics.scale).toBe(0.4)
    expect(metrics.offset).toEqual({ x: 80, y: 0 })
    expect(metrics.source).toEqual({ x: 0, y: 0, width: 800, height: 800 })
  })

  it('supports zoom and rotated crop coordinates', () => {
    const metrics = cropMetrics({ width: 1200, height: 800 }, 320, 2, 90, { x: -40, y: 80 })
    expect(metrics.scale).toBe(0.8)
    expect(metrics.source.width).toBe(400)
    expect(metrics.source.height).toBe(400)
    expect(metrics.source.x).toBe(250)
    expect(metrics.source.y).toBe(300)
  })

  it('accepts only bounded supported image files', () => {
    expect(validateAvatarSource(new File(['png'], 'avatar.png', { type: 'image/png' }), 1024)).toBe(true)
    expect(validateAvatarSource(new File(['svg'], 'avatar.svg', { type: 'image/svg+xml' }), 1024)).toBe(false)
    expect(validateAvatarSource(new File([new Uint8Array(2048)], 'large.webp', { type: 'image/webp' }), 1024)).toBe(false)
  })
})
