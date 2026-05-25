import { RankRow } from './RankRow'
import { LoadingSkeleton } from './LoadingSkeleton'
import { ErrorState } from './ErrorState'
import { EmptyState } from './EmptyState'
import type { LeaderboardResponse } from '@/api/types'

export function LeaderboardCard({
  title, emoji, description, data, isLoading, isError, onRetry, compact = false,
}: {
  title: string; emoji: string; description?: string
  data?: LeaderboardResponse
  isLoading: boolean; isError: boolean
  onRetry?: () => void
  compact?: boolean
}) {
  return (
    <div className="glass rounded-2xl p-4">
      <header className="mb-3">
        <h2 className="text-base font-bold gradient-text"><span className="mr-1">{emoji}</span>{title}</h2>
        {description && <p className="text-xs text-zinc-500 mt-0.5">{description}</p>}
        {data?.stale && <p className="text-[10px] text-amber-600 mt-0.5">⚠ 数据可能稍有滞后</p>}
      </header>
      {isLoading && <LoadingSkeleton rows={compact ? 5 : 10} />}
      {isError && <ErrorState onRetry={onRetry} />}
      {!isLoading && !isError && data && (
        data.list.length === 0
          ? <EmptyState message="暂无数据" />
          : <div className="space-y-1.5">
              {data.list.map(item => <RankRow key={item.user_id} item={item} compact={compact} />)}
            </div>
      )}
    </div>
  )
}
