import { CheckCircle2, AlertTriangle, XCircle } from 'lucide-react';

export default function ServiceStatus({ services }) {
  if (!services || services.length === 0) return null;

  const getStatusIcon = (state) => {
    switch(state) {
      case 'Running': return <CheckCircle2 className="w-4 h-4 text-green-500" />;
      case 'Stopped': return <XCircle className="w-4 h-4 text-red-500" />;
      default: return <AlertTriangle className="w-4 h-4 text-amber-500" />;
    }
  };

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
      {services.map((svc, idx) => (
        <div key={idx} className="flex items-center justify-between bg-slate-900/50 p-3 rounded-lg border border-slate-700/50">
          <div className="flex items-center gap-3">
            {getStatusIcon(svc.state)}
            <div>
              <div className="text-sm font-medium text-slate-200">{svc.displayName}</div>
              <div className="text-xs text-slate-500">{svc.name}</div>
            </div>
          </div>
          <span className={`text-xs font-semibold px-2 py-1 rounded-md ${
            svc.state === 'Running' ? 'bg-green-500/10 text-green-400' : 
            svc.state === 'Stopped' ? 'bg-red-500/10 text-red-400' : 'bg-amber-500/10 text-amber-400'
          }`}>
            {svc.state}
          </span>
        </div>
      ))}
    </div>
  );
}