import React, { useState } from 'react';
import {
    AreaChart,
    Area,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    ResponsiveContainer,
    Legend,
} from 'recharts';

interface ShipmentTrendChartProps {
    data?: Array<{
        date: string;
        dispatched: number;
        delivered: number;
        pending: number;
    }>;
}

// 模拟最近7天数据
const generateMockData = () => {
    const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
    return days.map((day) => ({
        date: day,
        dispatched: Math.floor(Math.random() * 50) + 20,
        delivered: Math.floor(Math.random() * 45) + 15,
        pending: Math.floor(Math.random() * 20) + 5,
    }));
};

const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
        return (
            <div className="glass-card !p-3 !rounded-lg" style={{ minWidth: '160px' }}>
                <p className="text-slate-300 font-semibold mb-2 text-sm">{label}</p>
                {payload.map((entry: any, index: number) => (
                    <div key={index} className="flex items-center justify-between gap-4 text-xs">
                        <span className="flex items-center gap-2">
                            <span
                                className="w-2 h-2 rounded-full"
                                style={{ backgroundColor: entry.color }}
                            />
                            {entry.name}
                        </span>
                        <span className="font-semibold text-white">{entry.value}</span>
                    </div>
                ))}
            </div>
        );
    }
    return null;
};

const ShipmentTrendChart: React.FC<ShipmentTrendChartProps> = ({ data }) => {
    const [timeRange, setTimeRange] = useState<'week' | 'month' | 'year'>('week');
    const chartData = data || generateMockData();

    const timeRangeOptions = [
        { key: 'week', label: '本周' },
        { key: 'month', label: '本月' },
        { key: 'year', label: '今年' },
    ];

    return (
        <div className="glass-card h-full">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-slate-100 m-0">运输趋势</h3>
                <div className="flex gap-1 p-1 rounded-lg bg-slate-800/50">
                    {timeRangeOptions.map((option) => (
                        <button
                            key={option.key}
                            onClick={() => setTimeRange(option.key as any)}
                            className={`
                                px-3 py-1.5 text-xs font-medium rounded-md transition-all duration-200
                                ${timeRange === option.key
                                    ? 'bg-blue-600 text-white shadow-lg'
                                    : 'text-slate-400 hover:text-white hover:bg-slate-700/50'
                                }
                            `}
                        >
                            {option.label}
                        </button>
                    ))}
                </div>
            </div>

            <div style={{ height: '280px' }}>
                <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                        <defs>
                            <linearGradient id="dispatchedGradient" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.4} />
                                <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                            </linearGradient>
                            <linearGradient id="deliveredGradient" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#10b981" stopOpacity={0.4} />
                                <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                            </linearGradient>
                            <linearGradient id="pendingGradient" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.4} />
                                <stop offset="95%" stopColor="#f59e0b" stopOpacity={0} />
                            </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" stroke="rgba(71, 85, 105, 0.3)" />
                        <XAxis
                            dataKey="date"
                            stroke="#64748b"
                            fontSize={12}
                            tickLine={false}
                            axisLine={false}
                        />
                        <YAxis
                            stroke="#64748b"
                            fontSize={12}
                            tickLine={false}
                            axisLine={false}
                        />
                        <Tooltip content={<CustomTooltip />} />
                        <Legend
                            verticalAlign="top"
                            height={36}
                            iconType="circle"
                            iconSize={8}
                            formatter={(value) => (
                                <span className="text-slate-400 text-xs">{value}</span>
                            )}
                        />
                        <Area
                            type="monotone"
                            dataKey="dispatched"
                            name="已发运"
                            stroke="#3b82f6"
                            strokeWidth={2}
                            fill="url(#dispatchedGradient)"
                            dot={false}
                            activeDot={{ r: 6, strokeWidth: 2, stroke: '#fff' }}
                        />
                        <Area
                            type="monotone"
                            dataKey="delivered"
                            name="已送达"
                            stroke="#10b981"
                            strokeWidth={2}
                            fill="url(#deliveredGradient)"
                            dot={false}
                            activeDot={{ r: 6, strokeWidth: 2, stroke: '#fff' }}
                        />
                        <Area
                            type="monotone"
                            dataKey="pending"
                            name="待处理"
                            stroke="#f59e0b"
                            strokeWidth={2}
                            fill="url(#pendingGradient)"
                            dot={false}
                            activeDot={{ r: 6, strokeWidth: 2, stroke: '#fff' }}
                        />
                    </AreaChart>
                </ResponsiveContainer>
            </div>
        </div>
    );
};

export default ShipmentTrendChart;
