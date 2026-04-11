export default function ProcessTable({ processes }) {
  if (!processes || processes.length === 0) return <div className="text-slate-400 text-sm">Нет данных о процессах</div>;

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm text-left text-slate-300">
        <thead className="text-xs text-slate-400 uppercase bg-slate-900/50 border-b border-slate-700/50">
          <tr>
            <th className="px-4 py-3 font-medium">PID</th>
            <th className="px-4 py-3 font-medium">Процесс</th>
            <th className="px-4 py-3 font-medium">CPU %</th>
            <th className="px-4 py-3 font-medium">RAM MB</th>
          </tr>
        </thead>
        <tbody>
          {processes.map((proc, idx) => (
            <tr key={proc.pid} className="border-b border-slate-700/30 hover:bg-slate-750/50 transition-colors">
              <td className="px-4 py-2.5 text-slate-500">{proc.pid}</td>
              <td className="px-4 py-2.5 font-medium text-slate-200">{proc.name}</td>
              <td className="px-4 py-2.5">
                <div className="flex items-center gap-2">
                  <span className="w-8">{proc.cpu.toFixed(1)}</span>
                  {/* Мини-бар прямо в таблице */}
                  <div className="w-16 bg-slate-700 h-1.5 rounded-full overflow-hidden hidden sm:block">
                    <div 
                      className={`h-full ${proc.cpu > 50 ? 'bg-red-500' : proc.cpu > 20 ? 'bg-amber-500' : 'bg-blue-500'}`} 
                      style={{ width: `${Math.min(proc.cpu, 100)}%` }} 
                    />
                  </div>
                </div>
              </td>
              <td className="px-4 py-2.5">{proc.ram} MB</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}