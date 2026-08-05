// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import axe from 'axe-core'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import { LocaleProvider } from '../shared/i18n'
import type * as ImageCropModule from '../shared/media/image-crop'
import { cropImageToFile } from '../shared/media/image-crop'
import { AkAvatarUploader } from './AkAvatarUploader'

vi.mock('../shared/media/image-crop', async (importOriginal) => {
  const actual = await importOriginal<typeof ImageCropModule>()
  return {
    ...actual,
    cropImageToFile: vi.fn(() => Promise.resolve(new File(['cropped'], 'avatar-cropped.png', { type: 'image/png' }))),
  }
})

beforeAll(() => {
  let objectUrlIndex = 0
  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    value: vi.fn(() => `blob:avatar-${String(++objectUrlIndex)}`),
  })
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    value: vi.fn(),
  })
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false, media: query, onchange: null,
      addListener: vi.fn(), removeListener: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn(),
    })),
  })
  class TestResizeObserver {
    observe() { return undefined }
    disconnect() { return undefined }
    unobserve() { return undefined }
  }
  Object.defineProperty(globalThis, 'ResizeObserver', { configurable: true, value: TestResizeObserver })
})

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

describe('AkAvatarUploader', () => {
  it('crops a selected image before uploading and announces progress', async () => {
    const upload = vi.fn((_file: File, onProgress: (percent: number) => void) => {
      onProgress(35)
      onProgress(100)
      return Promise.resolve()
    })
    const { container } = render(
      <LocaleProvider>
        <AkAvatarUploader alt="Test avatar" fallback="T" onUpload={upload} />
      </LocaleProvider>,
    )
    const input = screen.getByLabelText(/选择图片|Choose image/)
    const source = new File(['source'], 'portrait.jpg', { type: 'image/jpeg' })
    fireEvent.change(input, { target: { files: [source] } })

    await screen.findByRole('dialog', { name: /裁剪头像|Crop avatar/ })
    const image = container.ownerDocument.querySelector<HTMLImageElement>('.ak-image-crop-viewport img')
    if (!image) throw new Error('crop preview image is missing')
    Object.defineProperty(image, 'naturalWidth', { configurable: true, value: 1200 })
    Object.defineProperty(image, 'naturalHeight', { configurable: true, value: 800 })
    fireEvent.load(image)

    fireEvent.click(screen.getByRole('button', { name: /向右旋转 90 度|Rotate 90 degrees right/ }))
    fireEvent.click(screen.getByRole('button', { name: /向右移动图片|Move image right/ }))
    fireEvent.click(screen.getByRole('button', { name: /使用此裁剪|Use this crop/ }))

    await waitFor(() => { expect(cropImageToFile).toHaveBeenCalledOnce() })
    await screen.findByText(/512 × 512/)
    fireEvent.click(screen.getByRole('button', { name: /上传头像|Upload avatar/ }))
    await waitFor(() => { expect(upload).toHaveBeenCalledOnce() })
    expect(upload.mock.calls[0]?.[0]).toMatchObject({ name: 'avatar-cropped.png', type: 'image/png' })
    expect((await screen.findByRole('status')).textContent).toMatch(/头像已更新|avatar has been updated/i)

    const accessibility = await axe.run(document.body)
    expect(accessibility.violations).toEqual([])
  })

  it('rejects unsupported files before opening the crop dialog', () => {
    render(
      <LocaleProvider>
        <AkAvatarUploader alt="Test avatar" fallback="T" onUpload={vi.fn()} />
      </LocaleProvider>,
    )
    fireEvent.change(screen.getByLabelText(/选择图片|Choose image/), {
      target: { files: [new File(['svg'], 'avatar.svg', { type: 'image/svg+xml' })] },
    })
    expect(screen.getByRole('alert').textContent).toMatch(/JPEG|PNG|WebP/)
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('preserves the cropped preview when upload fails so the user can retry', async () => {
    const upload = vi.fn(() => Promise.reject(new Error('storage unavailable')))
    const { container } = render(
      <LocaleProvider>
        <AkAvatarUploader alt="Test avatar" fallback="T" onUpload={upload} />
      </LocaleProvider>,
    )
    fireEvent.change(screen.getByLabelText(/选择图片|Choose image/), {
      target: { files: [new File(['source'], 'portrait.webp', { type: 'image/webp' })] },
    })
    await screen.findByRole('dialog', { name: /裁剪头像|Crop avatar/ })
    const image = container.ownerDocument.querySelector<HTMLImageElement>('.ak-image-crop-viewport img')
    if (!image) throw new Error('crop preview image is missing')
    Object.defineProperty(image, 'naturalWidth', { configurable: true, value: 900 })
    Object.defineProperty(image, 'naturalHeight', { configurable: true, value: 900 })
    fireEvent.load(image)
    fireEvent.click(screen.getByRole('button', { name: /使用此裁剪|Use this crop/ }))
    await waitFor(() => { expect(cropImageToFile).toHaveBeenCalledOnce() })

    const uploadButton = screen.getByRole('button', { name: /上传头像|Upload avatar/ })
    fireEvent.click(uploadButton)
    await waitFor(() => { expect(upload).toHaveBeenCalledOnce() })
    expect((await screen.findByRole('alert')).textContent).toMatch(/上传失败|upload failed/i)
    expect((uploadButton as HTMLButtonElement).disabled).toBe(false)
    expect(screen.getByText(/512 × 512/)).toBeTruthy()
  })
})
