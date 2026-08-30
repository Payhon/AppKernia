import { ConfigProvider } from 'antd'
import enUS from 'antd/locale/en_US'
import zhCN from 'antd/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/en'
import 'dayjs/locale/zh-cn'
import i18n from 'i18next'
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type PropsWithChildren } from 'react'
import { I18nextProvider } from 'react-i18next'

import enAuth from '../../locales/en-US/auth.json'
import enApps from '../../locales/en-US/apps.json'
import enCommon from '../../locales/en-US/common.json'
import enContent from '../../locales/en-US/content.json'
import enErrors from '../../locales/en-US/errors.json'
import enMobileReleases from '../../locales/en-US/mobile_releases.json'
import enNavigation from '../../locales/en-US/navigation.json'
import enNotifications from '../../locales/en-US/notifications.json'
import enOpenApi from '../../locales/en-US/openapi.json'
import enProfile from '../../locales/en-US/profile.json'
import enPushChannels from '../../locales/en-US/push_channels.json'
import enShareConfigs from '../../locales/en-US/share_configs.json'
import enSettings from '../../locales/en-US/settings.json'
import enSystem from '../../locales/en-US/system.json'
import enValidation from '../../locales/en-US/validation.json'
import zhAuth from '../../locales/zh-CN/auth.json'
import zhApps from '../../locales/zh-CN/apps.json'
import zhCommon from '../../locales/zh-CN/common.json'
import zhContent from '../../locales/zh-CN/content.json'
import zhErrors from '../../locales/zh-CN/errors.json'
import zhMobileReleases from '../../locales/zh-CN/mobile_releases.json'
import zhNavigation from '../../locales/zh-CN/navigation.json'
import zhNotifications from '../../locales/zh-CN/notifications.json'
import zhOpenApi from '../../locales/zh-CN/openapi.json'
import zhProfile from '../../locales/zh-CN/profile.json'
import zhPushChannels from '../../locales/zh-CN/push_channels.json'
import zhShareConfigs from '../../locales/zh-CN/share_configs.json'
import zhSettings from '../../locales/zh-CN/settings.json'
import zhSystem from '../../locales/zh-CN/system.json'
import zhValidation from '../../locales/zh-CN/validation.json'
import { adminTheme } from '../../app/theme'

export const supportedLocales = ['zh-CN', 'en-US'] as const
export type AdminLocale = (typeof supportedLocales)[number]

const localeStorageKey = 'ak.admin.locale'
const enCatalog = { ...enApps, ...enAuth, ...enCommon, ...enContent, ...enErrors, ...enMobileReleases, ...enNavigation, ...enNotifications, ...enOpenApi, ...enProfile, ...enPushChannels, ...enShareConfigs, ...enSettings, ...enSystem, ...enValidation }
const zhCatalog = { ...zhApps, ...zhAuth, ...zhCommon, ...zhContent, ...zhErrors, ...zhMobileReleases, ...zhNavigation, ...zhNotifications, ...zhOpenApi, ...zhProfile, ...zhPushChannels, ...zhShareConfigs, ...zhSettings, ...zhSystem, ...zhValidation }

function readStoredLocale(): string | null {
  try {
    return window.localStorage.getItem(localeStorageKey)
  } catch {
    return null
  }
}

function storeLocale(locale: AdminLocale): void {
  try {
    window.localStorage.setItem(localeStorageKey, locale)
  } catch {
    // Storage can be unavailable in privacy-restricted browser contexts.
  }
}

function normalizeLocale(value: string | null | undefined): AdminLocale {
  return value?.toLowerCase().startsWith('en') ? 'en-US' : 'zh-CN'
}

const initialLocale = normalizeLocale(
  typeof window === 'undefined'
    ? undefined
    : readStoredLocale() ?? window.navigator.language,
)

void i18n.init({
  fallbackLng: 'zh-CN',
  initAsync: false,
  interpolation: { escapeValue: false, prefix: '{', suffix: '}' },
  keySeparator: false,
  lng: initialLocale,
  resources: {
    'en-US': { translation: enCatalog },
    'zh-CN': { translation: zhCatalog },
  },
})

interface LocaleContextValue {
  locale: AdminLocale
  setLocale: (locale: AdminLocale) => Promise<void>
}

const LocaleContext = createContext<LocaleContextValue | null>(null)

export function readActiveLocale(): AdminLocale {
  return normalizeLocale(i18n.resolvedLanguage)
}

export function LocaleProvider({ children }: PropsWithChildren) {
  const [locale, setLocaleState] = useState<AdminLocale>(initialLocale)

  const setLocale = useCallback(async (nextLocale: AdminLocale) => {
    await i18n.changeLanguage(nextLocale)
    dayjs.locale(nextLocale === 'zh-CN' ? 'zh-cn' : 'en')
    document.documentElement.lang = nextLocale
    document.documentElement.dir = 'ltr'
    document.title = i18n.t('app.name')
    storeLocale(nextLocale)
    setLocaleState(nextLocale)
  }, [])

  useEffect(() => {
    void setLocale(initialLocale)
  }, [setLocale])

  const value = useMemo(() => ({ locale, setLocale }), [locale, setLocale])
  const antLocale = locale === 'zh-CN' ? zhCN : enUS

  return (
    <LocaleContext.Provider value={value}>
      <I18nextProvider i18n={i18n}>
        <ConfigProvider locale={antLocale} theme={adminTheme}>
          {children}
        </ConfigProvider>
      </I18nextProvider>
    </LocaleContext.Provider>
  )
}

export function useLocale(): LocaleContextValue {
  const value = useContext(LocaleContext)
  if (!value) throw new Error('LOCALE_PROVIDER_MISSING')
  return value
}

export { i18n }
