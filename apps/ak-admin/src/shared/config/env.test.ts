import { describe, expect, it } from 'vitest'

import { parseAdminEnvironment } from './env'

describe('parseAdminEnvironment', () => {
  it('accepts only the admin API prefix', () => {
    expect(
      parseAdminEnvironment({
        VITE_AK_API_BASE_URL: 'https://api.example.com/admin-api/v1',
        VITE_AK_APP_ENV: 'production',
      }),
    ).toEqual({
      VITE_AK_API_BASE_URL: 'https://api.example.com/admin-api/v1',
      VITE_AK_APP_ENV: 'production',
    })
  })

  it('rejects the mobile API prefix', () => {
    expect(() =>
      parseAdminEnvironment({
        VITE_AK_API_BASE_URL: 'https://api.example.com/api/v1',
        VITE_AK_APP_ENV: 'production',
      }),
    ).toThrow()
  })

  it('accepts the same-origin Docker proxy prefix', () => {
    expect(parseAdminEnvironment({
      VITE_AK_API_BASE_URL: '/admin-api/v1',
      VITE_AK_APP_ENV: 'production',
    }).VITE_AK_API_BASE_URL).toBe('/admin-api/v1')
  })
})
