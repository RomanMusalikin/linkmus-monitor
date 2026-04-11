import { ResponsiveContainer, AreaChart, Area } from 'recharts';

export default function Sparkline({ data, color = "#3b82f6" }) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <AreaChart data={data}>
        <defs>
          <linearGradient id={`colorUv-${color}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={color} stopOpacity={0.3}/>
            <stop offset="95%" stopColor={color} stopOpacity={0}/>
          </linearGradient>
        </defs>
        <Area 
          type="monotone" 
          dataKey="value" 
          stroke={color} 
          fillOpacity={1} 
          fill={`url(#colorUv-${color})`} 
          isAnimationActive={false} // Отключаем анимацию для микро-графиков, как обсуждали!
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}