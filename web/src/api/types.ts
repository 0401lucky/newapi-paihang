export type LeaderboardType =
  | 'rich' | 'spender' | 'topup' | 'foodie' | 'loyal'
  | 'gourmet' | 'biteking' | 'tokens' | 'nightowl'

export type RangeType = 'today' | '7d' | '30d' | 'all'

export interface LeaderboardItem {
  rank: number
  user_id: number
  name: string
  value: number
  value_display: string
  extra?: { model?: string; total_calls?: number } | null
}

export interface LeaderboardResponse {
  type: LeaderboardType
  range: RangeType
  total: number
  page: number
  page_size: number
  list: LeaderboardItem[]
  updated_at: number
  cached: boolean
  stale?: boolean
}

export interface LeaderboardMeta {
  type: LeaderboardType
  name: string
  emoji: string
  description: string
  ranges: RangeType[]
}

export interface MetaResponse {
  leaderboards: LeaderboardMeta[]
  embed: { tabs: LeaderboardType[]; site_url: string; site_name: string }
  version: string
}

export interface PersonalEntry { rank: number; value?: number }

export interface PersonalRank {
  user_id: number
  name: string
  ranks: Partial<Record<LeaderboardType, PersonalEntry>>
}

export interface RankResponse {
  found: boolean
  keyword: string
  results?: PersonalRank[]
}

export interface ApiEnvelope<T> {
  code: number
  msg?: string
  data: T
}
