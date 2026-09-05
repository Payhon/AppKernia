// @vitest-environment jsdom

import { describe, expect, it } from 'vitest'

import {
  normalizeAdminBasePath,
  withAdminBasePath,
  withoutAdminBasePath,
} from './base-path'

describe('Admin base path', () => {
  it('normalizes a configured path and preserves root', () => {
    expect(normalizeAdminBasePath('/admin/')).toBe('/admin')
    expect(normalizeAdminBasePath('//ops//admin/')).toBe('/ops/admin')
    expect(normalizeAdminBasePath('/')).toBe('/')
    expect(normalizeAdminBasePath('admin')).toBe('/')
  })

  it('adds and removes the physical prefix without changing logical routes', () => {
    expect(withAdminBasePath('/login', '/admin')).toBe('/admin/login')
    expect(withAdminBasePath('/', '/admin')).toBe('/admin/')
    expect(withoutAdminBasePath('/admin/dashboard', '/admin')).toBe('/dashboard')
    expect(withoutAdminBasePath('/admin', '/admin')).toBe('/')
    expect(withoutAdminBasePath('/administrator', '/admin')).toBe('/administrator')
  })
})
