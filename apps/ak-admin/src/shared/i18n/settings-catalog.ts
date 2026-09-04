import enSettings from '../../locales/en-US/settings.json'
import zhSettings from '../../locales/zh-CN/settings.json'
import { registerAdminTranslationCatalog } from './index'

registerAdminTranslationCatalog('zh-CN', zhSettings)
registerAdminTranslationCatalog('en-US', enSettings)
