import { Avatar } from './Avatar'
import type { LeaderboardItem } from '@/api/types'

function rowClass(rank: number) {
  if (rank === 1) return 'glass-row-gold'
  if (rank === 2) return 'glass-row-silver'
  if (rank === 3) return 'glass-row-bronze'
  return 'glass-row'
}

function rankIcon(rank: number) {
  if (rank === 1) return '🥇'
  if (rank === 2) return '🥈'
  if (rank === 3) return '🥉'
  return String(rank).padStart(2, '0')
}

export function RankRow({ item, compact = false }: { item: LeaderboardItem; compact?: boolean }) {
  const padY = compact ? 'py-1.5' : 'py-2.5'
  return (
    <div
      className={`flex items-center gap-3 px-3 ${padY} rounded-xl ${rowClass(item.rank)} transition-transform duration-150 hover:scale-[1.01]`}
      style={{ contentVisibility: 'auto', containIntrinsicSize: compact ? '40px' : '56px' }}
    >
      <div className="w-7 text-center font-bold text-zinc-800 text-sm tabular-nums">
        {rankIcon(item.rank)}
      </div>
      <Avatar userId={item.user_id} name={item.name} size={compact ? 24 : 30} />
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-zinc-900 truncate">{item.name}</div>
        {item.extra?.model && (
          <div className="text-[10px] text-zinc-500 truncate">{item.extra.model}</div>
        )}
      </div>
      <div className="text-sm font-bold gradient-text tabular-nums whitespace-nowrap">
        {item.value_display}
      </div>
    </div>
  )
}
