export function EmptyState({ message = '暂无数据' }: { message?: string }) {
  return (
    <div className="py-12 text-center text-zinc-400 text-sm">
      <div className="text-4xl mb-2 opacity-40">📭</div>
      {message}
    </div>
  )
}
