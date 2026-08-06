import { describe, expect, it, vi } from 'vitest'

import { readSidebarMode, sidebarModeStorageKey, writeSidebarMode } from './sidebar-store'

describe('sidebar mode preference', () => {
  it.each(['expanded', 'collapsed', 'hidden'] as const)('reads the supported %s mode', (mode) => {
    expect(readSidebarMode({ getItem: () => mode, setItem: vi.fn() })).toBe(mode)
  })

  it('falls back to expanded for missing, invalid, or unavailable storage', () => {
    expect(readSidebarMode(null)).toBe('expanded')
    expect(readSidebarMode({ getItem: () => 'invalid', setItem: vi.fn() })).toBe('expanded')
    expect(readSidebarMode({ getItem: () => { throw new Error('blocked') }, setItem: vi.fn() })).toBe('expanded')
  })

  it('writes only the non-sensitive display mode preference', () => {
    const setItem = vi.fn()
    writeSidebarMode('hidden', { getItem: vi.fn(), setItem })
    expect(setItem).toHaveBeenCalledWith(sidebarModeStorageKey, 'hidden')
  })
})
