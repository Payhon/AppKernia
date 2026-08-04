import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import coreMenuSeed from '../../../../blueprint/admin-frontend/spec/admin-menu-seed.json'
import { ConfiguredMenuIcon, isConfiguredMenuIcon } from './menu-icons'

describe('configured menu icon registry', () => {
  it('accepts only compile-time allowlisted icon names', () => {
    expect(isConfiguredMenuIcon('DashboardOutlined')).toBe(true)
    expect(isConfiguredMenuIcon('UserSwitchOutlined')).toBe(true)
    expect(isConfiguredMenuIcon('server.dynamic.Import')).toBe(false)
  })

  it('has an allowlisted configured icon for every core menu row', () => {
    expect(coreMenuSeed.menus).toHaveLength(35)
    for (const menu of coreMenuSeed.menus) {
      expect(menu.icon, menu.code).toBeTypeOf('string')
      expect(isConfiguredMenuIcon(menu.icon), menu.code).toBe(true)
    }
  })

  it('renders an accessible-hidden fallback for missing and unknown names', () => {
    const missing = renderToStaticMarkup(<ConfiguredMenuIcon name={null} />)
    const unknown = renderToStaticMarkup(<ConfiguredMenuIcon name="server.dynamic.Import" />)
    expect(missing).toContain('anticon-appstore')
    expect(unknown).toContain('anticon-appstore')
    expect(missing).toContain('aria-hidden="true"')
  })
})
