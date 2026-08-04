import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useAuthStore } from '../features/auth/store'
import { useLocale, type AdminLocale } from '../shared/i18n'

export function LocaleSwitcher() {
  const { t } = useTranslation()
  const { locale, setLocale } = useLocale()
  const authenticated = useAuthStore((state) => state.status === 'authenticated')
  const updateLocale = useAuthStore((state) => state.updateLocale)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState(false)

  const changeLocale = async (nextLocale: AdminLocale) => {
    const previousLocale = locale
    setSaveError(false)
    await setLocale(nextLocale)
    if (!authenticated) return
    setSaving(true)
    try {
      await updateLocale(nextLocale)
    } catch {
      await setLocale(previousLocale)
      setSaveError(true)
    } finally {
      setSaving(false)
    }
  }

  return (
    <span className="ak-locale-control">
      <select
        aria-label={t('profile.language.title')}
        aria-describedby={saveError ? 'ak-locale-save-error' : undefined}
        className="ak-locale-switcher"
        disabled={saving}
        onChange={(event) => { void changeLocale(event.currentTarget.value as AdminLocale) }}
        value={locale}
      >
        <option value="zh-CN">{t('common.language.zh-CN')}</option>
        <option value="en-US">{t('common.language.en-US')}</option>
      </select>
      {saveError ? <span className="ak-locale-error" id="ak-locale-save-error" role="alert">{t('profile.language.save_error')}</span> : null}
    </span>
  )
}
