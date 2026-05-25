import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './embed.html', './admin.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: { primary: '#8b5cf6', secondary: '#ec4899' },
        gold: '#fbbf24', silver: '#a1a1aa', bronze: '#d97706',
      },
      fontFamily: {
        sans: ['ui-sans-serif', 'system-ui', '-apple-system', 'Segoe UI', 'PingFang SC', 'sans-serif'],
        mono: ['ui-monospace', 'SF Mono', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [],
} satisfies Config
