import { FullscreenExitOutlined, FullscreenOutlined } from '@ant-design/icons'
import { Button, Tooltip, message } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

export function FullscreenToggle() {
  const { t } = useTranslation()
  const [fullscreen, setFullscreen] = useState(() => Boolean(document.fullscreenElement))
  const supported = document.fullscreenEnabled && typeof document.documentElement.requestFullscreen === 'function'

  useEffect(() => {
    const update = () => { setFullscreen(Boolean(document.fullscreenElement)) }
    document.addEventListener('fullscreenchange', update)
    return () => { document.removeEventListener('fullscreenchange', update) }
  }, [])

  const toggle = async () => {
    try {
      if (document.fullscreenElement) await document.exitFullscreen()
      else await document.documentElement.requestFullscreen()
    } catch {
      void message.error(t('shell.fullscreen_error'))
    }
  }

  const label = t(fullscreen ? 'shell.exit_fullscreen' : 'shell.enter_fullscreen')
  return (
    <Tooltip title={label}>
      <Button
        aria-label={label}
        aria-pressed={fullscreen}
        className="ak-shell-icon-button"
        disabled={!supported}
        icon={fullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
        onClick={() => { void toggle() }}
        type="text"
      />
    </Tooltip>
  )
}
