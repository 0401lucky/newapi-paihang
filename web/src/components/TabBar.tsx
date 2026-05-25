import type { LeaderboardMeta, LeaderboardType } from '@/api/types'

export function TabBar({ tabs, current, onChange }: {
  tabs: LeaderboardMeta[]
  current: LeaderboardType
  onChange: (t: LeaderboardType) => void
}) {
  return (
    <div className="flex gap-1 overflow-x-auto pb-2 -mx-1 px-1 snap-x">
      {tabs.map(t => {
        const active = t.type === current
        return (
          <button
            key={t.type}
            onClick={() => onChange(t.type)}
            className={`snap-start whitespace-nowrap px-3 py-1.5 rounded-lg text-sm font-semibold transition-colors ${
              active
                ? 'bg-gradient-to-br from-brand-primary to-brand-secondary text-white shadow'
                : 'bg-white/60 text-zinc-600 hover:bg-white border border-zinc-200/60'
            }`}
            aria-pressed={active}
          >
            <span className="mr-1">{t.emoji}</span>{t.name}
          </button>
        )
      })}
    </div>
  )
}
