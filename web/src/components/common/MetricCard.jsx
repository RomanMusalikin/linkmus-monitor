export default function MetricCard({ title, icon: Icon, children, className = "" }) {
  return (
    <div className={`bg-slate-800 rounded-xl border border-slate-700 p-5 ${className}`}>
      <div className="flex items-center gap-2 mb-4 border-b border-slate-700/50 pb-3">
        {Icon && <Icon className="w-5 h-5 text-blue-500" />}
        <h2 className="text-lg font-semibold text-slate-100">{title}</h2>
      </div>
      <div>
        {children}
      </div>
    </div>
  );
}