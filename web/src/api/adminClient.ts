const TOKEN_KEY = 'admin_token'

function token() { return localStorage.getItem(TOKEN_KEY) ?? '' }
function setToken(t: string) { localStorage.setItem(TOKEN_KEY, t) }
function clearToken() { localStorage.removeItem(TOKEN_KEY) }

async function authed<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      Authorization: 'Bearer ' + token(),
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
  })
  if (res.status === 401) { clearToken(); throw new Error('未授权，请重新登录') }
  const env = await res.json()
  if (env.code !== 0) throw new Error(env.msg || '请求失败')
  return env.data
}

export const adminApi = {
  token, setToken, clearToken,
  stats: () => authed<{ cache_hits: number; cache_misses: number; cache_size: number; hit_rate: number }>('/admin/stats'),
  clearCache: (prefix = '') => authed<{ cleared: number }>(`/admin/cache/clear?prefix=${encodeURIComponent(prefix)}`, { method: 'POST' }),
  getHidden: () => authed<{ env: number[]; admin: number[] }>('/admin/hidden-users'),
  addHidden: (userId: number) => authed('/admin/hidden-users', { method: 'POST', body: JSON.stringify({ user_id: userId }) }),
  removeHidden: (userId: number) => authed(`/admin/hidden-users/${userId}`, { method: 'DELETE' }),
}
