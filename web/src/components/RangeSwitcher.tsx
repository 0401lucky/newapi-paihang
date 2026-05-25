import type { RangeType } from '@/api/types'

const LABELS: Record<RangeType, string> = {
  today: '今日', '7d': '7 天', '30d': '30 天', all: '全部',
}

export function RangeSwitcher({ ranges, current, onChange }: {
  ranges: RangeType[]; current: RangeType; onChange: (r: RangeType) => void
}) {
  return (
    <div className="inline-flex rounded-lg bg-white/50 border border-zinc-200/60 p-0.5">
      {ranges.map(r => {
        const active = r === current
        return (
          <button
            key={r}
            onClick={() => onChange(r)}
            className={`px-3 py-1 text-xs font-semibold rounded-md transition-colors ${
              active ? 'bg-white shadow text-zinc-900' : 'text-zinc-500 hover:text-zinc-700'
            }`}
            aria-pressed={active}
          >{LABELS[r]}</button>
        )
      })}
    </div>
  )
}
