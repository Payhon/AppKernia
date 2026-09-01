import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { feedbackSearchSchema, replySchema } from './model'

vi.mock('../auth/store', () => ({ authSession: { adminRequest: (path: string, init?: RequestInit) => fetch(`http://feedback.test/admin-api/v1${path}`, init) } }))
import { feedbackImage, listFeedback, replyFeedback, updateFeedback } from './api'

const server = setupServer()
beforeAll(() => { server.listen({ onUnhandledRequest: 'error' }) })
afterEach(() => { server.resetHandlers() })
afterAll(() => { server.close() })

describe('Feedback management contracts', () => {
  it('keeps App identity in the route and encodes filters without leaking selector state', async () => {
    server.use(http.get('http://feedback.test/admin-api/v1/apps/:app/feedbacks', ({ request, params }) => {
      expect(params['app']).toBe('app/a')
      const query = new URL(request.url).searchParams
      expect(query.get('app_id')).toBeNull()
      expect(query.get('q')).toBe('long text & 中文')
      expect(query.get('status')).toBe('pending')
      return HttpResponse.json({ data: { items: [], total: 0, page: 2, page_size: 20 } })
    }))
    await expect(listFeedback('app/a', { app_id: '00000000-0000-4000-8000-000000000001', q: 'long text & 中文', status: 'pending', page: 2, page_size: 20 })).resolves.toMatchObject({ total: 0 })
  })
  it('does not replay a failed reply; a user retry preserves the idempotency key', async () => {
    const keys: (string | null)[] = []
    server.use(http.post('http://feedback.test/admin-api/v1/apps/a/feedbacks/f/replies', ({ request }) => {
      keys.push(request.headers.get('Idempotency-Key'))
      return keys.length === 1 ? HttpResponse.json({ error: { code: 'COMMON.UNKNOWN', message: 'Unavailable' } }, { status: 503 }) : HttpResponse.json({ data: { id: 'f' } })
    }))
    const input = { body: 'Investigated', status: 'resolved' as const, lock_version: 3 }
    await expect(replyFeedback('a', 'f', input, 'retry-key')).rejects.toMatchObject({ status: 503 })
    expect(keys).toHaveLength(1)
    await expect(replyFeedback('a', 'f', input, 'retry-key')).resolves.toMatchObject({ id: 'f' })
    expect(keys).toEqual(['retry-key', 'retry-key'])
  })
  it('surfaces stale optimistic versions without overwriting the server state', async () => {
    server.use(http.patch('http://feedback.test/admin-api/v1/apps/a/feedbacks/f', () => HttpResponse.json({ error: { code: 'COMMON.CONFLICT', message: 'Stale version' } }, { status: 409 })))
    await expect(updateFeedback('a', 'f', { status: 'resolved', lock_version: 1 })).rejects.toMatchObject({ status: 409, code: 'COMMON.CONFLICT' })
  })
  it('refuses non-image private attachments and propagates permission denial', async () => {
    const path = 'http://feedback.test/admin-api/v1/apps/a/feedbacks/f/attachments/image/content'
    server.use(http.get(path, () => new HttpResponse('<script>bad</script>', { headers: { 'Content-Type': 'text/html' } })))
    await expect(feedbackImage('a', 'f', 'image', new AbortController().signal)).rejects.toThrow('Invalid feedback image response')
    server.use(http.get(path, () => HttpResponse.json({ error: { code: 'COMMON.FORBIDDEN', message: 'Forbidden' } }, { status: 403 })))
    await expect(feedbackImage('a', 'f', 'image', new AbortController().signal)).rejects.toMatchObject({ status: 403 })
  })
  it('validates URL state and rejects blank or oversized replies in either language', () => {
    expect(feedbackSearchSchema.parse({ status: 'assigned' }).status).toBe('')
    expect(feedbackSearchSchema.parse({ created_from: 'invalid' }).created_from).toBeUndefined()
    for (const body of ['  ', '字'.repeat(2001), 'x'.repeat(2001)]) expect(replySchema.safeParse({ body, status: 'resolved' }).success).toBe(false)
    expect(replySchema.parse({ body: ' 已处理 ', status: 'resolved' }).body).toBe('已处理')
  })
})
