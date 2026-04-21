import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend } from 'recharts';

export default function NetworkLines({ data }) {
  if (!data || data.length === 0) {
    return (
      <div className="h-48 flex items-center justify-center text-slate-500 text-sm">
        Нет данных о трафике
      </div>
    );
  }

  const tickInterval = Math.max(0, Math.floor(data.length / 10) - 1);
  const fmt = data.length > 36 ? (t) => t.slice(0, 5) : undefined;

  return (
    <div className="h-52 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 5, right: 8, left: -10, bottom: 28 }}>
          <defs>
            <linearGradient id="recvGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#06b6d4" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#06b6d4" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="sentGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
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
            tickFormatter={(v) =>
              v >= 1024 * 1024 ? `${(v / 1024 / 1024).toFixed(0)}M`
              : v >= 1024 ? `${(v / 1024).toFixed(0)}K`
              : `${v}`
            }
          />
          <Tooltip
            contentStyle={{ backgroundColor: '#0f172a', borderColor: '#334155', color: '#f1f5f9', borderRadius: '0.5rem', fontSize: '12px' }}
            formatter={(val, name) => {
              if (val == null) return ['—', name];
              const kb = val / 1024;
              return [kb >= 1024 ? `${(kb / 1024).toFixed(2)} MB/s` : `${kb.toFixed(1)} KB/s`, name];
            }}
            labelStyle={{ color: '#94a3b8', marginBottom: '2px' }}
          />
          <Legend verticalAlign="top" height={24} wrapperStyle={{ fontSize: '11px', color: '#94a3b8' }} />
          <Area type="monotone" dataKey="recv" name="↓ Входящий" stroke="#06b6d4" strokeWidth={1.5}
            fillOpacity={1} fill="url(#recvGrad)" dot={false} isAnimationActive={false} connectNulls={false} />
          <Area type="monotone" dataKey="sent" name="↑ Исходящий" stroke="#3b82f6" strokeWidth={1.5}
            fillOpacity={1} fill="url(#sentGrad)" dot={false} isAnimationActive={false} connectNulls={false} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
