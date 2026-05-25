import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, beforeEach } from 'vitest'
import { useUrlState } from '../useUrlState'

describe('useUrlState', () => {
  beforeEach(() => { window.history.pushState({}, '', '/') })

  it('reads initial from URL', () => {
    window.history.pushState({}, '', '/?tab=foodie')
    const { result } = renderHook(() => useUrlState({ tab: 'rich', range: '7d' }))
    expect(result.current[0].tab).toBe('foodie')
  })

  it('updates URL on patch', () => {
    const { result } = renderHook(() => useUrlState({ tab: 'rich' }))
    act(() => result.current[1]({ tab: 'foodie' }))
    expect(window.location.search).toContain('tab=foodie')
  })
})
