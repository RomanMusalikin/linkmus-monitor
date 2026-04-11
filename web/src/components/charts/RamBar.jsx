import ProgressBar from '../common/ProgressBar';

export default function RamBar({ used, total }) {
  // Защита от деления на ноль на случай ошибок бэкенда
  const percent = total > 0 ? (used / total) * 100 : 0;

  return (
    <div className="mb-6">
      <div className="flex justify-between text-sm mb-2">
        <span className="text-slate-400">Общая загрузка</span>
        <span className="text-slate-200 font-medium">
          {used.toFixed(1)} / {total.toFixed(1)} GB
        </span>
      </div>
      <ProgressBar value={percent} />
    </div>
  );
}