import { createApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'
import i18next from 'i18next'

import enCatalog from '../locales/en-US/openapi.json'
import enReferenceCatalog from '../locales/en-US/api_reference.json'
import zhCatalog from '../locales/zh-CN/openapi.json'
import zhReferenceCatalog from '../locales/zh-CN/api_reference.json'
import { dismissOpenApiNotice, isOpenApiNoticeDismissed, resolveOpenApiLocale, scalarLocale } from './config'
import { createOpenApiFetch } from './reference'
import './styles.css'

function elementById(id: string): HTMLElement {
  const element = document.getElementById(id)
  if (!element) throw new Error(`OPENAPI_ELEMENT_MISSING:${id}`)
  return element
}

function sessionStorageOrNull(): Storage | null {
  try {
    return window.sessionStorage
  } catch {
    return null
  }
}

async function bootstrap(): Promise<void> {
  const locale = resolveOpenApiLocale(window.location.search, window.navigator.language)
  const referenceCatalog = locale === 'en-US' ? enReferenceCatalog : zhReferenceCatalog
  const docsI18n = i18next.createInstance()
  await docsI18n.init({
    fallbackLng: 'zh-CN',
    interpolation: { escapeValue: false, prefix: '{', suffix: '}' },
    keySeparator: false,
    lng: locale,
    resources: {
      'en-US': { translation: enCatalog },
      'zh-CN': { translation: zhCatalog },
    },
  })

  document.documentElement.lang = locale
  document.documentElement.dir = 'ltr'
  document.title = docsI18n.t('openapi.title')
  elementById('ak-openapi-skip-link').textContent = docsI18n.t('openapi.skip_to_reference')
  elementById('ak-openapi-notice-title').textContent = docsI18n.t('openapi.notice.title')
  elementById('ak-openapi-notice-body').textContent = docsI18n.t('openapi.notice.body')
  const notice = elementById('ak-openapi-notice')
  const noticeClose = elementById('ak-openapi-notice-close')
  noticeClose.setAttribute('aria-label', docsI18n.t('openapi.notice.close'))
  const noticeStorage = sessionStorageOrNull()
  const hideNotice = () => {
    notice.hidden = true
    dismissOpenApiNotice(noticeStorage)
  }
  noticeClose.addEventListener('click', hideNotice)
  if (isOpenApiNoticeDismissed(noticeStorage)) notice.hidden = true

  const hardenScalarDom = () => {
    document.querySelectorAll<HTMLButtonElement>('.scalar-code-copy:not([aria-label])').forEach((button) => {
      button.setAttribute('aria-label', docsI18n.t('openapi.copy_code'))
    })
    document.querySelectorAll<HTMLElement>('.scalar-mcp-layer').forEach((element) => {
      element.remove()
    })
    document.querySelectorAll<HTMLElement>('ul[aria-label]').forEach((element) => {
      const style = window.getComputedStyle(element)
      if ((style.overflowY === 'auto' || style.overflowY === 'scroll') && !element.hasAttribute('tabindex')) {
        element.tabIndex = 0
      }
    })
  }
  const accessibilityObserver = new MutationObserver(hardenScalarDom)
  accessibilityObserver.observe(elementById('openapi-reference'), { childList: true, subtree: true })

  createApiReference('#openapi-reference', {
    agent: { disabled: true },
    customFetch: createOpenApiFetch(locale, referenceCatalog),
    defaultOpenAllTags: false,
    defaultOpenFirstTag: false,
    documentDownloadType: 'direct',
    hideDarkModeToggle: true,
    layout: 'modern',
    localization: { locale: scalarLocale(locale) },
    operationTitleSource: 'summary',
    persistAuth: false,
    showDeveloperTools: 'never',
    telemetry: false,
    theme: 'default',
    url: '/openapi/openapi.yaml',
    withDefaultFonts: false,
  })
  hardenScalarDom()
}

void bootstrap().catch(() => {
  const error = document.getElementById('ak-openapi-error')
  if (!error) return
  const locale = resolveOpenApiLocale(window.location.search, window.navigator.language)
  error.textContent = locale === 'en-US'
    ? enCatalog['openapi.load_error']
    : zhCatalog['openapi.load_error']
  error.hidden = false
})
