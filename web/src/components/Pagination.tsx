export function Pagination({ page, pageSize, total, onChange }: {
  page: number; pageSize: number; total: number; onChange: (p: number) => void
}) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  if (totalPages <= 1) return null
  return (
    <div className="flex items-center justify-center gap-2 mt-4 text-sm">
      <button disabled={page <= 1} onClick={() => onChange(page - 1)}
        className="px-3 py-1 rounded-md bg-white/70 border border-zinc-200 disabled:opacity-40 hover:bg-white">上一页</button>
      <span className="text-zinc-600">第 {page} / {totalPages} 页</span>
      <button disabled={page >= totalPages} onClick={() => onChange(page + 1)}
        className="px-3 py-1 rounded-md bg-white/70 border border-zinc-200 disabled:opacity-40 hover:bg-white">下一页</button>
    </div>
  )
}
