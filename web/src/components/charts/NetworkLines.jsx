import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend } from 'recharts';

export default function NetworkLines({ data }) {
  if (!data || data.length === 0) {
    return (
      <div className="h-48 flex items-center justify-center text-slate-500 text-sm">
        Нет данных о трафике
      </div>
    );
  }

  return (
    <div className="h-48 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
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
          <XAxis dataKey="time" stroke="#475569" fontSize={11} tickLine={false} axisLine={false} />
          <YAxis stroke="#475569" fontSize={11} tickLine={false} axisLine={false}
            tickFormatter={(v) => v >= 1024 ? `${(v/1024).toFixed(0)}K` : `${v}`} />
          <Tooltip
            contentStyle={{ backgroundColor: '#0f172a', borderColor: '#334155', color: '#f1f5f9', borderRadius: '0.5rem', fontSize: '12px' }}
            formatter={(val, name) => [`${(val/1024).toFixed(1)} KB/s`, name]}
          />
          <Legend verticalAlign="top" height={28} wrapperStyle={{ fontSize: '11px', color: '#94a3b8' }} />
          <Area type="monotone" dataKey="recv" name="↓ Входящий" stroke="#06b6d4" strokeWidth={1.5}
            fillOpacity={1} fill="url(#recvGrad)" dot={false} isAnimationActive={false} />
          <Area type="monotone" dataKey="sent" name="↑ Исходящий" stroke="#3b82f6" strokeWidth={1.5}
            fillOpacity={1} fill="url(#sentGrad)" dot={false} isAnimationActive={false} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
