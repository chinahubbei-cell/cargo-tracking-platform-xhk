import React from 'react';
import {
    PieChart,
    Pie,
    Cell,
    ResponsiveContainer,
    Tooltip,
} from 'recharts';

interface DonutChartProps {
    title?: string;
    subtitle?: string;
    data: Array<{
        name: string;
        value: number;
        color: string;
    }>;
    centerValue?: string | number;
    centerLabel?: string;
}

const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
        const data = payload[0].payload;
        return (
            <div style={{
                background: 'white',
                border: '1px solid var(--border-light)',
                borderRadius: 8,
                padding: '10px 14px',
                boxShadow: 'var(--shadow-lg)',
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{
                        width: 10,
                        height: 10,
                        borderRadius: '50%',
                        background: data.color
                    }} />
                    <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{data.name}</span>
                </div>
                <p style={{
                    margin: '6px 0 0 0',
                    fontSize: 18,
                    fontWeight: 600,
                    color: 'var(--text-primary)'
                }}>
                    {data.value}
                </p>
            </div>
        );
    }
    return null;
};

const DonutChart: React.FC<DonutChartProps> = ({
    title = '运单状态分布',
    subtitle = 'Shipment Status Distribution',
    data,
    centerValue,
    centerLabel,
}) => {
    const total = data.reduce((sum, item) => sum + item.value, 0);

    return (
        <div className="card" style={{ height: '100%' }}>
            <div className="card-header">
                <div>
                    <h3 className="card-title">{title}</h3>
                    <p className="card-subtitle">{subtitle}</p>
                </div>
            </div>

            <div style={{ position: 'relative', height: 180 }}>
                <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                        <Pie
                            data={data}
                            cx="50%"
                            cy="50%"
                            innerRadius={55}
                            outerRadius={80}
                            paddingAngle={3}
                            dataKey="value"
                            animationBegin={0}
                            animationDuration={800}
                        >
                            {data.map((entry, index) => (
                                <Cell
                                    key={`cell-${index}`}
                                    fill={entry.color}
                                    stroke="white"
                                    strokeWidth={2}
                                />
                            ))}
                        </Pie>
                        <Tooltip content={<CustomTooltip />} />
                    </PieChart>
                </ResponsiveContainer>

                {/* 中心文本 */}
                <div style={{
                    position: 'absolute',
                    top: '50%',
                    left: '50%',
                    transform: 'translate(-50%, -50%)',
                    textAlign: 'center',
                    pointerEvents: 'none',
                }}>
                    <div style={{
                        fontSize: 24,
                        fontWeight: 700,
                        color: 'var(--text-primary)',
                        lineHeight: 1.2
                    }}>
                        {centerValue ?? total}
                    </div>
                    <div style={{
                        fontSize: 11,
                        color: 'var(--text-muted)',
                        textTransform: 'uppercase',
                        letterSpacing: '0.05em'
                    }}>
                        {centerLabel ?? '总计'}
                    </div>
                </div>
            </div>

            {/* 图例 */}
            <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(2, 1fr)',
                gap: 10,
                marginTop: 16
            }}>
                {data.map((item, index) => (
                    <div
                        key={index}
                        style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 8,
                            fontSize: 13
                        }}
                    >
                        <span style={{
                            width: 10,
                            height: 10,
                            borderRadius: '50%',
                            background: item.color,
                            flexShrink: 0
                        }} />
                        <span style={{ color: 'var(--text-tertiary)', flex: 1 }} className="truncate">
                            {item.name}
                        </span>
                        <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
                            {item.value}
                        </span>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default DonutChart;
