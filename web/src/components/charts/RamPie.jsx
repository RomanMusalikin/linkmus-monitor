import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';

export default function RamPie({ used, cached, free }) {
  const data = [
    { name: 'Использовано', value: used, color: '#3b82f6' }, // Синий
    { name: 'Кэш/Буферы', value: cached, color: '#f59e0b' }, // Желтый
    { name: 'Свободно', value: free, color: '#334155' }      // Серый
  ];

  return (
    <div className="h-48 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={55}
            outerRadius={75}
            paddingAngle={2}
            dataKey="value"
            stroke="none"
          >
            {data.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.color} />
            ))}
          </Pie>
          <Tooltip 
            contentStyle={{ backgroundColor: '#1e293b', borderColor: '#475569', borderRadius: '0.5rem' }}
            itemStyle={{ color: '#f1f5f9' }}
            formatter={(value) => [`${value} GB`, '']}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}