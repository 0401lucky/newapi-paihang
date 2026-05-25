import { useState, type FormEvent } from 'react'
import { api } from '@/api/client'
import type { PersonalRank } from '@/api/types'

export function PersonalRankWidget() {
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(false)
  const [results, setResults] = useState<PersonalRank[] | null>(null)
  const [notFound, setNotFound] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!keyword.trim()) return
    setLoading(true); setError(null); setResults(null); setNotFound(false)
    try {
      const data = await api.rank(keyword.trim())
      if (!data.found) setNotFound(true)
      else setResults(data.results ?? [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '查询失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="glass rounded-xl p-4">
      <form onSubmit={onSubmit} className="flex gap-2">
        <input
          value={keyword} onChange={e => setKeyword(e.target.value)}
          placeholder="输入用户名或昵称查我的排名"
          className="flex-1 px-3 py-1.5 rounded-md bg-white/80 border border-zinc-200 text-sm focus:outline-none focus:ring-2 focus:ring-brand-primary/40"
          maxLength={50}
        />
        <button disabled={loading || !keyword.trim()}
          className="px-3 py-1.5 rounded-md bg-brand-primary text-white text-sm font-semibold disabled:opacity-50">
          {loading ? '查询中…' : '查我在哪'}
        </button>
      </form>
      {error && <div className="mt-3 text-xs text-red-600">{error}</div>}
      {notFound && <div className="mt-3 text-xs text-zinc-500">没找到匹配的用户</div>}
      {results && results.map(r => (
        <div key={r.user_id} className="mt-3 text-sm">
          <div className="font-semibold mb-1">{r.name} <span className="text-xs text-zinc-400">#{r.user_id}</span></div>
          <div className="flex flex-wrap gap-2">
            {Object.entries(r.ranks).length === 0 && (
              <span className="text-xs text-zinc-400">该用户还没上任何榜</span>
            )}
            {Object.entries(r.ranks).map(([type, entry]) => (
              <span key={type} className="text-xs px-2 py-1 rounded-md bg-white/80 border border-zinc-200">
                {type} · #{entry.rank}
              </span>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
