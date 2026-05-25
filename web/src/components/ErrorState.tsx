export function ErrorState({ onRetry, message = '数据加载失败，请稍后再试' }: {
  onRetry?: () => void; message?: string
}) {
  return (
    <div className="py-12 text-center">
      <div className="text-4xl mb-2 opacity-50">⚠️</div>
      <div className="text-sm text-zinc-600 mb-3">{message}</div>
      {onRetry && (
        <button onClick={onRetry}
          className="text-xs px-3 py-1.5 rounded-md bg-brand-primary text-white hover:bg-brand-primary/90">
          重试
        </button>
      )}
    </div>
  )
}
