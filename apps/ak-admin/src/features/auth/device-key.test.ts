// @vitest-environment jsdom

import { beforeAll, beforeEach, describe, expect, it } from 'vitest'

import { readOrCreateAdminDeviceKey } from './device-key'

describe('readOrCreateAdminDeviceKey', () => {
  beforeAll(() => {
    const values = new Map<string, string>()
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      value: {
        clear: () => {
          values.clear()
        },
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => {
          values.set(key, value)
        },
      },
    })
  })

  beforeEach(() => {
    window.localStorage.clear()
  })

  it('persists one non-secret UUID per browser storage profile', () => {
    const first = readOrCreateAdminDeviceKey()
    const second = readOrCreateAdminDeviceKey()

    expect(first).toMatch(/^[0-9a-f-]{36}$/)
    expect(second).toBe(first)
  })

  it('replaces an invalid stored value', () => {
    window.localStorage.setItem('ak.admin.device-key.v1', 'invalid')

    expect(readOrCreateAdminDeviceKey()).not.toBe('invalid')
  })
})
