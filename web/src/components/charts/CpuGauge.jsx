import { PieChart, Pie, Cell, ResponsiveContainer } from 'recharts';

export default function CpuGauge({ value }) {
  // Данные: Занято и Свободно
  const data = [
    { name: 'Used', value: value },
    { name: 'Free', value: 100 - value }
  ];

  // Цвет меняется от нагрузки
  const activeColor = value > 85 ? '#ef4444' : value > 60 ? '#f59e0b' : '#22c55e';
  const COLORS = [activeColor, '#334155'];

  return (
    <div className="h-40 w-full relative">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={45}
            outerRadius={65}
            startAngle={90}
            endAngle={-270}
            dataKey="value"
            stroke="none"
          >
            {data.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
            ))}
          </Pie>
        </PieChart>
      </ResponsiveContainer>
      {/* Текст в центре бублика */}
      <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
        <span className="text-2xl font-bold text-slate-100">{value}%</span>
      </div>
    </div>
  );
}