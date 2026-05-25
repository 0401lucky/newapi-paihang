import { useEffect, useState, type FormEvent } from 'react'
import { adminApi } from '@/api/adminClient'

interface Stats {
  cache_hits: number
  cache_misses: number
  cache_size: number
  hit_rate: number
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md bg-white/70 p-2 text-center">
      <div className="text-xs text-zinc-500">{label}</div>
      <div className="text-lg font-bold gradient-text">{value}</div>
    </div>
  )
}

function LoginForm({ onLogin }: { onLogin: () => void }) {
  const [t, setT] = useState('')
  const [err, setErr] = useState<string | null>(null)
  async function submit(e: FormEvent) {
    e.preventDefault()
    adminApi.setToken(t)
    try {
      await adminApi.stats()
      onLogin()
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : String(e))
    }
  }
  return (
    <div className="min-h-screen flex items-center justify-center page-gradient">
      <form onSubmit={submit} className="glass rounded-2xl p-6 w-80 space-y-3">
        <h1 className="text-lg font-bold gradient-text">🛠 后台登录</h1>
        <input type="password" value={t} onChange={e => setT(e.target.value)}
          placeholder="输入 ADMIN_TOKEN"
          className="w-full px-3 py-2 rounded-md bg-white border border-zinc-200 text-sm" />
        {err && <div className="text-xs text-red-600">{err}</div>}
        <button className="w-full py-2 rounded-md bg-brand-primary text-white text-sm font-semibold">登录</button>
      </form>
    </div>
  )
}

function Dashboard({ onLogout }: { onLogout: () => void }) {
  const [stats, setStats] = useState<Stats | null>(null)
  const [hidden, setHidden] = useState<{ env: number[]; admin: number[] }>({ env: [], admin: [] })
  const [newId, setNewId] = useState('')
  const [msg, setMsg] = useState<string | null>(null)

  async function refresh() {
    setStats(await adminApi.stats())
    setHidden(await adminApi.getHidden())
  }
  useEffect(() => { refresh() }, [])

  async function doClear() {
    const { cleared } = await adminApi.clearCache()
    setMsg(`已清空缓存（${cleared} 项）`)
    refresh()
  }
  async function doAdd() {
    const id = parseInt(newId, 10)
    if (!id) return
    await adminApi.addHidden(id)
    setNewId('')
    refresh()
  }
  async function doRemove(id: number) {
    await adminApi.removeHidden(id)
    refresh()
  }

  return (
    <div className="min-h-screen page-gradient p-6">
      <div className="max-w-2xl mx-auto space-y-4">
        <header className="flex items-center justify-between">
          <h1 className="text-xl font-bold gradient-text">🛠 排行榜后台</h1>
          <button onClick={onLogout} className="text-xs px-3 py-1 rounded-md bg-white/70 border">退出</button>
        </header>

        {msg && <div className="glass rounded-md p-2 text-sm text-emerald-700">{msg}</div>}

        <section className="glass rounded-2xl p-4">
          <h2 className="font-semibold mb-2">系统状态</h2>
          {stats ? (
            <div className="grid grid-cols-3 gap-3 text-sm">
              <Metric label="缓存命中率" value={`${(stats.hit_rate * 100).toFixed(1)}%`} />
              <Metric label="命中次数" value={String(stats.cache_hits)} />
              <Metric label="缓存项" value={String(stats.cache_size)} />
            </div>
          ) : '加载中...'}
        </section>

        <section className="glass rounded-2xl p-4">
          <h2 className="font-semibold mb-2">缓存管理</h2>
          <button onClick={doClear}
            className="px-3 py-1.5 text-sm rounded-md bg-brand-primary text-white">清空全部</button>
        </section>

        <section className="glass rounded-2xl p-4">
          <h2 className="font-semibold mb-3">隐藏用户</h2>
          <div className="text-xs text-zinc-500 mb-2">
            环境变量隐藏（只读）：{hidden.env.length === 0 ? '无' : hidden.env.join(', ')}
          </div>
          <div className="flex gap-2 mb-3">
            <input value={newId} onChange={e => setNewId(e.target.value)}
              placeholder="输入 user_id"
              className="flex-1 px-3 py-1.5 rounded-md bg-white border border-zinc-200 text-sm" />
            <button onClick={doAdd}
              className="px-3 py-1.5 rounded-md bg-brand-primary text-white text-sm font-semibold">+ 添加</button>
          </div>
          <div className="flex flex-wrap gap-2">
            {hidden.admin.length === 0 && <span className="text-xs text-zinc-400">暂无临时隐藏</span>}
            {hidden.admin.map(id => (
              <span key={id} className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-white border border-zinc-200 text-xs">
                {id}
                <button onClick={() => doRemove(id)} className="text-red-500 hover:text-red-700">×</button>
              </span>
            ))}
          </div>
        </section>
      </div>
    </div>
  )
}

export function AdminApp() {
  const [authed, setAuthed] = useState(!!adminApi.token())
  if (!authed) return <LoginForm onLogin={() => setAuthed(true)} />
  return <Dashboard onLogout={() => { adminApi.clearToken(); setAuthed(false) }} />
}
