import React from 'react';
import {
    LineChart,
    Line,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    ResponsiveContainer,
    Legend,
    AreaChart,
    Area,
} from 'recharts';

interface TrendChartProps {
    title?: string;
    subtitle?: string;
    data?: Array<{
        date: string;
        shipments: number;
        delivered: number;
    }>;
    type?: 'line' | 'area';
}

// 模拟数据
const generateMockData = () => {
    const months = ['1月', '2月', '3月', '4月', '5月', '6月'];
    return months.map((month) => ({
        date: month,
        shipments: Math.floor(Math.random() * 500) + 300,
        delivered: Math.floor(Math.random() * 450) + 250,
    }));
};

const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
        return (
            <div style={{
                background: 'white',
                border: '1px solid var(--border-light)',
                borderRadius: 8,
                padding: '12px 16px',
                boxShadow: 'var(--shadow-lg)',
            }}>
                <p style={{
                    margin: '0 0 8px 0',
                    fontWeight: 600,
                    color: 'var(--text-primary)',
                    fontSize: 13
                }}>
                    {label}
                </p>
                {payload.map((entry: any, index: number) => (
                    <div key={index} style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        gap: 16,
                        fontSize: 12,
                        color: 'var(--text-secondary)',
                        marginTop: 4
                    }}>
                        <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                            <span style={{
                                width: 8,
                                height: 8,
                                borderRadius: '50%',
                                background: entry.color
                            }} />
                            {entry.name}
                        </span>
                        <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
                            {entry.value}
                        </span>
                    </div>
                ))}
            </div>
        );
    }
    return null;
};

const TrendChart: React.FC<TrendChartProps> = ({
    title = '月度运输趋势',
    subtitle = 'Monthly Shipment Volume',
    data,
    type = 'area',
}) => {
    const chartData = data || generateMockData();

    return (
        <div className="card" style={{ height: '100%' }}>
            <div className="card-header" style={{ marginBottom: 20 }}>
                <div>
                    <h3 className="card-title">{title}</h3>
                    <p className="card-subtitle">{subtitle}</p>
                </div>
            </div>

            <div style={{ height: 260 }}>
                <ResponsiveContainer width="100%" height="100%">
                    {type === 'area' ? (
                        <AreaChart data={chartData} margin={{ top: 5, right: 20, left: -20, bottom: 5 }}>
                            <defs>
                                <linearGradient id="colorShipments" x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="5%" stopColor="#2563eb" stopOpacity={0.15} />
                                    <stop offset="95%" stopColor="#2563eb" stopOpacity={0} />
                                </linearGradient>
                                <linearGradient id="colorDelivered" x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="5%" stopColor="#10b981" stopOpacity={0.15} />
                                    <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                                </linearGradient>
                            </defs>
                            <CartesianGrid strokeDasharray="3 3" stroke="var(--border-light)" vertical={false} />
                            <XAxis
                                dataKey="date"
                                stroke="var(--text-muted)"
                                fontSize={12}
                                tickLine={false}
                                axisLine={{ stroke: 'var(--border-light)' }}
                            />
                            <YAxis
                                stroke="var(--text-muted)"
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
                                    <span style={{ color: 'var(--text-secondary)', fontSize: 12 }}>{value}</span>
                                )}
                            />
                            <Area
                                type="monotone"
                                dataKey="shipments"
                                name="发运量"
                                stroke="#2563eb"
                                strokeWidth={2}
                                fill="url(#colorShipments)"
                                dot={false}
                                activeDot={{ r: 5, strokeWidth: 2, stroke: 'white' }}
                            />
                            <Area
                                type="monotone"
                                dataKey="delivered"
                                name="送达量"
                                stroke="#10b981"
                                strokeWidth={2}
                                fill="url(#colorDelivered)"
                                dot={false}
                                activeDot={{ r: 5, strokeWidth: 2, stroke: 'white' }}
                            />
                        </AreaChart>
                    ) : (
                        <LineChart data={chartData} margin={{ top: 5, right: 20, left: -20, bottom: 5 }}>
                            <CartesianGrid strokeDasharray="3 3" stroke="var(--border-light)" vertical={false} />
                            <XAxis
                                dataKey="date"
                                stroke="var(--text-muted)"
                                fontSize={12}
                                tickLine={false}
                                axisLine={{ stroke: 'var(--border-light)' }}
                            />
                            <YAxis
                                stroke="var(--text-muted)"
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
                            />
                            <Line
                                type="monotone"
                                dataKey="shipments"
                                name="发运量"
                                stroke="#2563eb"
                                strokeWidth={2}
                                dot={false}
                                activeDot={{ r: 5, strokeWidth: 2, stroke: 'white' }}
                            />
                            <Line
                                type="monotone"
                                dataKey="delivered"
                                name="送达量"
                                stroke="#10b981"
                                strokeWidth={2}
                                dot={false}
                                activeDot={{ r: 5, strokeWidth: 2, stroke: 'white' }}
                            />
                        </LineChart>
                    )}
                </ResponsiveContainer>
            </div>
        </div>
    );
};

export default TrendChart;
