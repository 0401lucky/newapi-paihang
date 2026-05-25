import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'
import { api, ApiError } from '../client'

const server = setupServer(
  http.get('/api/meta', () => HttpResponse.json({
    code: 0, data: { leaderboards: [], embed: { tabs: [], site_url: '', site_name: 'Test' }, version: 'x' },
  })),
  http.get('/api/leaderboard/rich', () => HttpResponse.json({
    code: 0, data: { type: 'rich', range: 'all', total: 1, page: 1, page_size: 5,
      list: [{ rank: 1, user_id: 2, name: 'x', value: 100, value_display: '$100' }],
      updated_at: 0, cached: true },
  })),
  http.get('/api/leaderboard/bad', () => HttpResponse.json({ code: 400, msg: 'invalid type', data: null }, { status: 400 })),
)

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('api client', () => {
  it('fetches meta', async () => {
    const m = await api.meta()
    expect(m.embed.site_name).toBe('Test')
  })

  it('fetches leaderboard', async () => {
    const r = await api.leaderboard('rich', 'all', 1, 5)
    expect(r.list[0].name).toBe('x')
  })

  it('throws ApiError on 400', async () => {
    await expect(api.leaderboard('bad' as any, 'all')).rejects.toBeInstanceOf(ApiError)
  })
})
