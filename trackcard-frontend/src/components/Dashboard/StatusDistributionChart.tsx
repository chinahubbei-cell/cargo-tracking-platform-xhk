import React from 'react';
import {
    PieChart,
    Pie,
    Cell,
    ResponsiveContainer,
    Tooltip,
} from 'recharts';

interface StatusDistributionProps {
    title: string;
    data: Array<{
        name: string;
        value: number;
        color: string;
    }>;
    centerLabel?: string;
    centerValue?: string | number;
}

const RADIAN = Math.PI / 180;

const renderCustomizedLabel = ({
    cx,
    cy,
    midAngle,
    innerRadius,
    outerRadius,
    percent,
}: any) => {
    const radius = innerRadius + (outerRadius - innerRadius) * 0.5;
    const x = cx + radius * Math.cos(-midAngle * RADIAN);
    const y = cy + radius * Math.sin(-midAngle * RADIAN);

    if (percent < 0.05) return null;

    return (
        <text
            x={x}
            y={y}
            fill="white"
            textAnchor="middle"
            dominantBaseline="central"
            fontSize={12}
            fontWeight={600}
        >
            {`${(percent * 100).toFixed(0)}%`}
        </text>
    );
};

const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
        const data = payload[0].payload;
        return (
            <div className="glass-card !p-3 !rounded-lg">
                <div className="flex items-center gap-2">
                    <span
                        className="w-3 h-3 rounded-full"
                        style={{ backgroundColor: data.color }}
                    />
                    <span className="text-slate-300 font-medium">{data.name}</span>
                </div>
                <p className="text-white font-bold text-lg m-0 mt-1">{data.value}</p>
            </div>
        );
    }
    return null;
};

const StatusDistributionChart: React.FC<StatusDistributionProps> = ({
    title,
    data,
    centerLabel,
    centerValue,
}) => {
    // 计算总数（供未来使用）

    return (
        <div className="glass-card h-full">
            <h3 className="text-lg font-semibold text-slate-100 mb-4">{title}</h3>

            <div className="relative" style={{ height: '200px' }}>
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
                            labelLine={false}
                            label={renderCustomizedLabel}
                            animationBegin={0}
                            animationDuration={1000}
                        >
                            {data.map((entry, index) => (
                                <Cell
                                    key={`cell-${index}`}
                                    fill={entry.color}
                                    stroke="transparent"
                                />
                            ))}
                        </Pie>
                        <Tooltip content={<CustomTooltip />} />
                    </PieChart>
                </ResponsiveContainer>

                {/* 中心文本 */}
                {(centerLabel || centerValue) && (
                    <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                        {centerValue !== undefined && (
                            <span className="text-2xl font-bold text-white">{centerValue}</span>
                        )}
                        {centerLabel && (
                            <span className="text-xs text-slate-400">{centerLabel}</span>
                        )}
                    </div>
                )}
            </div>

            {/* 图例 */}
            <div className="grid grid-cols-2 gap-2 mt-4">
                {data.map((item, index) => (
                    <div key={index} className="flex items-center gap-2">
                        <span
                            className="w-3 h-3 rounded-full flex-shrink-0"
                            style={{ backgroundColor: item.color }}
                        />
                        <span className="text-xs text-slate-400 truncate">{item.name}</span>
                        <span className="text-xs font-semibold text-slate-200 ml-auto">
                            {item.value}
                        </span>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default StatusDistributionChart;
