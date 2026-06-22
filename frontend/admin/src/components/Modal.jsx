export default function Modal({ title, onClose, children, devLabel }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4">
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-base font-semibold text-gray-800">{title}</h2>
          <div className="flex items-center gap-2">
            {devLabel && <span className="text-[10px] font-mono bg-black/70 text-white px-1.5 py-0.5 rounded select-none">{devLabel}</span>}
            <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl leading-none">&times;</button>
          </div>
        </div>
        <div className="px-6 py-4">{children}</div>
      </div>
    </div>
  )
}
