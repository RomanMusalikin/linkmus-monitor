import { AlertTriangle } from 'lucide-react';

export default function FsrmQuota({ path, usedGB, totalGB, thresholdPercent = 85 }) {
  const percent = (usedGB / totalGB) * 100;
  const isWarning = percent >= thresholdPercent;

  return (
    <div className="bg-slate-900/50 p-4 rounded-lg border border-slate-700/50 relative overflow-hidden">
      {/* Линия порога */}
      <div 
        className="absolute top-0 bottom-0 border-r border-dashed border-red-500/50 z-0"
        style={{ left: `${thresholdPercent}%` }}
        title={`Порог уведомления: ${thresholdPercent}%`}
      />
      
      <div className="relative z-10">
        <div className="flex justify-between items-center mb-2">
          <div className="text-sm text-slate-300 font-medium">Квота: <span className="text-blue-400">{path}</span></div>
          <div className="text-sm text-slate-100 font-bold">{usedGB} / {totalGB} GB ({percent.toFixed(0)}%)</div>
        </div>
        
        <div className="w-full bg-slate-700 h-2.5 rounded-full overflow-hidden mb-3">
          <div 
            className={`h-full transition-all ${isWarning ? 'bg-red-500' : 'bg-blue-500'}`}
            style={{ width: `${Math.min(percent, 100)}%` }}
          />
        </div>

        {isWarning && (
          <div className="flex items-center gap-2 text-xs text-amber-400 bg-amber-500/10 p-2 rounded">
            <AlertTriangle className="w-4 h-4" />
            <span>Внимание: Превышен порог квоты ({thresholdPercent}%)</span>
          </div>
        )}
      </div>
    </div>
  );
}