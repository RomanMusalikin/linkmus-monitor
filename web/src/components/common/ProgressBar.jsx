export default function ProgressBar({ value }) {
  // Защита от кривых данных: ограничиваем от 0 до 100
  const safeValue = Math.min(Math.max(value, 0), 100);
  
  // Логика цветов
  const colorClass = safeValue > 85 ? 'bg-red-500' : safeValue > 60 ? 'bg-amber-500' : 'bg-green-500';

  return (
    <div className="w-full bg-slate-700 h-1.5 rounded-full overflow-hidden">
      <div 
        className={`${colorClass} h-full transition-all duration-500`} 
        style={{ width: `${safeValue}%` }} 
      />
    </div>
  );
}