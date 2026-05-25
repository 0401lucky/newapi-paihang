import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { LeaderboardType, RangeType } from '@/api/types'

export function useLeaderboard(
  type: LeaderboardType, range: RangeType, page = 1, pageSize = 20, enabled = true,
) {
  return useQuery({
    queryKey: ['lb', type, range, page, pageSize],
    queryFn: () => api.leaderboard(type, range, page, pageSize),
    staleTime: 60_000,
    refetchInterval: 60_000,
    retry: 3,
    enabled,
  })
}
