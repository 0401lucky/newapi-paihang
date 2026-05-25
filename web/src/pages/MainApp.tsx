import { useMeta } from '@/hooks/useMeta'
import { useLeaderboard } from '@/hooks/useLeaderboard'
import { useUrlState } from '@/hooks/useUrlState'
import { TabBar } from '@/components/TabBar'
import { RangeSwitcher } from '@/components/RangeSwitcher'
import { LeaderboardCard } from '@/components/LeaderboardCard'
import { Pagination } from '@/components/Pagination'
import { PersonalRankWidget } from '@/components/PersonalRankWidget'
import type { LeaderboardType, RangeType } from '@/api/types'

function timeAgo(ts: number) {
  const diff = Math.max(0, Math.floor(Date.now() / 1000 - ts))
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  return `${Math.floor(diff / 86400)} 天前`
}

export function MainApp() {
  const meta = useMeta()
  const [state, setState] = useUrlState({ tab: 'rich', range: '7d', page: '1' })

  const currentTab = state.tab as LeaderboardType
  const currentRange = state.range as RangeType
  const currentPage = parseInt(state.page, 10) || 1

  const tabMeta = meta.data?.leaderboards.find(t => t.type === currentTab)
  const supportedRanges = tabMeta?.ranges ?? (['7d'] as RangeType[])
  const effectiveRange: RangeType = supportedRanges.includes(currentRange) ? currentRange : supportedRanges[0]

  const lb = useLeaderboard(currentTab, effectiveRange, currentPage, 20, !!tabMeta)
  const siteName = meta.data?.embed.site_name ?? 'NewAPI 排行榜'

  return (
    <div className="page-gradient min-h-screen">
      <div className="max-w-3xl mx-auto px-4 py-8">
        <header className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <h1 className="text-2xl font-bold gradient-text">🏆 {siteName}</h1>
          <div className="sm:w-72"><PersonalRankWidget /></div>
        </header>

        {meta.data && (
          <TabBar
            tabs={meta.data.leaderboards}
            current={currentTab}
            onChange={t => setState({ tab: t, page: '1' })}
          />
        )}

        {tabMeta && (
          <div className="my-3 flex items-center justify-between">
            <RangeSwitcher
              ranges={supportedRanges}
              current={effectiveRange}
              onChange={r => setState({ range: r, page: '1' })}
            />
            {lb.data && (
              <span className="text-xs text-zinc-500">
                更新于 {timeAgo(lb.data.updated_at)}
              </span>
            )}
          </div>
        )}

        {tabMeta && (
          <LeaderboardCard
            title={tabMeta.name}
            emoji={tabMeta.emoji}
            description={tabMeta.description}
            data={lb.data}
            isLoading={lb.isLoading}
            isError={lb.isError}
            onRetry={() => lb.refetch()}
          />
        )}

        {lb.data && (
          <Pagination
            page={currentPage}
            pageSize={lb.data.page_size}
            total={lb.data.total}
            onChange={p => setState({ page: String(p) })}
          />
        )}

        <footer className="mt-12 text-center text-xs text-zinc-400">
          基于 NewAPI · 数据每分钟更新
        </footer>
      </div>
    </div>
  )
}
