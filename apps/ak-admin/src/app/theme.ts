import type { ThemeConfig } from 'antd'

export const adminTheme: ThemeConfig = {
  token: {
    colorPrimary: '#1E40AF',
    colorInfo: '#2563EB',
    colorSuccess: '#15803D',
    colorWarning: '#B45309',
    colorError: '#B91C1C',
    colorBgLayout: '#F4F7FB',
    colorText: '#172033',
    colorTextSecondary: '#536179',
    colorTextDescription: '#536179',
    colorLink: '#075985',
    colorLinkHover: '#0C4A6E',
    borderRadius: 8,
    borderRadiusLG: 12,
    controlHeight: 40,
    fontFamily: 'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  },
  components: {
    Button: { primaryShadow: 'none' },
    Card: { boxShadowTertiary: '0 1px 3px rgba(15, 23, 42, 0.08)' },
    Layout: { bodyBg: '#F4F7FB', headerBg: '#FFFFFF', siderBg: '#0F2147' },
    Menu: { darkItemBg: '#0F2147', darkSubMenuItemBg: '#0B1935' },
  },
}
