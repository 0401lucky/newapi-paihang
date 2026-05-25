import { useMemo } from 'react'

function avatarStyle(userId: number) {
  const hue = (userId * 137.508) % 360
  return {
    background: `linear-gradient(135deg, hsl(${hue}, 70%, 60%), hsl(${(hue+40)%360}, 70%, 50%))`,
  }
}

export function Avatar({ userId, name, size = 32 }: { userId: number; name: string; size?: number }) {
  const style = useMemo(() => ({
    ...avatarStyle(userId),
    width: size, height: size,
    fontSize: Math.max(11, size * 0.4),
  }), [userId, size])
  const letter = (name?.trim()?.[0] ?? '?').toUpperCase()
  return (
    <div
      role="img"
      aria-label={name}
      className="flex-shrink-0 rounded-full flex items-center justify-center text-white font-semibold select-none"
      style={style}
    >{letter}</div>
  )
}
