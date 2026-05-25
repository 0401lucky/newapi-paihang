import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { RankRow } from '../RankRow'

describe('RankRow', () => {
  it('renders gold for rank 1', () => {
    render(<RankRow item={{ rank: 1, user_id: 1, name: 'Alice', value: 1, value_display: '$1' }} />)
    expect(screen.getByText('🥇')).toBeInTheDocument()
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('$1')).toBeInTheDocument()
  })
  it('renders extra model when present', () => {
    render(<RankRow item={{ rank: 5, user_id: 5, name: 'Bob', value: 1, value_display: '$1', extra: { model: 'gpt-4o' } }} />)
    expect(screen.getByText('gpt-4o')).toBeInTheDocument()
  })
})
