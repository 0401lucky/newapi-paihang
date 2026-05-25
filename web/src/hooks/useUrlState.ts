import { useCallback, useEffect, useState } from 'react'

export function useUrlState<T extends Record<string, string>>(defaults: T): [T, (patch: Partial<T>) => void] {
  const [state, setState] = useState<T>(() => readFromUrl(defaults))

  useEffect(() => {
    const onPop = () => setState(readFromUrl(defaults))
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [defaults])

  const update = useCallback((patch: Partial<T>) => {
    setState(prev => {
      const next = { ...prev, ...patch }
      const url = new URL(window.location.href)
      Object.entries(next).forEach(([k, v]) => {
        if (v == null || v === '') url.searchParams.delete(k)
        else url.searchParams.set(k, String(v))
      })
      window.history.pushState({}, '', url)
      return next
    })
  }, [])

  return [state, update]
}

function readFromUrl<T extends Record<string, string>>(defaults: T): T {
  const params = new URLSearchParams(window.location.search)
  const out = { ...defaults }
  for (const k of Object.keys(defaults)) {
    const v = params.get(k)
    if (v != null) (out as Record<string, string>)[k] = v
  }
  return out
}
