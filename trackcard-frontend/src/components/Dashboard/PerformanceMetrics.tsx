import React from 'react';

interface MetricItem {
    label: string;
    value: number;
    max: number;
    color: string;
    unit?: string;
}

interface PerformanceMetricsProps {
    metrics?: MetricItem[];
}

// 环形进度条组件
const CircularProgress: React.FC<{
    value: number;
    max: number;
    color: string;
    size?: number;
    strokeWidth?: number;
    children?: React.ReactNode;
}> = ({ value, max, color, size = 80, strokeWidth = 6, children }) => {
    const radius = (size - strokeWidth) / 2;
    const circumference = radius * 2 * Math.PI;
    const percent = Math.min(value / max, 1);
    const strokeDashoffset = circumference - percent * circumference;

    return (
        <div className="relative" style={{ width: size, height: size }}>
            <svg width={size} height={size} className="transform -rotate-90">
                {/* 背景环 */}
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    fill="transparent"
                    stroke="rgba(71, 85, 105, 0.3)"
                    strokeWidth={strokeWidth}
                />
                {/* 进度环 */}
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    fill="transparent"
                    stroke={color}
                    strokeWidth={strokeWidth}
                    strokeLinecap="round"
                    strokeDasharray={circumference}
                    strokeDashoffset={strokeDashoffset}
                    className="progress-ring-circle"
                    style={{
                        filter: `drop-shadow(0 0 4px ${color}80)`,
                    }}
                />
            </svg>
            <div className="absolute inset-0 flex items-center justify-center">
                {children}
            </div>
        </div>
    );
};

const defaultMetrics: MetricItem[] = [
    { label: '准时率', value: 98.5, max: 100, color: '#10b981', unit: '%' },
    { label: '设备在线率', value: 94.2, max: 100, color: '#3b82f6', unit: '%' },
    { label: '客户满意度', value: 4.8, max: 5, color: '#8b5cf6', unit: '/5' },
    { label: '运输效率', value: 87, max: 100, color: '#06b6d4', unit: '%' },
];

const PerformanceMetrics: React.FC<PerformanceMetricsProps> = ({
    metrics = defaultMetrics
}) => {
    return (
        <div className="glass-card h-full">
            <h3 className="text-lg font-semibold text-slate-100 mb-6">核心绩效指标</h3>

            <div className="grid grid-cols-2 gap-6">
                {metrics.map((metric, index) => (
                    <div
                        key={metric.label}
                        className="flex flex-col items-center animate-fade-in"
                        style={{ animationDelay: `${index * 100}ms` }}
                    >
                        <CircularProgress
                            value={metric.value}
                            max={metric.max}
                            color={metric.color}
                            size={72}
                            strokeWidth={5}
                        >
                            <span className="text-lg font-bold text-white">
                                {metric.value.toFixed(metric.max <= 10 ? 1 : 0)}
                            </span>
                        </CircularProgress>
                        <span className="text-xs text-slate-400 mt-2 text-center">
                            {metric.label}
                        </span>
                    </div>
                ))}
            </div>

            {/* 底部说明 */}
            <div className="mt-6 pt-4 border-t border-slate-700/50">
                <div className="flex items-center justify-between text-xs">
                    <span className="text-slate-500">数据更新时间</span>
                    <span className="text-slate-400">
                        {new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}
                    </span>
                </div>
            </div>
        </div>
    );
};

export default PerformanceMetrics;
