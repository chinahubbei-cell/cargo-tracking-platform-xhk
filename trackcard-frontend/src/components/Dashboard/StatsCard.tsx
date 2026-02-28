import React, { useEffect, useState } from 'react';
import { ArrowUpOutlined, ArrowDownOutlined } from '@ant-design/icons';
import {
    AreaChart,
    Area,
    ResponsiveContainer,
} from 'recharts';

interface StatsCardProps {
    title: string;
    value: number | string;
    icon: React.ReactNode;
    trend?: number;
    trendLabel?: string;
    status?: 'blue' | 'green' | 'orange' | 'red';
    suffix?: string;
    sparklineData?: number[];
    onClick?: () => void;
    delay?: number;
}

const StatsCard: React.FC<StatsCardProps> = ({
    title,
    value,
    icon,
    trend,
    trendLabel,
    status = 'blue',
    suffix = '',
    sparklineData,
    onClick,
    delay = 0,
}) => {
    const [displayValue, setDisplayValue] = useState<number | string>(0);
    const [isVisible, setIsVisible] = useState(false);

    useEffect(() => {
        const timer = setTimeout(() => setIsVisible(true), delay);
        return () => clearTimeout(timer);
    }, [delay]);

    useEffect(() => {
        if (!isVisible) return;

        if (typeof value === 'number') {
            const duration = 800;
            const steps = 20;
            const stepValue = value / steps;
            let current = 0;
            const interval = setInterval(() => {
                current += stepValue;
                if (current >= value) {
                    setDisplayValue(value);
                    clearInterval(interval);
                } else {
                    setDisplayValue(Math.floor(current));
                }
            }, duration / steps);
            return () => clearInterval(interval);
        } else {
            setDisplayValue(value);
        }
    }, [value, isVisible]);

    // 生成sparkline数据
    const chartData = sparklineData?.map((v, i) => ({ value: v, idx: i })) ||
        Array.from({ length: 7 }, (_, i) => ({
            value: Math.random() * 50 + 50,
            idx: i
        }));

    const getSparklineColor = () => {
        switch (status) {
            case 'green': return '#10b981';
            case 'orange': return '#f59e0b';
            case 'red': return '#ef4444';
            default: return '#2563eb';
        }
    };

    return (
        <div
            className={`stat-card animate-fade-in`}
            style={{
                opacity: isVisible ? 1 : 0,
                animationDelay: `${delay}ms`
            }}
            onClick={onClick}
        >
            {/* 图标 */}
            <div className={`stat-card-icon ${status}`}>
                {icon}
            </div>

            {/* 数值 */}
            <div className="stat-value animate-count">
                {displayValue}{suffix}
            </div>

            {/* 标签 */}
            <div className="stat-label">{title}</div>

            {/* 迷你图表 */}
            <div className="sparkline">
                <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={chartData}>
                        <defs>
                            <linearGradient id={`gradient-${status}`} x1="0" y1="0" x2="0" y2="1">
                                <stop offset="0%" stopColor={getSparklineColor()} stopOpacity={0.3} />
                                <stop offset="100%" stopColor={getSparklineColor()} stopOpacity={0} />
                            </linearGradient>
                        </defs>
                        <Area
                            type="monotone"
                            dataKey="value"
                            stroke={getSparklineColor()}
                            strokeWidth={2}
                            fill={`url(#gradient-${status})`}
                            dot={false}
                        />
                    </AreaChart>
                </ResponsiveContainer>
            </div>

            {/* 趋势指示器 */}
            {trend !== undefined && (
                <div className={`stat-trend ${trend >= 0 ? 'up' : 'down'}`}>
                    {trend >= 0 ? <ArrowUpOutlined /> : <ArrowDownOutlined />}
                    <span>{Math.abs(trend)}%</span>
                    {trendLabel && <span style={{ fontWeight: 400, marginLeft: 4 }}>{trendLabel}</span>}
                </div>
            )}
        </div>
    );
};

export default StatsCard;
