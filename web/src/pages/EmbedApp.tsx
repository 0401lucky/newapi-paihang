import { useState, useEffect, useMemo } from 'react'
import { useMeta } from '@/hooks/useMeta'
import { useLeaderboard } from '@/hooks/useLeaderboard'
import { TabBar } from '@/components/TabBar'
import { LeaderboardCard } from '@/components/LeaderboardCard'
import type { LeaderboardType, RangeType } from '@/api/types'

function readEmbedParams() {
  const p = new URLSearchParams(window.location.search)
  return {
    tabs: (p.get('tabs') || '').split(',').filter(Boolean) as LeaderboardType[],
    range: (p.get('range') || '7d') as RangeType,
    limit: Math.min(parseInt(p.get('limit') || '5', 10) || 5, 10),
    theme: p.get('theme') as 'light' | 'dark' | null,
    site: p.get('site') || '',
  }
}

export function EmbedApp() {
  const params = useMemo(readEmbedParams, [])
  const meta = useMeta()
  const allTabs = meta.data?.leaderboards ?? []
  const tabsToShow = params.tabs.length > 0
    ? allTabs.filter(t => params.tabs.includes(t.type))
    : allTabs.filter(t => (meta.data?.embed.tabs ?? []).includes(t.type))

  const [current, setCurrent] = useState<LeaderboardType>('rich')
  useEffect(() => {
    if (tabsToShow.length > 0) {
      setCurrent(prev =>
        tabsToShow.find(t => t.type === prev) ? prev : tabsToShow[0].type)
    }
  }, [tabsToShow])

  const tabMeta = tabsToShow.find(t => t.type === current)
  const supportedRanges = tabMeta?.ranges ?? (['7d'] as RangeType[])
  const effectiveRange: RangeType = supportedRanges.includes(params.range) ? params.range : supportedRanges[0]
  const lb = useLeaderboard(current, effectiveRange, 1, params.limit, !!tabMeta)
  const siteUrl = params.site || meta.data?.embed.site_url || ''

  useEffect(() => {
    if (params.theme === 'dark') document.documentElement.classList.add('dark')
  }, [params.theme])

  return (
    <div className="page-gradient min-h-screen p-3">
      {tabsToShow.length > 0 && (
        <TabBar tabs={tabsToShow} current={current} onChange={setCurrent} />
      )}
      <div className="mt-2">
        {tabMeta && (
          <LeaderboardCard
            title={tabMeta.name}
            emoji={tabMeta.emoji}
            data={lb.data}
            isLoading={lb.isLoading}
            isError={lb.isError}
            onRetry={() => lb.refetch()}
            compact
          />
        )}
      </div>
      {siteUrl && (
        <div className="mt-3 text-center">
          <a href={siteUrl} target="_top"
            className="text-xs text-brand-primary font-semibold hover:underline">
            查看完整版排行榜 →
          </a>
        </div>
      )}
    </div>
  )
}
