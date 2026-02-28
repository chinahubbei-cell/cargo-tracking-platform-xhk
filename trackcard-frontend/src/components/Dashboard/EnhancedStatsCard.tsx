import React, { useEffect, useState } from 'react';
import { ArrowUpOutlined, ArrowDownOutlined } from '@ant-design/icons';

interface EnhancedStatsCardProps {
    title: string;
    value: number | string;
    icon: React.ReactNode;
    trend?: number;
    trendLabel?: string;
    status?: 'success' | 'warning' | 'danger' | 'info' | 'default';
    suffix?: string;
    animationDelay?: number;
    onClick?: () => void;
}

const EnhancedStatsCard: React.FC<EnhancedStatsCardProps> = ({
    title,
    value,
    icon,
    trend,
    trendLabel,
    status = 'default',
    suffix = '',
    animationDelay = 0,
    onClick,
}) => {
    const [displayValue, setDisplayValue] = useState<number | string>(0);
    const [isVisible, setIsVisible] = useState(false);

    useEffect(() => {
        const timer = setTimeout(() => {
            setIsVisible(true);
        }, animationDelay);
        return () => clearTimeout(timer);
    }, [animationDelay]);

    useEffect(() => {
        if (!isVisible) return;

        if (typeof value === 'number') {
            const duration = 1000;
            const steps = 30;
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

    const getStatusGradient = () => {
        switch (status) {
            case 'success': return 'from-emerald-500/20 to-emerald-600/10';
            case 'warning': return 'from-amber-500/20 to-amber-600/10';
            case 'danger': return 'from-rose-500/20 to-rose-600/10';
            case 'info': return 'from-cyan-500/20 to-cyan-600/10';
            default: return 'from-blue-500/20 to-blue-600/10';
        }
    };

    const getIconColor = () => {
        switch (status) {
            case 'success': return '#10b981';
            case 'warning': return '#f59e0b';
            case 'danger': return '#ef4444';
            case 'info': return '#06b6d4';
            default: return '#3b82f6';
        }
    };

    const getStatusBorder = () => {
        switch (status) {
            case 'success': return 'hover:border-emerald-500/50';
            case 'warning': return 'hover:border-amber-500/50';
            case 'danger': return 'hover:border-rose-500/50';
            case 'info': return 'hover:border-cyan-500/50';
            default: return 'hover:border-blue-500/50';
        }
    };

    return (
        <div
            className={`
                glass-card cursor-pointer
                transform transition-all duration-500 ease-out
                ${isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4'}
                ${getStatusBorder()}
                ${status === 'danger' ? 'pulse-danger' : ''}
            `}
            onClick={onClick}
            style={{ animationDelay: `${animationDelay}ms` }}
        >
            <div className="flex items-start justify-between mb-4">
                <div className={`
                    w-12 h-12 rounded-xl flex items-center justify-center
                    bg-gradient-to-br ${getStatusGradient()}
                    transition-transform duration-300 hover:scale-110
                `}>
                    <span style={{ color: getIconColor(), fontSize: '24px' }}>{icon}</span>
                </div>
                {trend !== undefined && (
                    <div className={`
                        flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold
                        ${trend >= 0
                            ? 'bg-emerald-500/20 text-emerald-400'
                            : 'bg-rose-500/20 text-rose-400'
                        }
                    `}>
                        {trend >= 0
                            ? <ArrowUpOutlined style={{ fontSize: '10px' }} />
                            : <ArrowDownOutlined style={{ fontSize: '10px' }} />
                        }
                        <span>{Math.abs(trend)}%</span>
                    </div>
                )}
            </div>

            <div className="space-y-1">
                <h3 className="stat-value m-0 flex items-baseline gap-1">
                    <span className="animate-count">{displayValue}</span>
                    {suffix && <span className="text-lg text-slate-400">{suffix}</span>}
                </h3>
                <p className="stat-label m-0">{title}</p>
                {trendLabel && (
                    <p className="text-xs text-slate-500 m-0">{trendLabel}</p>
                )}
            </div>

            {/* 装饰性渐变线 */}
            <div
                className="absolute bottom-0 left-0 right-0 h-1 rounded-b-2xl opacity-60"
                style={{
                    background: `linear-gradient(90deg, transparent, ${getIconColor()}, transparent)`
                }}
            />
        </div>
    );
};

export default EnhancedStatsCard;
