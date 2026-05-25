import type {
  ApiEnvelope, LeaderboardResponse, LeaderboardType,
  MetaResponse, RangeType, RankResponse,
} from './types'

const BASE = ''

export class ApiError extends Error {
  constructor(public code: number, msg: string) { super(msg) }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    ...init,
    headers: { 'Accept': 'application/json', ...(init?.headers || {}) },
  })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const env = await res.json() as ApiEnvelope<unknown>
      msg = env.msg || msg
    } catch {}
    throw new ApiError(res.status, msg)
  }
  const env = await res.json() as ApiEnvelope<T>
  if (env.code !== 0) throw new ApiError(env.code, env.msg || 'unknown')
  return env.data
}

export const api = {
  meta: () => request<MetaResponse>('/api/meta'),
  leaderboard: (type: LeaderboardType, range: RangeType, page = 1, pageSize = 20) =>
    request<LeaderboardResponse>(
      `/api/leaderboard/${type}?range=${range}&page=${page}&page_size=${pageSize}`),
  rank: (keyword: string) =>
    request<RankResponse>(`/api/rank/${encodeURIComponent(keyword)}`),
  health: () => request<{ db: boolean; version: string }>('/api/health'),
}
