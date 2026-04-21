import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip } from 'recharts';

export default function CpuHistory({ data }) {
  const chartData = (data || []).map(p => ({
    cpu: p.value != null ? p.value : (p.cpu != null ? p.cpu : null),
    time: p.time ?? '',
  }));

  const tickInterval = Math.max(0, Math.floor(chartData.length / 10) - 1);
  const fmt = chartData.length > 36 ? (t) => t.slice(0, 5) : undefined;

  return (
    <div className="h-64 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} margin={{ top: 10, right: 8, left: -20, bottom: 28 }}>
          <defs>
            <linearGradient id="cpuGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.4} />
              <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#334155" vertical={false} />
          <XAxis
            dataKey="time"
            stroke="#475569"
            fontSize={10}
            tickLine={false}
            axisLine={false}
            interval={tickInterval}
            tickFormatter={fmt}
            angle={-35}
            textAnchor="end"
            dy={4}
          />
          <YAxis
            stroke="#475569"
            fontSize={11}
            tickLine={false}
            axisLine={false}
            domain={[0, 100]}
            tickFormatter={(v) => `${v}%`}
          />
          <Tooltip
            contentStyle={{ backgroundColor: '#1e293b', borderColor: '#475569', color: '#f1f5f9', fontSize: '12px', borderRadius: '0.5rem' }}
            itemStyle={{ color: '#3b82f6' }}
            formatter={(val) => val != null ? [`${val}%`, 'CPU'] : ['—', 'CPU']}
            labelStyle={{ color: '#94a3b8', marginBottom: '2px' }}
          />
          <Area
            type="monotone"
            dataKey="cpu"
            stroke="#3b82f6"
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#cpuGradient)"
            dot={false}
            isAnimationActive={false}
            connectNulls={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
