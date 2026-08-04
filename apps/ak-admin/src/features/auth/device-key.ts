const storageKey = 'ak.admin.device-key.v1'
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

export function readOrCreateAdminDeviceKey(): string {
  try {
    const current = window.localStorage.getItem(storageKey)
    if (current && uuidPattern.test(current)) return current.toLowerCase()
    const next = crypto.randomUUID()
    window.localStorage.setItem(storageKey, next)
    return next
  } catch {
    return crypto.randomUUID()
  }
}
