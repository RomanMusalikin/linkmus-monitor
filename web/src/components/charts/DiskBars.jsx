import ProgressBar from '../common/ProgressBar';

export default function DiskBars({ disks }) {
  return (
    <div className="space-y-4">
      {disks.map((disk, idx) => {
        const unit = disk.unit || 'GB';
        const percent = unit === '%'
          ? disk.used
          : (disk.used / disk.total) * 100;
        return (
          <div key={idx} className="bg-slate-900/50 p-3 rounded-lg border border-slate-700/50">
            <div className="flex justify-between text-sm mb-2">
              <span className="text-slate-200 font-medium">{disk.mount}</span>
              <div className="text-slate-400">
                {unit === '%' ? (
                  <span className="text-slate-100">{disk.used.toFixed(1)}%</span>
                ) : (
                  <><span className="text-slate-100">{disk.used}</span> / {disk.total} {unit}</>
                )}
                <span className="ml-3 inline-block w-8 text-right font-semibold text-slate-300">
                  {percent.toFixed(0)}%
                </span>
              </div>
            </div>
            <ProgressBar value={percent} />
          </div>
        );
      })}
    </div>
  );
}